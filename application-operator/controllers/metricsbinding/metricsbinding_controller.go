// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricsbinding

import (
	"context"
	"github.com/Jeffail/gabs/v2"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vztemplate "github.com/verrazzano/verrazzano/application-operator/controllers/template"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	k8scorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scontroller "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
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

// createOrUpdateScrapeConfig is a mutation function that creates or updates the scrape config data within the given Prometheus ConfigMap
func (r *Reconciler) createOrUpdateScrapeConfig(metricsBinding *vzapi.MetricsBinding, configMap *k8scorev1.ConfigMap, log vzlog.VerrazzanoLogger) error {
	log.Debugw("Scrape Config is being created or update in the Prometheus config", "resource", metricsBinding.GetName())

	// Get the MetricsTemplate from the MetricsBinding
	template, err := r.getMetricsTemplate(context.Background(), metricsBinding, log)
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
