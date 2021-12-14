// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstemplate

import (
	"context"
	"github.com/go-logr/logr"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/app/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
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

// setupWithManagerForGVK creates a controller for a specific GKV and adds it to the manager.
func (r *Reconciler) setupWithManagerForGVK(mgr k8scontroller.Manager, group string, version string, kind string) error {
	u := unstructured.Unstructured{}
	u.SetGroupVersionKind(schema.GroupVersionKind{Group: group, Version: version, Kind: kind})
	return k8scontroller.NewControllerManagedBy(mgr).For(&u).Complete(r)
}

// SetupWithManager creates controllers for each supported GKV and adds it to the manager
// See https://book-v1.book.kubebuilder.io/beyond_basics/controller_watches.html for potentially better way to watch arbitrary resources
func (r *Reconciler) SetupWithManager(mgr k8scontroller.Manager) error {
	//TODO: Need some way to lookup set of supported workload GVKs.
	if err := r.setupWithManagerForGVK(mgr, "apps", "v1", "Deployment"); err != nil {
		return err
	}
	if err := r.setupWithManagerForGVK(mgr, "apps", "v1", "ReplicaSet"); err != nil {
		return err
	}
	if err := r.setupWithManagerForGVK(mgr, "apps", "v1", "StatefulSet"); err != nil {
		return err
	}
	if err := r.setupWithManagerForGVK(mgr, "apps", "v1", "DaemonSet"); err != nil {
		return err
	}
	if err := r.setupWithManagerForGVK(mgr, "weblogic.oracle", "v7", "Domain"); err != nil {
		return err
	}
	if err := r.setupWithManagerForGVK(mgr, "weblogic.oracle", "v8", "Domain"); err != nil {
		return err
	}
	if err := r.setupWithManagerForGVK(mgr, "coherence.oracle.com", "v1", "Coherence"); err != nil {
		return err
	}
	return nil
}

// Reconcile reconciles a workload to keep the Prometheus ConfigMap scrape job configuration up to date.
// No kubebuilder annotations are used as the application RBAC for the application operator is now manually managed.
func (r *Reconciler) Reconcile(req k8scontroller.Request) (k8scontroller.Result, error) {
	r.Log.V(1).Info("Reconcile metrics scrape config", "resource", req.NamespacedName)
	ctx := context.Background()

	// Fetch request resource into an Unstructured type
	resource, err := r.getRequestedResource(req.NamespacedName)
	if err != nil {
		return k8scontroller.Result{}, k8sclient.IgnoreNotFound(err)
	}

	// Check for label in resource
	// If no label exists, do nothing
	labels := resource.GetLabels()
	resourceUID, keyExists := labels["app.verrazzano.io/metrics-workload-uid"]
	if !keyExists || resourceUID != string(resource.GetUID()) {
		return k8scontroller.Result{}, nil
	}

	if resource.GetDeletionTimestamp() == nil {
		return r.reconcileTraitCreateOrUpdate(ctx, resource)
	}
	return r.reconcileTraitDelete(ctx, resource)
}

// getRequestedResource returns an Unstructured value from the namespace and name given in the request
func (r *Reconciler) getRequestedResource (namespacedName types.NamespacedName) (*unstructured.Unstructured, error) {
	var uns *unstructured.Unstructured
	r.Log.Info("Fetch related resource", "resource", namespacedName)
	if err := r.Client.Get(context.TODO(), namespacedName, uns); err != nil {
		return nil, err
	}
	return uns, nil
}

// reconcileTraitDelete completes the reconcile process for an object that is being deleted
func (r *Reconciler) reconcileTraitDelete(ctx context.Context, resource *unstructured.Unstructured) (k8scontroller.Result, error) {
	if err := r.mutatePrometheusScrapeConfig(ctx, resource, r.deleteScrapeConfig); err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}
	return k8scontroller.Result{}, nil
}

// reconcileTraitCreateOrUpdate completes the reconcile process for an object that is being created or updated
func (r *Reconciler) reconcileTraitCreateOrUpdate(ctx context.Context, resource *unstructured.Unstructured) (k8scontroller.Result, error) {
	if err := r.mutatePrometheusScrapeConfig(ctx, resource, r.createOrUpdateScrapeConfig); err != nil {
		return k8scontroller.Result{Requeue: true}, err
	}
	return k8scontroller.Result{}, nil
}

