// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclusterapplicationconfiguration

import (
	"context"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
)

// Reconciler reconciles a MultiClusterApplicationConfiguration resource. It fetches the embedded
// OAM ApplicationConfiguration, mutates it based on the MultiClusterApplicationConfiguration, and
// updates the status of the MultiClusterApplicationConfiguration to reflect the success or
// failure of the changes to the embedded resource
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// Reconcile reconciles a MultiClusterApplicationConfiguration resource. It fetches the embedded OAM
// app config, mutates it based on the MultiClusterApplicationConfiguration, and updates the status
// of the MultiClusterApplicationConfiguration to reflect the success or failure of the changes to
// the embedded resource
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("multiclusterapplicationconfiguration", req.NamespacedName)
	var mcAppConfig clustersv1alpha1.MultiClusterApplicationConfiguration
	result := reconcile.Result{}
	ctx := context.Background()
	err := r.fetchMultiClusterAppConfig(ctx, req.NamespacedName, &mcAppConfig)
	if err != nil {
		return result, clusters.IgnoreNotFoundWithLog("MultiClusterApplicationConfiguration", err, logger)
	}

	if !clusters.IsPlacedInThisCluster(ctx, r, mcAppConfig.Spec.Placement) {
		return ctrl.Result{}, nil
	}

	logger.Info("MultiClusterApplicationConfiguration create or update with underlying OAM applicationconfiguration",
		"applicationconfiguration", mcAppConfig.Spec.Template.Metadata.Name,
		"placement", mcAppConfig.Spec.Placement.Clusters[0].Name)
	opResult, err := r.createOrUpdateAppConfig(ctx, mcAppConfig)

	return r.updateStatus(ctx, &mcAppConfig, opResult, err)
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.MultiClusterApplicationConfiguration{}).
		Complete(r)
}

func (r *Reconciler) fetchMultiClusterAppConfig(ctx context.Context, name types.NamespacedName, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration) error {
	return r.Get(ctx, name, mcAppConfig)
}

func (r *Reconciler) createOrUpdateAppConfig(ctx context.Context, mcAppConfig clustersv1alpha1.MultiClusterApplicationConfiguration) (controllerutil.OperationResult, error) {
	var oamAppConfig v1alpha2.ApplicationConfiguration
	oamAppConfig.Namespace = mcAppConfig.Namespace
	oamAppConfig.Name = mcAppConfig.Name

	return controllerutil.CreateOrUpdate(ctx, r.Client, &oamAppConfig, func() error {
		r.mutateAppConfig(mcAppConfig, &oamAppConfig)
		// This SetControllerReference call will trigger garbage collection i.e. the OAM app config
		// will automatically get deleted when the MultiClusterApplicationConfiguration is deleted
		return controllerutil.SetControllerReference(&mcAppConfig, &oamAppConfig, r.Scheme)
	})
}

// mutateAppConfig mutates the OAM app config to reflect the contents of the parent
// MultiClusterApplicationConfiguration
func (r *Reconciler) mutateAppConfig(mcAppConfig clustersv1alpha1.MultiClusterApplicationConfiguration, oamAppConfig *v1alpha2.ApplicationConfiguration) {
	oamAppConfig.Spec = mcAppConfig.Spec.Template.Spec
	oamAppConfig.Labels = mcAppConfig.Spec.Template.Metadata.Labels
	oamAppConfig.Annotations = mcAppConfig.Spec.Template.Metadata.Annotations
}

func (r *Reconciler) updateStatus(ctx context.Context, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration, opResult controllerutil.OperationResult, err error) (ctrl.Result, error) {
	condition, state := clusters.GetConditionAndStateFromResult(err, opResult, "OAM Application Configuration")
	if clusters.StatusNeedsUpdate(mcAppConfig.Status.Conditions, state, condition, state) {
		mcAppConfig.Status.State = state
		mcAppConfig.Status.Conditions = append(mcAppConfig.Status.Conditions, condition)
		return reconcile.Result{}, r.Status().Update(ctx, mcAppConfig)
	}
	return reconcile.Result{}, nil
}
