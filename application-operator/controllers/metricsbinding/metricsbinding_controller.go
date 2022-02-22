// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	"fmt"
	"time"

	"github.com/Jeffail/gabs/v2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vztemplate "github.com/verrazzano/verrazzano/application-operator/controllers/template"
	vzlog "github.com/verrazzano/verrazzano/pkg/log/vzlog"
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

// SetupWithManager creates controller for the MetricsBinding
func (r *Reconciler) SetupWithManager(mgr k8scontroller.Manager) error {
	return k8scontroller.NewControllerManagedBy(mgr).For(&vzapi.MetricsBinding{}).Complete(r)
}

// Reconcile reconciles a workload to keep the Prometheus ConfigMap scrape job configuration up to date.
// No kubebuilder annotations are used as the application RBAC for the application operator is now manually managed.
func (r *Reconciler) Reconcile(req k8scontroller.Request) (k8scontroller.Result, error) {
	ctx := context.Background()
	metricsBinding := vzapi.MetricsBinding{}
	if err := r.Client.Get(context.TODO(), req.NamespacedName, &metricsBinding); err != nil {
		return clusters.IgnoreNotFoundWithLog(err, zap.S())
	}
	log, err := clusters.GetResourceLogger("metricsbinding", req.NamespacedName, &metricsBinding)
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for metrics binding", err)
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

	// Mutate the MetricsBinding before the scrape config
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, metricsBinding, func() error {
		return r.updateMetricsBinding(metricsBinding, log)
	})
	if err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Mutate the scrape config by adding or updating the job
	if err := r.mutatePrometheusScrapeConfig(ctx, metricsBinding, r.createOrUpdateScrapeConfig, log); err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Requeue with a delay to account for situations where the scrape config
	// has changed but without the MetricsBinding changing.
	var seconds = rand.IntnRange(45, 90)
	var duration = time.Duration(seconds) * time.Second
	return reconcile.Result{Requeue: true, RequeueAfter: duration}, nil
}

