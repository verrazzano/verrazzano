// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclustercomponent

import (
	"context"
	"errors"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	vzlog2 "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	finalizerName  = "multiclustercomponent.verrazzano.io"
	controllerName = "multiclustercomponent"
)

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
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if ctx == nil {
		return ctrl.Result{}, errors.New("context cannot be nil")
	}

	// We do not want any resource to get reconciled if it is in namespace kube-system
	// This is due to a bug found in OKE, it should not affect functionality of any vz operators
	// If this is the case then return success
	if req.Namespace == vzconst.KubeSystem {
		log := zap.S().With(vzlog.FieldResourceNamespace, req.Namespace, vzlog.FieldResourceName, req.Name, vzlog.FieldController, controllerName)
		log.Infof("Multi-cluster component resource %v should not be reconciled in kube-system namespace, ignoring", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	var mcComp clustersv1alpha1.MultiClusterComponent
	err := r.fetchMultiClusterComponent(ctx, req.NamespacedName, &mcComp)
	if err != nil {
		return clusters.IgnoreNotFoundWithLog(err, zap.S())
	}
	log, err := clusters.GetResourceLogger("mccomponent", req.NamespacedName, &mcComp)
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for multi-cluster component resource: %v", err)
		return clusters.NewRequeueWithDelay(), nil
	}
	log.Oncef("Reconciling multi-cluster component resource %v, generation %v", req.NamespacedName, mcComp.Generation)

	res, err := r.doReconcile(ctx, mcComp, log)
	if clusters.ShouldRequeue(res) {
		return res, nil
	}
	// Never return an error since it has already been logged and we don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		return clusters.NewRequeueWithDelay(), nil
	}

	log.Oncef("Finished reconciling multi-cluster component %v", req.NamespacedName)

	return ctrl.Result{}, nil
}

// doReconcile performs the reconciliation operations for the MC component
func (r *Reconciler) doReconcile(ctx context.Context, mcComp clustersv1alpha1.MultiClusterComponent, log vzlog2.VerrazzanoLogger) (ctrl.Result, error) {
	// delete the wrapped resource since MC is being deleted
	if !mcComp.ObjectMeta.DeletionTimestamp.IsZero() {
		err := clusters.DeleteAssociatedResource(ctx, r.Client, &mcComp, finalizerName, &v1alpha2.Component{}, types.NamespacedName{Namespace: mcComp.Namespace, Name: mcComp.Name})
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
			err := r.Status().Update(ctx, &mcComp)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		// if this mc component is no longer placed on this cluster, remove the associated component
		err := clusters.DeleteAssociatedResource(ctx, r.Client, &mcComp, finalizerName, &v1alpha2.Component{}, types.NamespacedName{Namespace: mcComp.Namespace, Name: mcComp.Name})
		return ctrl.Result{}, err
	}

	log.Debug("MultiClusterComponent create or update with underlying component",
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
	if oamComp.Labels == nil {
		oamComp.Labels = map[string]string{}
	}
	oamComp.Labels[vzconst.VerrazzanoManagedLabelKey] = constants.LabelVerrazzanoManagedDefault
	oamComp.Annotations = mcComp.Spec.Template.Metadata.Annotations
}

func (r *Reconciler) updateStatus(ctx context.Context, mcComp *clustersv1alpha1.MultiClusterComponent, opResult controllerutil.OperationResult, err error) (ctrl.Result, error) {
	clusterName := clusters.GetClusterName(ctx, r.Client)
	newCondition := clusters.GetConditionFromResult(err, opResult, "OAM Component")
	return clusters.UpdateStatus(mcComp, &mcComp.Status, mcComp.Spec.Placement, newCondition, clusterName,
		r.AgentChannel, func() error { return r.Status().Update(ctx, mcComp) })
}
