// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vztemplate "github.com/verrazzano/verrazzano/application-operator/controllers/template"
	"github.com/verrazzano/verrazzano/application-operator/internal/metrics"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/metricsutils"
	constants2 "github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	k8scontroller "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"
)

// Reconciler reconciles a metrics workload object
type Reconciler struct {
	k8sclient.Client
	Log     *zap.SugaredLogger
	Scheme  *k8sruntime.Scheme
	Scraper string
}

const controllerName = "metricsbinding"

// SetupWithManager creates controller for the MetricsBinding
func (r *Reconciler) SetupWithManager(mgr k8scontroller.Manager) error {
	return k8scontroller.NewControllerManagedBy(mgr).For(&vzapi.MetricsBinding{}).Complete(r)
}

// Reconcile reconciles a workload to keep the Prometheus ConfigMap scrape job configuration up to date.
// No kubebuilder annotations are used as the application RBAC for the application operator is now manually managed.
func (r *Reconciler) Reconcile(ctx context.Context, req k8scontroller.Request) (k8scontroller.Result, error) {

	// We do not want any resource to get reconciled if it is in namespace kube-system
	// This is due to a bug found in OKE, it should not affect functionality of any vz operators
	// If this is the case then return success
	if req.Namespace == vzconst.KubeSystem {
		log := zap.S().With(vzlogInit.FieldResourceNamespace, req.Namespace, vzlogInit.FieldResourceName, req.Name, vzlogInit.FieldController, controllerName)
		log.Infof("Metrics binding resource %v should not be reconciled in kube-system namespace, ignoring", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	metricsBinding := vzapi.MetricsBinding{}
	if err := r.Client.Get(context.TODO(), req.NamespacedName, &metricsBinding); err != nil {
		return clusters.IgnoreNotFoundWithLog(err, zap.S())
	}
	log, err := clusters.GetResourceLogger("metricsbinding", req.NamespacedName, &metricsBinding)
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for metrics binding resource: %v", err)
		return clusters.NewRequeueWithDelay(), nil
	}
	log.Oncef("Reconciling metrics binding resource %v, generation %v", req.NamespacedName, metricsBinding.Generation)

	res, err := r.doReconcile(ctx, metricsBinding, log)
	if clusters.ShouldRequeue(res) {
		return res, nil
	}
	if err != nil {
		return clusters.NewRequeueWithDelay(), err
	}
	log.Oncef("Finished reconciling metrics binding %v", req.NamespacedName)

	return k8scontroller.Result{}, nil
}

// doReconcile performs the reconciliation operations for the ingress trait
func (r *Reconciler) doReconcile(ctx context.Context, metricsBinding vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (k8scontroller.Result, error) {
	// Reconcile based on the status of the deletion timestamp
	if metricsBinding.GetDeletionTimestamp().IsZero() {
		return r.reconcileBindingCreateOrUpdate(ctx, &metricsBinding, log)
	}
	return r.reconcileBindingDelete(ctx, &metricsBinding, log)
}

// reconcileBindingDelete completes the reconcile process for an object that is being deleted
func (r *Reconciler) reconcileBindingDelete(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (k8scontroller.Result, error) {
	log.Debugw("Reconcile for deleted object", "resource", metricsBinding.GetName())

	// Mutate the scrape config by deleting the entry
	if err := r.mutatePrometheusScrapeConfig(ctx, metricsBinding, r.deleteScrapeConfig, log); err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Remove the finalizer if deletion was successful
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, metricsBinding, func() error {
		controllerutil.RemoveFinalizer(metricsBinding, finalizerName)
		return nil
	})
	if err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	return k8scontroller.Result{}, nil
}

// reconcileBindingCreateOrUpdate completes the reconcile process for an object that is being created or updated
func (r *Reconciler) reconcileBindingCreateOrUpdate(ctx context.Context, metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (k8scontroller.Result, error) {
	log.Debugw("Reconcile for created or updated object", "resource", metricsBinding.GetName())

	// Requeue with a delay to account for situations where the scrape config
	// has changed but without the MetricsBinding changing.
	var seconds = rand.IntnRange(45, 90)
	var requeueDuration = time.Duration(seconds) * time.Second

	// Handle the case where the workload uses the default metrics template - in this case, we will
	// delete the metrics binding if processing succeeds, since this is a one-time conversion of
	// legacy apps using default metrics template, to ServiceMonitor. If it's not using VMI config map,
	// we treat it like custom metrics setup
	if isLegacyDefaultMetricsBinding(metricsBinding) {
		if err := metrics.HandleDefaultMetricsTemplate(ctx, r.Client, metricsBinding, log); err != nil {
			return k8scontroller.Result{Requeue: true}, err
		}
		if err := r.deleteMetricsBinding(metricsBinding, log); err != nil {
			return k8scontroller.Result{Requeue: true}, err
		}
		// Requeue with a delay to account for situations where the scrape config
		// has changed but without the MetricsBinding changing.
		return reconcile.Result{Requeue: true, RequeueAfter: requeueDuration}, nil
	}

	// Update the MetricsBinding to add workload as owner ref
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, metricsBinding, func() error {
		return r.updateMetricsBinding(metricsBinding, log)
	})
	if err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Handle the case where the workloaded uses a custom metrics template
	if err := metrics.HandleCustomMetricsTemplate(ctx, r.Client, metricsBinding, log.GetZapLogger()); err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Mutate the scrape config by adding or updating the job - NOT NEEDED - will be handled by HandleCustomMetricsTemplate?
	// if err := r.mutatePrometheusScrapeConfig(ctx, metricsBinding, r.createOrUpdateScrapeConfig, log); err != nil {
	// 	return k8scontroller.Result{Requeue: true}, err
	// }

	// Requeue with a delay to account for situations where the scrape config
	// has changed but without the MetricsBinding changing.
	return reconcile.Result{Requeue: true, RequeueAfter: requeueDuration}, nil
}

func (r *Reconciler) deleteMetricsBinding(metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	return r.Delete(context.Background(), metricsBinding)
}

func (r *Reconciler) updateMetricsBinding(metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	// Add the finalizer
	controllerutil.AddFinalizer(metricsBinding, finalizerName)

	// Retrieve the workload object from the MetricsBinding
	workloadObject, err := r.getWorkloadObject(metricsBinding)
	if err != nil {
		return log.ErrorfNewErr("Failed to get the Workload from the MetricsBinding %s: %v", metricsBinding.Spec.Workload.Name, err)
	}

	// Return error if UID is not found
	if len(workloadObject.GetUID()) == 0 {
		err = fmt.Errorf("could not get UID from workload resource: %s, %s", workloadObject.GetKind(), workloadObject.GetName())
		return log.ErrorfNewErr("Failed to find UID for workload %s: %v", workloadObject.GetName(), err)
	}

	// Set the owner reference for the MetricsBinding so that it gets deleted with the workload
	trueValue := true
	metricsBinding.SetOwnerReferences([]k8smetav1.OwnerReference{
		{
			Name:               workloadObject.GetName(),
			APIVersion:         workloadObject.GetAPIVersion(),
			Kind:               workloadObject.GetKind(),
			UID:                workloadObject.GetUID(),
			Controller:         &trueValue,
			BlockOwnerDeletion: &trueValue,
		},
	})
	return nil
}

// mutatePrometheusScrapeConfig takes the resource and a mutate function that determines the mutations of the scrape config
// mutations are dependant upon the status of the deletion timestamp
func (r *Reconciler) mutatePrometheusScrapeConfig(ctx context.Context, metricsBinding *vzapi.MetricsBinding, mutateFn func(metricsBinding *vzapi.MetricsBinding, configData *gabs.Container, log vzlog.VerrazzanoLogger) (*gabs.Container, error), log vzlog.VerrazzanoLogger) error {
	log.Debugw("Mutating the Prometheus Scrape Config", "resource", metricsBinding.GetName())

	var configMap = r.getPromConfigMap(metricsBinding)
	if configMap != nil {
		return r.mutatePrometheusConfigMap(ctx, metricsBinding, configMap, mutateFn, log)
	}
	// Get data from the config secret
	secret, key := r.getPromConfigSecret(metricsBinding)
	return r.mutatePrometheusConfigSecret(ctx, metricsBinding, secret, key, mutateFn, log)
}

// deleteScrapeConfig is a mutation function that deletes the scrape config data from the Prometheus ConfigMap
func (r *Reconciler) deleteScrapeConfig(metricsBinding *vzapi.MetricsBinding, configData *gabs.Container, log vzlog.VerrazzanoLogger) (*gabs.Container, error) {
	log.Debugw("Scrape Config is being deleted from the Prometheus Config", "resource", metricsBinding.GetName())

	// Verify the Owner Reference exists
	if len(metricsBinding.OwnerReferences) < 1 {
		return nil, fmt.Errorf("No Owner Reference found in the MetricsBinding: %s", metricsBinding.GetName())
	}

	// Delete scrape config with job name matching resource
	// parse the scrape config so we can manipulate it
	jobNameToDelete := createJobName(metricsBinding)
	updatedScrapeConfigs, err := metricsutils.EditScrapeJob(configData, jobNameToDelete, nil)
	if err != nil {
		return nil, err
	}
	return updatedScrapeConfigs, nil
}

// createOrUpdateScrapeConfig is a mutation function that creates or updates the scrape config data within the given Prometheus ConfigMap
func (r *Reconciler) createOrUpdateScrapeConfig(metricsBinding *vzapi.MetricsBinding, configMap *k8scorev1.ConfigMap, log vzlog.VerrazzanoLogger) error {
	log.Debugw("Scrape Config is being created or update in the Prometheus config", "resource", metricsBinding.GetName())

	// Get the MetricsTemplate from the MetricsBinding
	template, err := metrics.GetMetricsTemplate(context.Background(), r.Client, metricsBinding, log.GetZapLogger())
	if err != nil {
		return err
	}

	// Get data from the configmap
	promConfig, err := getConfigData(configMap)
	if err != nil {
		return log.ErrorfNewErr("Failed to get Prometheus config map %s: %v", configMap.GetName(), err)
	}

	// Get the namespace for the metrics binding
	workloadNamespace := k8scorev1.Namespace{}
	err = r.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: metricsBinding.GetNamespace()}, &workloadNamespace)
	if err != nil {
		return log.ErrorfNewErr("Failed to get metrics binding namespace %s: %v", metricsBinding.GetName(), err)
	}

	// Create Unstructured Namespace
	workloadNamespaceUnstructuredMap, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(&workloadNamespace)
	if err != nil {
		return log.ErrorfNewErr("Failed to get the unstructured for namespace %s: %v", workloadNamespace.GetName(), err)
	}
	workloadNamespaceUnstructured := unstructured.Unstructured{Object: workloadNamespaceUnstructuredMap}

	// Get the workload object for the template processor
	workloadObject, err := r.getWorkloadObject(metricsBinding)
	if err != nil {
		return log.ErrorfNewErr("Failed to get the workload object for metrics binding %s: %v", metricsBinding.GetName(), err)
	}

	// Organize inputs for template processor
	templateInputs := map[string]interface{}{
		"workload":  workloadObject.Object,
		"namespace": workloadNamespaceUnstructured.Object,
	}

	// Get scrape config from the template processor and process the template inputs
	templateProcessor := vztemplate.NewProcessor(r.Client, template.Spec.PrometheusConfig.ScrapeConfigTemplate)
	scrapeConfigString, err := templateProcessor.Process(templateInputs)
	if err != nil {
		return log.ErrorfNewErr("Failed to process metrics template %s: %v", template.GetName(), err)
	}

	// Prepend job name to the scrape config
	createdJobName := createJobName(metricsBinding)
	scrapeConfigString = formatJobName(createdJobName) + scrapeConfigString

	// Format scrape config into readable container
	configYaml, err := yaml.YAMLToJSON([]byte(scrapeConfigString))
	if err != nil {
		return log.ErrorfNewErr("Failed to convert scrape config YAML to JSON: %v", err)
	}
	newScrapeConfig, err := gabs.ParseJSON(configYaml)
	if err != nil {
		return log.ErrorfNewErr("Failed to convert scrape config JSON to container: %v", err)
	}

	// Create or Update scrape config with job name matching resource
	existingUpdated := false
	scrapeConfigs := promConfig.Search(prometheusScrapeConfigsLabel).Children()
	for index, scrapeConfig := range scrapeConfigs {
		existingJobName := scrapeConfig.Search(vzconst.PrometheusJobNameKey).Data()
		if existingJobName == createdJobName {
			// Remove and recreate scrape config
			err = promConfig.ArrayRemoveP(index, prometheusScrapeConfigsLabel)
			if err != nil {
				return log.ErrorfNewErr("Failed to remove scrape config: %v", err)
			}
			err = promConfig.ArrayAppendP(newScrapeConfig.Data(), prometheusScrapeConfigsLabel)
			if err != nil {
				return log.ErrorfNewErr("Failed to append scrape config: %v", err)
			}
			existingUpdated = true
			break
		}
	}
	if !existingUpdated {
		err = promConfig.ArrayAppendP(newScrapeConfig.Data(), prometheusScrapeConfigsLabel)
		if err != nil {
			return log.ErrorfNewErr("Failed to append scrape config: %v", err)
		}
	}

	// Repopulate the ConfigMap data
	newPromConfigData, err := yaml.JSONToYAML(promConfig.Bytes())
	if err != nil {
		return log.ErrorfNewErr("Failed to convert scrape config JSON to YAML: %v", err)
	}
	configMap.Data[prometheusConfigKey] = string(newPromConfigData)
	return nil
}

