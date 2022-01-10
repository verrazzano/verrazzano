// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstemplate

import (
	"context"
	"fmt"

	"github.com/Jeffail/gabs/v2"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	vztemplate "github.com/verrazzano/verrazzano/application-operator/controllers/template"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scontroller "sigs.k8s.io/controller-runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// Reconciler reconciles a metrics workload object
type Reconciler struct {
	k8sclient.Client
	Log     logr.Logger
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
	r.Log.V(1).Info("Reconcile metrics scrape config", "resource", req.NamespacedName)
	ctx := context.Background()

	// Fetch requested resource into MetricsBinding
	metricsBinding := vzapi.MetricsBinding{}
	if err := r.Client.Get(context.TODO(), req.NamespacedName, &metricsBinding); err != nil {
		return k8scontroller.Result{}, k8sclient.IgnoreNotFound(err)
	}

	// Reconcile based on the status of the deletion timestamp
	if metricsBinding.GetDeletionTimestamp().IsZero() {
		return r.reconcileTemplateCreateOrUpdate(ctx, &metricsBinding)
	}
	return r.reconcileTemplateDelete(ctx, &metricsBinding)
}

// reconcileTemplateDelete completes the reconcile process for an object that is being deleted
func (r *Reconciler) reconcileTemplateDelete(ctx context.Context, metricsBinding *vzapi.MetricsBinding) (k8scontroller.Result, error) {
	r.Log.V(2).Info("Reconcile for deleted object", "resource", metricsBinding.GetName())

	// For deletion, we have to remove the finalizer if it exists
	err := r.removeFinalizerIfRequired(ctx, metricsBinding)
	if err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Mutate the scrape config by deleting the entry
	if err := r.mutatePrometheusScrapeConfig(ctx, metricsBinding, r.deleteScrapeConfig); err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}
	return k8scontroller.Result{}, nil
}

// reconcileTemplateCreateOrUpdate completes the reconcile process for an object that is being created or updated
func (r *Reconciler) reconcileTemplateCreateOrUpdate(ctx context.Context, metricsBinding *vzapi.MetricsBinding) (k8scontroller.Result, error) {
	r.Log.V(2).Info("Reconcile for created or updated object", "resource", metricsBinding.GetName())

	// For creation, the finalizer must be added
	err := r.addFinalizerIfRequired(ctx, metricsBinding)
	if err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}

	// Mutate the scrape config by adding or updating the job
	if err := r.mutatePrometheusScrapeConfig(ctx, metricsBinding, r.createOrUpdateScrapeConfig); err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}
	return k8scontroller.Result{}, nil
}

// addFinalizerIfRequired adds the finalizer to the Template if required
// The finalizer is only added if the Template is not being deleted and the finalizer has not previously been added
func (r *Reconciler) addFinalizerIfRequired(ctx context.Context, metricsBinding *vzapi.MetricsBinding) error {
	if metricsBinding.GetDeletionTimestamp().IsZero() && !vzstring.SliceContainsString(metricsBinding.GetFinalizers(), finalizerName) {
		resourceName := metricsBinding.GetName()
		r.Log.V(2).Info("Adding finalizer from resource", "resource", resourceName)
		metricsBinding.SetFinalizers(append(metricsBinding.GetFinalizers(), finalizerName))
		if err := r.Update(ctx, metricsBinding); err != nil {
			r.Log.Error(err, fmt.Sprintf("Could not update the finalizer for resource: %s/%s", metricsBinding.GetObjectKind(), metricsBinding.GetName()))
			return err
		}
	}
	return nil
}

// removeFinalizerIfRequired removes the finalizer from the template if required
// The finalizer is only removed if the template is being deleted and the finalizer had been added
func (r *Reconciler) removeFinalizerIfRequired(ctx context.Context, metricsBinding *vzapi.MetricsBinding) error {
	if !metricsBinding.GetDeletionTimestamp().IsZero() && vzstring.SliceContainsString(metricsBinding.GetFinalizers(), finalizerName) {
		resourceName := metricsBinding.GetName()
		r.Log.Info("Removing finalizer from resource", "resource", resourceName)
		metricsBinding.SetFinalizers(vzstring.RemoveStringFromSlice(metricsBinding.GetFinalizers(), finalizerName))
		if err := r.Update(ctx, metricsBinding); err != nil {
			r.Log.Error(err, fmt.Sprintf("Could not update the finalizer for resource: %s/%s, ", metricsBinding.GetObjectKind(), metricsBinding.GetName()))
			return err
		}
	}
	return nil
}

