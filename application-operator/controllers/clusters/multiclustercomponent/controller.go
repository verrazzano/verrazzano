// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclustercomponent

import (
	"context"

	"go.uber.org/zap"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const finalizerName = "multiclustercomponent.verrazzano.io"

// Reconciler reconciles a MultiClusterComponent object
type Reconciler struct {
	client.Client
	Log          *zap.SugaredLogger
	Scheme       *runtime.Scheme
	AgentChannel chan clusters.StatusUpdateMessage
}

// Reconcile reconciles a MultiClusterComponent resource. It fetches the embedded OAM Component,
// mutates it based on the MultiClusterComponent, and updates the status of the
// MultiClusterComponent to reflect the success or failure of the changes to the embedded resource
func (r *Reconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.With("multiclustercomponent", req.NamespacedName)
	var mcComp clustersv1alpha1.MultiClusterComponent
	result := reconcile.Result{}
	ctx := context.Background()
	err := r.fetchMultiClusterComponent(ctx, req.NamespacedName, &mcComp)
	if err != nil {
		return result, clusters.IgnoreNotFoundWithLog("MultiClusterComponent", err, log)
	}

	// delete the wrapped resource since MC is being deleted
	if !mcComp.ObjectMeta.DeletionTimestamp.IsZero() {
		err = clusters.DeleteAssociatedResource(ctx, r.Client, &mcComp, finalizerName, &v1alpha2.Component{}, types.NamespacedName{Namespace: mcComp.Namespace, Name: mcComp.Name})
		if err != nil {
			log.Errorf("Failed to delete associated component and finalizer: %v", err)
		}
		return ctrl.Result{}, err
	}

	oldState := clusters.SetEffectiveStateIfChanged(mcComp.Spec.Placement, &mcComp.Status)

	if !clusters.IsPlacedInThisCluster(ctx, r, mcComp.Spec.Placement) {
		if oldState != mcComp.Status.State {
			// This must be done whether the resource is placed in this cluster or not, because we
			// could be in an admin cluster and receive cluster level statuses from managed clusters,
			// which can change our effective state
			err = r.Status().Update(ctx, &mcComp)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		// if this mc component is no longer placed on this cluster, remove the associated component
		err = clusters.DeleteAssociatedResource(ctx, r.Client, &mcComp, finalizerName, &v1alpha2.Component{}, types.NamespacedName{Namespace: mcComp.Namespace, Name: mcComp.Name})
		return ctrl.Result{}, err
	}

	log.Debugw("MultiClusterComponent create or update with underlying component",
		"component", mcComp.Spec.Template.Metadata.Name,
		"placement", mcComp.Spec.Placement.Clusters[0].Name)
	opResult, err := r.createOrUpdateComponent(ctx, mcComp)

	// Add our finalizer if not already added
	if err == nil {
		_, err = clusters.AddFinalizer(ctx, r.Client, &mcComp, finalizerName)
	}

	ctrlResult, updateErr := r.updateStatus(ctx, &mcComp, opResult, err)

	// if an error occurred in createOrUpdate, return that error with a requeue
	// even if update status succeeded
	if err != nil {
		res := ctrl.Result{Requeue: true, RequeueAfter: clusters.GetRandomRequeueDelay()}
		return res, err
	}

	return ctrlResult, updateErr
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
		return nil
	})
}

// mutateComponent mutates the OAM component to reflect the contents of the parent MultiClusterComponent
func (r *Reconciler) mutateComponent(mcComp clustersv1alpha1.MultiClusterComponent, oamComp *v1alpha2.Component) {
	oamComp.Spec = mcComp.Spec.Template.Spec
	oamComp.Labels = mcComp.Spec.Template.Metadata.Labels
	oamComp.Annotations = mcComp.Spec.Template.Metadata.Annotations
}

func (r *Reconciler) updateStatus(ctx context.Context, mcComp *clustersv1alpha1.MultiClusterComponent, opResult controllerutil.OperationResult, err error) (ctrl.Result, error) {
	clusterName := clusters.GetClusterName(ctx, r.Client)
	newCondition := clusters.GetConditionFromResult(err, opResult, "OAM Component")
	return clusters.UpdateStatus(mcComp, &mcComp.Status, mcComp.Spec.Placement, newCondition, clusterName,
		r.AgentChannel, func() error { return r.Status().Update(ctx, mcComp) })
}