// getPromConfigMap returns the Prometheus ConfigMap given in the MetricsBinding
func (r *Reconciler) getPromConfigMap(metricsBinding *vzapi.MetricsBinding) *k8scorev1.ConfigMap {
	targetConfigMap := metricsBinding.Spec.PrometheusConfigMap
	if targetConfigMap.Name == "" {
		return nil
	}
	return &k8scorev1.ConfigMap{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       configMapKind,
			APIVersion: k8sV1APIVersion,
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      targetConfigMap.Name,
			Namespace: targetConfigMap.Namespace,
		},
	}
}

// getPromConfigSecret returns the Prometheus Config Secret given in the MetricsBinding, along with the key
func (r *Reconciler) getPromConfigSecret(metricsBinding *vzapi.MetricsBinding) (*k8scorev1.Secret, string) {
	targetSecret := metricsBinding.Spec.PrometheusConfigSecret
	if targetSecret.Name == "" {
		return nil, ""
	}
	return &k8scorev1.Secret{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       constants2.SecretKind,
			APIVersion: k8sV1APIVersion,
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      targetSecret.Name,
			Namespace: targetSecret.Namespace,
		},
	}, targetSecret.Key
}

// getWorkloadObject returns the workload object based on the definition in the MetricsBinding
func (r *Reconciler) getWorkloadObject(metricsBinding *vzapi.MetricsBinding) (*unstructured.Unstructured, error) {
	// Retrieve the owner from the workload field of the MetricsBinding
	owner := metricsBinding.Spec.Workload
	workloadObject := unstructured.Unstructured{}
	workloadObject.SetKind(owner.TypeMeta.Kind)
	workloadObject.SetAPIVersion(owner.TypeMeta.APIVersion)
	workloadName := types.NamespacedName{Namespace: metricsBinding.GetNamespace(), Name: owner.Name}
	err := r.Client.Get(context.Background(), workloadName, &workloadObject)
	if err != nil {
		return nil, err
	}
	return &workloadObject, nil
}