// mutatePrometheusScrapeConfig takes the resource and a mutate function that determines the mutations of the scrape config
// mutations are dependant upon the status of the deletion timestamp
func (r *Reconciler) mutatePrometheusScrapeConfig(ctx context.Context, metricsBinding *vzapi.MetricsBinding, mutateFn func(metricsBinding *vzapi.MetricsBinding) (*k8scorev1.ConfigMap, error)) error {
	r.Log.V(2).Info("Mutating the Prometheus Scrape Config", "resource", metricsBinding.GetName())

	// Mutate the ConfigMap based on the given function
	configMap, err := mutateFn(metricsBinding)
	if err != nil {
		return err
	}

	//Apply the updated configmap
	r.Log.V(2).Info("Prometheus target ConfigMap is being altered", "resource", configMap.GetName())
	err = r.Client.Update(ctx, configMap)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Could not update the ConfigMap: %s", configMap.GetName()))
		return err
	}
	return nil
}

// Delete scrape config is a mutation function that deletes the scrape config data from the Prometheus ConfigMap
func (r *Reconciler) deleteScrapeConfig(metricsBinding *vzapi.MetricsBinding) (*k8scorev1.ConfigMap, error) {
	r.Log.V(2).Info("Scrape Config is being deleted from the Prometheus Config", "resource", metricsBinding.GetName())

	// Get the ConfigMap from the MetricsTemplate
	configMap, err := r.getPromConfigMap(metricsBinding)
	if err != nil {
		return nil, err
	}

	// Get data from the configmap
	promConfig, err := getConfigData(configMap)
	if err != nil {
		return nil, err
	}

	// Verify the Owner Reference exists
	if len(metricsBinding.OwnerReferences) < 1 {
		return nil, fmt.Errorf("No Owner Reference found in the MetricsBinding: %s", metricsBinding.GetName())
	}

	// Delete scrape config with job name matching resource
	scrapeConfigs := promConfig.Search(prometheusScrapeConfigsLabel).Children()
	for index, scrapeConfig := range scrapeConfigs {
		existingJobName := scrapeConfig.Search(prometheusJobNameLabel).Data()
		createdJobName := createJobName(metricsBinding)
		if existingJobName == createdJobName {
			err = promConfig.ArrayRemoveP(index, prometheusScrapeConfigsLabel)
			if err != nil {
				r.Log.Error(err, "Could remove array slice from Prometheus config")
				return nil, err
			}
		}
	}

	// Repopulate the configmap data
	newPromConfigData, err := yaml.JSONToYAML(promConfig.Bytes())
	if err != nil {
		r.Log.Error(err, "Could convert Prometheus config data to YAML")
		return nil, err
	}
	configMap.Data[prometheusConfigKey] = string(newPromConfigData)
	return configMap, nil
}