func (r *Reconciler) mutatePrometheusScrapeConfig(ctx context.Context, resource *unstructured.Unstructured, mutatefn func(configMap *v1.ConfigMap, namespacedName types.NamespacedName, resource *unstructured.Unstructured) error) error {
	// Verify that the configmap label
	labels := resource.GetLabels()
	configmapUID, labelExists := labels["app.verrazzano.io/metrics-prometheus-configmap-uid"]
	if !labelExists {
		return nil
	}

	// Find ConfigMap by the Given UID and delete the scrape config
	var configMap v1.ConfigMap
	err := r.getResourceFromUID(ctx, &configMap, configmapUID)
	if err != nil {
		return err
	}

	// Mutate the ConfigMap based on the given function
	err = mutatefn(&configMap, types.NamespacedName{Name: resource.GetName(), Namespace: resource.GetNamespace()}, resource)
	if err != nil {
		return err
	}

	//Apply the updated configmap
	r.Client.Update(ctx, &configMap)
	return nil
}

func (r *Reconciler) deleteScrapeConfig(configMap *v1.ConfigMap, namespacedName types.NamespacedName, resource *unstructured.Unstructured) error {
	// Get data from the configmap
	promConfig, err := getConfigData(configMap)
	if err != nil {
		return err
	}

	// Delete scrape config with job name matching resource
	scrapeConfigs := promConfig.Search(prometheusScrapeConfigsLabel).Children()
	for index, scrapeConfig := range scrapeConfigs {
		existingJobName := scrapeConfig.Search(prometheusJobNameLabel).Data()
		createdJobName := createJobName(namespacedName, resource.GetUID())
		if existingJobName == createdJobName {
			err = promConfig.ArrayRemoveP(index, prometheusScrapeConfigsLabel)
			if err != nil {
				return err
			}
		}
	}

	// Repopulate the configmap data
	newPromConfigData, err := yaml.JSONToYAML(promConfig.Bytes())
	if err != nil {
		return err
	}
	configMap.Data[prometheusConfigKey] = string(newPromConfigData)
	return nil
}

func (r *Reconciler) createOrUpdateScrapeConfig(configMap *v1.ConfigMap, namespacedName types.NamespacedName, resource *unstructured.Unstructured) error {
	// Get data from the configmap
	promConfig, err := getConfigData(configMap)
	if err != nil {
		return err
	}

	// Create get the metrics template from the UID
	labels := resource.GetLabels()
	metricsTemplateUID := labels["app.verrazzano.io/metrics-template-uid"]
	var metricsTemplate vzapi.MetricsTemplate
	err = r.getResourceFromUID(context.Background(), &metricsTemplate, metricsTemplateUID)
	if err != nil {
		return err
	}

	// Create or Update scrape config with job name matching resource
	scrapeConfigs := promConfig.Search(prometheusScrapeConfigsLabel).Children()
	for index, scrapeConfig := range scrapeConfigs {
		existingJobName := scrapeConfig.Search(prometheusJobNameLabel).Data()
		createdJobName := createJobName(namespacedName, resource.GetUID())
		if existingJobName == createdJobName {
			// Remove and recreate scrape config
			err = scrapeConfig.ArrayRemoveP(index, prometheusScrapeConfigsLabel)
			if err != nil {
				return err
			}
			err = scrapeConfig.ArrayAppendP("", prometheusScrapeConfigsLabel)
			if err != nil {
				return err
			}
		}
	}

	// Repopulate the configmap data
	newPromConfigData, err := yaml.JSONToYAML(promConfig.Bytes())
	if err != nil {
		return err
	}
	configMap.Data[prometheusConfigKey] = string(newPromConfigData)
	return nil
}

func (r *Reconciler) getResourceFromUID(ctx context.Context, resource k8sruntime.Object, objectUID string) error {
	var objects *unstructured.UnstructuredList
	r.Client.List(ctx, objects)
	for _, object := range objects.Items {
		if string(object.GetUID()) == objectUID {
			err := k8sruntime.DefaultUnstructuredConverter.FromUnstructured(object.UnstructuredContent(), resource)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
