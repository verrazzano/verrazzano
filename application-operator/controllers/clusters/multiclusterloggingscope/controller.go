// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclusterloggingscope

import (
	"context"

	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a MultiClusterLoggingScope object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile reconciles a MultiClusterLoggingScope resource. It fetches the embedded LoggingScope,
// mutates it based on the MultiClusterLoggingScope, and updates the status of the
// MultiClusterLoggingScope to reflect the success or failure of the changes to the embedded resource
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("multiclusterloggingscope", req.NamespacedName)
	var mcLogScope clustersv1alpha1.MultiClusterLoggingScope
	result := reconcile.Result{}
	ctx := context.Background()
	err := r.fetchMultiClusterLoggingScope(ctx, req.NamespacedName, &mcLogScope)
	if err != nil {
		return result, clusters.IgnoreNotFoundWithLog("MultiClusterLoggingScope", err, logger)
	}

	if !clusters.IsPlacedInThisCluster(ctx, r, mcLogScope.Spec.Placement) {
		return ctrl.Result{}, nil
	}

	logger.Info("MultiClusterLoggingScope create or update with underlying LoggingScope",
		"loggingscope", mcLogScope.Spec.Template.Metadata.Name,
		"placement", mcLogScope.Spec.Placement.Clusters[0].Name)
	opResult, err := r.createOrUpdateLoggingScope(ctx, mcLogScope)

	return r.updateStatus(ctx, &mcLogScope, opResult, err)
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.MultiClusterLoggingScope{}).
		Complete(r)
}

func (r *Reconciler) fetchMultiClusterLoggingScope(ctx context.Context, name types.NamespacedName, mcScope *clustersv1alpha1.MultiClusterLoggingScope) error {
	return r.Get(ctx, name, mcScope)
}

func (r *Reconciler) createOrUpdateLoggingScope(ctx context.Context, mcLogScope clustersv1alpha1.MultiClusterLoggingScope) (controllerutil.OperationResult, error) {
	var logScope v1alpha1.LoggingScope
	logScope.Namespace = mcLogScope.Namespace
	logScope.Name = mcLogScope.Name

	return controllerutil.CreateOrUpdate(ctx, r.Client, &logScope, func() error {
		r.mutateLoggingScope(mcLogScope, &logScope)
		// This SetControllerReference call will trigger garbage collection i.e. the LoggingScope
		// will automatically get deleted when the MultiClusterLoggingScope is deleted
		return controllerutil.SetControllerReference(&mcLogScope, &logScope, r.Scheme)
	})
}

func (r *Reconciler) mutateLoggingScope(mcLogScope clustersv1alpha1.MultiClusterLoggingScope, logScope *v1alpha1.LoggingScope) {
	logScope.Spec = mcLogScope.Spec.Template.Spec
	logScope.Labels = mcLogScope.Spec.Template.Metadata.Labels
	logScope.Annotations = mcLogScope.Spec.Template.Metadata.Annotations
}

func (r *Reconciler) updateStatus(ctx context.Context, mcLogScope *clustersv1alpha1.MultiClusterLoggingScope, opResult controllerutil.OperationResult, err error) (ctrl.Result, error) {
	clusterName := clusters.GetClusterName(ctx, r.Client)
	newCondition := clusters.GetConditionFromResult(err, opResult, "LoggingScope")
	return clusters.UpdateStatus(&mcLogScope.Status, mcLogScope.Spec.Placement, newCondition, clusterName,
		func() error { return r.Status().Update(ctx, mcLogScope) })
}