func (r *Reconciler) mutatePrometheusConfigMap(ctx context.Context, metricsBinding *vzapi.MetricsBinding, configMap *k8scorev1.ConfigMap, mutateFn func(metricsBinding *vzapi.MetricsBinding, configData *gabs.Container, log vzlog.VerrazzanoLogger) (*gabs.Container, error), log vzlog.VerrazzanoLogger) error {
	log.Debugw("Prometheus target ConfigMap is being altered", "resource", configMap.GetName())
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		// Get data from the configmap
		promConfig, err := getConfigData(configMap)
		if err != nil {
			return err
		}
		scrapeConfigs := promConfig.Search(prometheusScrapeConfigsLabel)
		updatedScrapeConfigs, err := mutateFn(metricsBinding, scrapeConfigs, log)
		if err != nil {
			return err
		}
		promConfig.Set(updatedScrapeConfigs, prometheusScrapeConfigsLabel)
		// scrape configs would have been edited in-place, in the promConfig Container, so serialize
		// the whole thing for the new data.
		newPromConfigData, err := yaml.JSONToYAML(promConfig.Bytes())
		if err != nil {
			return log.ErrorfNewErr("Failed to convert scrape config JSON to YAML: %v", err)
		}
		configMap.Data[prometheusConfigKey] = string(newPromConfigData)
		return nil
	})
	return err
}