// createOrUpdateScrapeConfig is a mutation function that creates or updates the scrape config data within the given Prometheus ConfigMap
func (r *Reconciler) createOrUpdateScrapeConfig(metricsBinding *vzapi.MetricsBinding) (*k8scorev1.ConfigMap, error) {
	r.Log.V(2).Info("Scrape Config is being created or update in the Prometheus config", "resource", metricsBinding.GetName())

	// Get the MetricsTemplate from the MetricsBinding
	template, err := r.getMetricsTemplate(metricsBinding)
	if err != nil {
		return nil, err
	}

	// Get the ConfigMap from the MetricsTemplate
	configMap, err := r.getPromConfigMap(metricsBinding)
	if err != nil {
		return nil, err
	}

	// Get data from the configmap
	promConfig, err := getConfigData(configMap)
	if err != nil {
		return nil, err
	}

	// Get the OwnerReference object
	if len(metricsBinding.OwnerReferences) < 1 {
		return nil, fmt.Errorf("No Owner Reference found in the MetricsBinding: %s", metricsBinding.GetName())
	}
	owner := metricsBinding.OwnerReferences[0]
	workload := unstructured.Unstructured{}
	workload.SetKind(owner.Kind)
	workload.SetAPIVersion(owner.APIVersion)
	workloadName := types.NamespacedName{Namespace: metricsBinding.GetNamespace(), Name: metricsBinding.OwnerReferences[0].Name}
	err = r.Client.Get(context.Background(), workloadName, &workload)
	if err != nil {
		return nil, err
	}

	// Get the namespace for the template
	workloadNamespace := k8scorev1.Namespace{}
	err = r.Client.Get(context.TODO(), k8sclient.ObjectKey{Name: template.GetNamespace()}, &workloadNamespace)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Could not get the Namespace: %s", workloadNamespace.GetName()))
		return nil, err
	}

	// Create Unstructured Namespace
	workloadNamespaceUnstructuredMap, err := k8sruntime.DefaultUnstructuredConverter.ToUnstructured(&workloadNamespace)
	if err != nil {
		return nil, err
	}
	workloadNamespaceUnstructured := unstructured.Unstructured{Object: workloadNamespaceUnstructuredMap}

	// Organize inputs for template processor
	templateInputs := map[string]interface{}{
		"workload":  workload.Object,
		"namespace": workloadNamespaceUnstructured.Object,
	}

	// Get scrape config from the template processor and process the template inputs
	templateProcessor := vztemplate.NewProcessor(r.Client, template.Spec.PrometheusConfig.ScrapeConfigTemplate)
	scrapeConfigString, err := templateProcessor.Process(templateInputs)
	if err != nil {
		return nil, err
	}

	// Prepend job name to the template
	createdJobName := createJobName(metricsBinding)
	scrapeConfigString = formatJobName(createdJobName) + scrapeConfigString

	// Format scrape config into readable container
	configYaml, err := yaml.YAMLToJSON([]byte(scrapeConfigString))
	if err != nil {
		r.Log.Error(err, "Could not convert scrape config YAML to JSON")
		return nil, err
	}
	newScrapeConfig, err := gabs.ParseJSON(configYaml)
	if err != nil {
		r.Log.Error(err, "Could not convert scrape config JSON to container")
		return nil, err
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
				return nil, err
			}
			err = promConfig.ArrayAppendP(newScrapeConfig.Data(), prometheusScrapeConfigsLabel)
			if err != nil {
				return nil, err
			}
			existingUpdated = true
			break
		}
	}
	if !existingUpdated {
		err = promConfig.ArrayAppendP(newScrapeConfig.Data(), prometheusScrapeConfigsLabel)
		if err != nil {
			return nil, err
		}
	}

	// Repopulate the ConfigMap data
	newPromConfigData, err := yaml.JSONToYAML(promConfig.Bytes())
	if err != nil {
		return nil, err
	}
	configMap.Data[prometheusConfigKey] = string(newPromConfigData)
	return configMap, nil
}

// getMetricsTemplate returns the MetricsTemplate given in the MetricsBinding
func (r *Reconciler) getMetricsTemplate(metricsBinding *vzapi.MetricsBinding) (*vzapi.MetricsTemplate, error) {
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
		r.Log.Error(err, fmt.Sprintf("Could not get the MetricsTemplate: %s", templateSpec.Name))
		return nil, err
	}

	return &template, nil
}

// getPromConfigMap returns the Prometheus ConfigMap given in the MetricsTemplate
func (r *Reconciler) getPromConfigMap(metricsBinding *vzapi.MetricsBinding) (*k8scorev1.ConfigMap, error) {
	configMap := k8scorev1.ConfigMap{
		TypeMeta: k8smetav1.TypeMeta{
			Kind:       configMapKind,
			APIVersion: configMapAPIVersion,
		},
	}

	targetConfigMap := metricsBinding.Spec.PrometheusConfigMap
	namespacedName := types.NamespacedName{Name: targetConfigMap.Name, Namespace: targetConfigMap.Namespace}
	err := r.Client.Get(context.Background(), namespacedName, &configMap)
	if err != nil {
		r.Log.Error(err, fmt.Sprintf("Could not get the Prometheus target ConfigMap: %s", targetConfigMap.Name))
		return nil, err
	}

	return &configMap, nil
}