func (r *Reconciler) updateMetricsBinding(metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) error {
	// Add the finalizer
	controllerutil.AddFinalizer(metricsBinding, finalizerName)

	// Retrieve the workload object from the MetricsBinding
	workloadObject, err := r.getWorkloadObject(metricsBinding)
	if err != nil {
		log.Errorf("Failed to get the Workload from the MetricsBinding %s: %v", metricsBinding.Spec.Workload.Name, err)
		return err
	}

	// Return error if UID is not found
	if len(workloadObject.GetUID()) == 0 {
		err = fmt.Errorf("Could not get UID from workload resource: %s, %s", workloadObject.GetKind(), workloadObject.GetName())
		log.Errorf("Failed to find UID for workload %s: %v", workloadObject.GetName(), err)
		return err
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
func (r *Reconciler) mutatePrometheusScrapeConfig(ctx context.Context, metricsBinding *vzapi.MetricsBinding, mutateFn func(metricsBinding *vzapi.MetricsBinding, configMap *k8scorev1.ConfigMap, log vzlog.VerrazzanoLogger) error, log vzlog.VerrazzanoLogger) error {
	log.Debugw("Mutating the Prometheus Scrape Config", "resource", metricsBinding.GetName())

	var configMap = r.getPromConfigMap(metricsBinding) // Apply the updated configmap
	log.Debugw("Prometheus target ConfigMap is being altered", "resource", configMap.GetName())
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, configMap, func() error {
		return mutateFn(metricsBinding, configMap, log)
	})
	if err != nil {
		return err
	}
	return nil
}

// deleteScrapeConfig is a mutation function that deletes the scrape config data from the Prometheus ConfigMap
func (r *Reconciler) deleteScrapeConfig(metricsBinding *vzapi.MetricsBinding, configMap *k8scorev1.ConfigMap, log vzlog.VerrazzanoLogger) error {
	log.Debugw("Scrape Config is being deleted from the Prometheus Config", "resource", metricsBinding.GetName())

	// Get data from the configmap
	promConfig, err := getConfigData(configMap)
	if err != nil {
		return err
	}

	// Verify the Owner Reference exists
	if len(metricsBinding.OwnerReferences) < 1 {
		return fmt.Errorf("No Owner Reference found in the MetricsBinding: %s", metricsBinding.GetName())
	}

	// Delete scrape config with job name matching resource
	scrapeConfigs := promConfig.Search(prometheusScrapeConfigsLabel).Children()
	for index, scrapeConfig := range scrapeConfigs {
		existingJobName := scrapeConfig.Search(prometheusJobNameLabel).Data()
		createdJobName := createJobName(metricsBinding)
		if existingJobName == createdJobName {
			err = promConfig.ArrayRemoveP(index, prometheusScrapeConfigsLabel)
			if err != nil {
				log.Errorf("Failed to remove array slice from Prometheus config: %v", err)
				return err
			}
		}
	}

	// Repopulate the configmap data
	newPromConfigData, err := yaml.JSONToYAML(promConfig.Bytes())
	if err != nil {
		log.Errorf("Failed to convert Prometheus config data to YAML: %v", err)
		return err
	}
	configMap.Data[prometheusConfigKey] = string(newPromConfigData)
	return nil
}

// createOrUpdateScrapeConfig is a mutation function that creates or updates the scrape config data within the given Prometheus ConfigMap
func (r *Reconciler) createOrUpdateScrapeConfig(metricsBinding *vzapi.MetricsBinding, configMap *k8scorev1.ConfigMap, log vzlog.VerrazzanoLogger) error {
	log.Debugw("Scrape Config is being created or update in the Prometheus config", "resource", metricsBinding.GetName())

	// Get the MetricsTemplate from the MetricsBinding
	template, err := r.getMetricsTemplate(metricsBinding, log)
	if err != nil {
		return err
	}

	// Get data from the configmap
	promConfig, err := getConfigData(configMap)
	if err != nil {
		return err
	}

	// Get the namespace for the template
	workloadNamespace := k8scorev1.Namespace{}
	err = r.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: template.GetNamespace()}, &workloadNamespace)
	if err != nil {
		log.Errorf("Failed get the Namespace %s: %v", workloadNamespace.GetName(), err)
		return err
	}

	// Create Unstructured Namespace
	workloadNamespaceUnstructuredMap, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(&workloadNamespace)
	if err != nil {
		return err
	}
	workloadNamespaceUnstructured := unstructured.Unstructured{Object: workloadNamespaceUnstructuredMap}

	// Get the workload object for the template processor
	workloadObject, err := r.getWorkloadObject(metricsBinding)
	if err != nil {
		return err
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
		return err
	}

	// Prepend job name to the scrape config
	createdJobName := createJobName(metricsBinding)
	scrapeConfigString = formatJobName(createdJobName) + scrapeConfigString

	// Format scrape config into readable container
	configYaml, err := yaml.YAMLToJSON([]byte(scrapeConfigString))
	if err != nil {
		log.Errorf("Failed to convert scrape config YAML to JSON: %v", err)
		return err
	}
	newScrapeConfig, err := gabs.ParseJSON(configYaml)
	if err != nil {
		log.Errorf("Failed to convert scrape config JSON to container: %v", err)
		return err
	}

	// Create or Update scrape config with job name matching resource
	existingUpdated := false
	scrapeConfigs := promConfig.Search(prometheusScrapeConfigsLabel).Children()
	for index, scrapeConfig := range scrapeConfigs {
		existingJobName := scrapeConfig.Search(prometheusJobNameLabel).Data()
		if existingJobName == createdJobName {
			// Remove and recreate scrape config
			err = promConfig.ArrayRemoveP(index, prometheusScrapeConfigsLabel)
			if err != nil {
				return err
			}
			err = promConfig.ArrayAppendP(newScrapeConfig.Data(), prometheusScrapeConfigsLabel)
			if err != nil {
				return err
			}
			existingUpdated = true
			break
		}
	}
	if !existingUpdated {
		err = promConfig.ArrayAppendP(newScrapeConfig.Data(), prometheusScrapeConfigsLabel)
		if err != nil {
			return err
		}
	}

	// Repopulate the ConfigMap data
	newPromConfigData, err := yaml.JSONToYAML(promConfig.Bytes())
	if err != nil {
		return err
	}
	configMap.Data[prometheusConfigKey] = string(newPromConfigData)
	return nil
}

// getMetricsTemplate returns the MetricsTemplate given in the MetricsBinding
func (r *Reconciler) getMetricsTemplate(metricsBinding *vzapi.MetricsBinding, log vzlog.VerrazzanoLogger) (*vzapi.MetricsTemplate, error) {
	template := vzapi.MetricsTemplate{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       metricsTemplateKind,
			APIVersion: metricsTemplateAPIVersion,
		},
	}

	templateSpec := metricsBinding.Spec.MetricsTemplate
	namespacedName := types.NamespacedName{Name: templateSpec.Name, Namespace: templateSpec.Namespace}
	err := r.Client.Get(context.Background(), namespacedName, &template)
	if err != nil {
		log.Errorf("Failed to get the MetricsTemplate %s: %v", templateSpec.Name, err)
		return nil, err
	}

	return &template, nil
}

// getPromConfigMap returns the Prometheus ConfigMap given in the MetricsBinding
func (r *Reconciler) getPromConfigMap(metricsBinding *vzapi.MetricsBinding) *k8scorev1.ConfigMap {
	targetConfigMap := metricsBinding.Spec.PrometheusConfigMap
	return &k8scorev1.ConfigMap{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       configMapKind,
			APIVersion: configMapAPIVersion,
		},
		ObjectMeta: k8smetav1.ObjectMeta{
			Name:      targetConfigMap.Name,
			Namespace: targetConfigMap.Namespace,
		},
	}
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