func (r *Reconciler) mutatePrometheusConfigSecret(ctx context.Context, metricsBinding *vzapi.MetricsBinding, secret *k8scorev1.Secret, key string, mutateFn func(metricsBinding *vzapi.MetricsBinding, configData *gabs.Container, log vzlog.VerrazzanoLogger) (*gabs.Container, error), log vzlog.VerrazzanoLogger) error {
	log.Debugw("Prometheus target config Secret is being altered", "resource", secret.GetName())
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		promConfig, err := getConfigDataFromSecret(secret, key)
		if err != nil {
			return err
		}
		updatedScrapeConfigs, err := mutateFn(metricsBinding, promConfig, log)
		if err != nil {
			return err
		}
		newPromConfigData, err := yaml.JSONToYAML(updatedScrapeConfigs.Bytes())
		if err != nil {
			return log.ErrorfNewErr("Failed to convert scrape config JSON to YAML: %v", err)
		}
		secret.Data[key] = newPromConfigData
		return nil
	})
	return err
}

// isLegacyDefaultMetricsBinding determines whether the given binding uses the
// "default" metrics template used pre-Verrazzano 1.4 AND the legacy VMI system prometheus config map
func isLegacyDefaultMetricsBinding(metricsBinding *vzapi.MetricsBinding) bool {
	templateName := metricsBinding.Spec.MetricsTemplate
	configMapName := metricsBinding.Spec.PrometheusConfigMap
	return templateName.Namespace == constants.LegacyDefaultMetricsTemplateNamespace &&
		templateName.Name == constants.LegacyDefaultMetricsTemplateName &&
		configMapName.Namespace == vzconst.VerrazzanoSystemNamespace &&
		configMapName.Name == vzconst.VmiPromConfigName
}
