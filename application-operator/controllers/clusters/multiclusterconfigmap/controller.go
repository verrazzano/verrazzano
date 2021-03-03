// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclusterconfigmap

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
)

// Reconciler reconciles a MultiClusterConfigMap object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile reconciles a MultiClusterConfigMap resource. It fetches the embedded ConfigMap,
// mutates it based on the MultiClusterConfigMap, and updates the status of the
// MultiClusterConfigMap to reflect the success or failure of the changes to the embedded resource
// Currently it does NOT support Immutable ConfigMap resources
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("multiclusterconfigmap", req.NamespacedName)
	var mcConfigMap clustersv1alpha1.MultiClusterConfigMap
	result := reconcile.Result{}
	ctx := context.Background()
	err := r.fetchMultiClusterConfigMap(ctx, req.NamespacedName, &mcConfigMap)
	if err != nil {
		return result, clusters.IgnoreNotFoundWithLog("MultiClusterConfigMap", err, logger)
	}

	if !clusters.IsPlacedInThisCluster(ctx, r, mcConfigMap.Spec.Placement) {
		return ctrl.Result{}, nil
	}

	logger.Info("MultiClusterConfigMap create or update with underlying ConfigMap",
		"ConfigMap", mcConfigMap.Spec.Template.Metadata.Name,
		"placement", mcConfigMap.Spec.Placement.Clusters[0].Name)
	// Immutable ConfigMaps are not supported - we need a webhook to validate, or add the support
	opResult, err := r.createOrUpdateConfigMap(ctx, mcConfigMap)

	return r.updateStatus(ctx, &mcConfigMap, opResult, err)

}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.MultiClusterConfigMap{}).
		Complete(r)
}

func (r *Reconciler) fetchMultiClusterConfigMap(ctx context.Context, name types.NamespacedName, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap) error {
	return r.Get(ctx, name, mcConfigMap)
}

func (r *Reconciler) createOrUpdateConfigMap(ctx context.Context, mcConfigMap clustersv1alpha1.MultiClusterConfigMap) (controllerutil.OperationResult, error) {
	var configMap corev1.ConfigMap
	configMap.Namespace = mcConfigMap.Namespace
	configMap.Name = mcConfigMap.Name

	return controllerutil.CreateOrUpdate(ctx, r.Client, &configMap, func() error {
		r.mutateConfigMap(mcConfigMap, &configMap)
		// This SetControllerReference call will trigger garbage collection i.e. the ConfigMap
		// will automatically get deleted when the MultiClusterConfigMap is deleted
		return controllerutil.SetControllerReference(&mcConfigMap, &configMap, r.Scheme)
	})
}

// mutateConfigMap mutates the K8S ConfigMap to reflect the contents of the parent MultiClusterConfigMap
func (r *Reconciler) mutateConfigMap(mcConfigMap clustersv1alpha1.MultiClusterConfigMap, configMap *corev1.ConfigMap) {
	configMap.Data = mcConfigMap.Spec.Template.Data
	configMap.BinaryData = mcConfigMap.Spec.Template.BinaryData
	configMap.Immutable = mcConfigMap.Spec.Template.Immutable
	configMap.Labels = mcConfigMap.Spec.Template.Metadata.Labels
	configMap.Annotations = mcConfigMap.Spec.Template.Metadata.Annotations
}

func (r *Reconciler) updateStatus(ctx context.Context, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap, opResult controllerutil.OperationResult, err error) (ctrl.Result, error) {
	clusterName := clusters.GetClusterName(ctx, r.Client)
	condition := clusters.GetConditionFromResult(err, opResult, "ConfigMap")
	clusterLevelStatus := clusters.CreateClusterLevelStatus(condition, clusterName)
	if clusters.StatusNeedsUpdate(mcConfigMap.Status, condition, clusterLevelStatus) {
		mcConfigMap.Status.Conditions = append(mcConfigMap.Status.Conditions, condition)
		clusters.UpdateClusterLevelStatus(&mcConfigMap.Status, clusterLevelStatus)
		mcConfigMap.Status.State = clusters.ComputeEffectiveState(mcConfigMap.Status, mcConfigMap.Spec.Placement)
		return reconcile.Result{}, r.Status().Update(ctx, mcConfigMap)
	}
	return reconcile.Result{}, nil
}
