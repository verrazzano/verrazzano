// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclustercomponent

import (
	"context"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/go-logr/logr"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a MultiClusterComponent object
type Reconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=clusters.verrazzano.io,resources=multiclustercomponents,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clusters.verrazzano.io,resources=multiclustercomponents/status,verbs=get;update;patch

// Reconcile reconciles a MultiClusterComponent resource. It fetches the embedded OAM Component,
// mutates it based on the MultiClusterComponent, and updates the status of the
// MultiClusterComponent to reflect the success or failure of the changes to the embedded resource
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("multiclustercomponent", req.NamespacedName)
	var mcComp clustersv1alpha1.MultiClusterComponent
	result := reconcile.Result{}
	ctx := context.Background()
	err := r.fetchMultiClusterComponent(ctx, req.NamespacedName, &mcComp)
	if err != nil {
		logger.Info("Failed to fetch MultiClusterSecret", "err", err)
		return result, client.IgnoreNotFound(err)
	}

	logger.Info("MultiClusterComponent create or update with underlying component",
		"component", mcComp.Spec.Template.Metadata.Name,
		"placement", mcComp.Spec.Placement.Clusters[0].Name)
	opResult, err := r.createOrUpdateComponent(ctx, mcComp)

	return r.updateStatus(ctx, &mcComp, opResult, err)
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.MultiClusterComponent{}).
		Complete(r)
}

func (r *Reconciler) fetchMultiClusterComponent(ctx context.Context, name types.NamespacedName, mcComp *clustersv1alpha1.MultiClusterComponent) error {
	return r.Get(ctx, name, mcComp)
}

func (r *Reconciler) createOrUpdateComponent(ctx context.Context, mcComp clustersv1alpha1.MultiClusterComponent) (controllerutil.OperationResult, error) {
	var oamComp v1alpha2.Component
	oamComp.Namespace = mcComp.Namespace
	oamComp.Name = mcComp.Name

	return controllerutil.CreateOrUpdate(ctx, r.Client, &oamComp, func() error {
		r.mutateComponent(mcComp, &oamComp)
		// This SetControllerReference call will trigger garbage collection i.e. the OAM component
		// will automatically get deleted when the MultiClusterComponent is deleted
		return controllerutil.SetControllerReference(&mcComp, &oamComp, r.Scheme)
	})
}

// mutateSecret mutates the corev1.Secret to reflect the contents of the parent MultiClusterSecret
func (r *Reconciler) mutateComponent(mcComp clustersv1alpha1.MultiClusterComponent, oamComp *v1alpha2.Component) {
	oamComp.Spec = mcComp.Spec.Template.Spec
}

func (r *Reconciler) updateStatus(ctx context.Context, mcComp *clustersv1alpha1.MultiClusterComponent, opResult controllerutil.OperationResult, err error) (ctrl.Result, error) {
	condition, state := clusters.GetConditionAndStateFromResult(err, opResult, "OAM Component")
	if clusters.StatusNeedsUpdate(mcComp.Status.Conditions, state, condition, state) {
		mcComp.Status.State = state
		mcComp.Status.Conditions = append(mcComp.Status.Conditions, condition)
		return reconcile.Result{}, r.Status().Update(ctx, mcComp)
	}
	return reconcile.Result{}, nil
}
