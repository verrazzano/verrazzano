// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package multiclustersecret

import (
	"context"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzlogInit "github.com/verrazzano/verrazzano/pkg/log"
	vzlog2 "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Reconciler reconciles a MultiClusterSecret object
type Reconciler struct {
	client.Client
	Log          *zap.SugaredLogger
	Scheme       *runtime.Scheme
	AgentChannel chan clusters.StatusUpdateMessage
}

const (
	finalizerName  = "multiclustersecret.verrazzano.io"
	controllerName = "multiclustersecret"
)

// Reconcile reconciles a MultiClusterSecret resource. It fetches the embedded Secret, mutates it
// based on the MultiClusterSecret, and updates the status of the MultiClusterSecret to reflect the
// success or failure of the changes to the embedded Secret
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	// We do not want any resource to get reconciled if it is in namespace kube-system
	// This is due to a bug found in OKE, it should not affect functionality of any vz operators
	// If this is the case then return success
	if req.Namespace == vzconst.KubeSystem {
		log := zap.S().With(vzlogInit.FieldResourceNamespace, req.Namespace, vzlogInit.FieldResourceName, req.Name, vzlogInit.FieldController, controllerName)
		log.Infof("Multi-cluster secret resource %v should not be reconciled in kube-system namespace, ignoring", req.NamespacedName)
		return reconcile.Result{}, nil
	}

	if ctx == nil {
		ctx = context.Background()
	}
	var mcSecret clustersv1alpha1.MultiClusterSecret
	err := r.fetchMultiClusterSecret(ctx, req.NamespacedName, &mcSecret)
	if err != nil {
		return clusters.IgnoreNotFoundWithLog(err, zap.S())
	}
	log, err := clusters.GetResourceLogger("mcsecret", req.NamespacedName, &mcSecret)
	if err != nil {
		zap.S().Errorf("Failed to create controller logger for multi-cluster secret resource: %v", err)
		return clusters.NewRequeueWithDelay(), nil
	}
	log.Oncef("Reconciling multi-cluster secret resource %v, generation %v", req.NamespacedName, mcSecret.Generation)

	res, err := r.doReconcile(ctx, mcSecret, log)
	if clusters.ShouldRequeue(res) {
		return res, nil
	}
	// Never return an error since it has already been logged and we don't want the
	// controller runtime to log again (with stack trace).  Just re-queue if there is an error.
	if err != nil {
		return clusters.NewRequeueWithDelay(), nil
	}

	log.Oncef("Finished reconciling multi-cluster secret %v", req.NamespacedName)

	return ctrl.Result{}, nil
}

// doReconcile performs the reconciliation operations for the MC secret
func (r *Reconciler) doReconcile(ctx context.Context, mcSecret clustersv1alpha1.MultiClusterSecret, log vzlog2.VerrazzanoLogger) (ctrl.Result, error) {
	// delete the wrapped resource since MC is being deleted
	if !mcSecret.ObjectMeta.DeletionTimestamp.IsZero() {
		err := clusters.DeleteAssociatedResource(ctx, r.Client, &mcSecret, finalizerName, &corev1.Secret{}, types.NamespacedName{Namespace: mcSecret.Namespace, Name: mcSecret.Name})
		if err != nil {
			log.Errorf("Failed to delete associated secret and finalizer: %v", err)
		}
		return ctrl.Result{}, err
	}

	oldState := clusters.SetEffectiveStateIfChanged(mcSecret.Spec.Placement, &mcSecret.Status)
	if !clusters.IsPlacedInThisCluster(ctx, r, mcSecret.Spec.Placement) {
		if oldState != mcSecret.Status.State {
			// This must be done whether the resource is placed in this cluster or not, because we
			// could be in an admin cluster and receive cluster level statuses from managed clusters,
			// which can change our effective state
			err := r.Status().Update(ctx, &mcSecret)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
		// if this mc secret is no longer placed on this cluster, remove the associated secret
		err := clusters.DeleteAssociatedResource(ctx, r.Client, &mcSecret, finalizerName, &corev1.Secret{}, types.NamespacedName{Namespace: mcSecret.Namespace, Name: mcSecret.Name})
		return ctrl.Result{}, err
	}

	log.Debug("MultiClusterSecret create or update with underlying secret",
		"secret", mcSecret.Spec.Template.Metadata.Name,
		"placement", mcSecret.Spec.Placement.Clusters[0].Name)
	opResult, err := r.createOrUpdateSecret(ctx, mcSecret)

	// Add our finalizer if not already added
	if err == nil {
		_, err = clusters.AddFinalizer(ctx, r.Client, &mcSecret, finalizerName)
	}

	ctrlResult, updateErr := r.updateStatus(ctx, &mcSecret, opResult, err)

	// if an error occurred in createOrUpdate, return that error with a requeue
	// even if update status succeeded
	if err != nil {
		res := ctrl.Result{Requeue: true, RequeueAfter: clusters.GetRandomRequeueDelay()}
		return res, err
	}

	return ctrlResult, updateErr

}

func (r *Reconciler) updateStatus(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret, opResult controllerutil.OperationResult, err error) (ctrl.Result, error) {
	clusterName := clusters.GetClusterName(ctx, r.Client)
	newCondition := clusters.GetConditionFromResult(err, opResult, "Secret")
	return clusters.UpdateStatus(mcSecret, &mcSecret.Status, mcSecret.Spec.Placement, newCondition, clusterName,
		r.AgentChannel, func() error { return r.Status().Update(ctx, mcSecret) })
}

// SetupWithManager registers our controller with the manager
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.MultiClusterSecret{}).
		Complete(r)
}

func (r *Reconciler) fetchMultiClusterSecret(ctx context.Context, name types.NamespacedName, mcSecretRef *clustersv1alpha1.MultiClusterSecret) error {
	return r.Get(ctx, name, mcSecretRef)
}

func (r *Reconciler) createOrUpdateSecret(ctx context.Context, mcSecret clustersv1alpha1.MultiClusterSecret) (controllerutil.OperationResult, error) {
	var secret corev1.Secret
	secret.Namespace = mcSecret.Namespace
	secret.Name = mcSecret.Name

	return controllerutil.CreateOrUpdate(ctx, r.Client, &secret, func() error {
		r.mutateSecret(mcSecret, &secret)
		return nil
	})

}

// mutateSecret mutates the corev1.Secret to reflect the contents of the parent MultiClusterSecret
func (r *Reconciler) mutateSecret(mcSecret clustersv1alpha1.MultiClusterSecret, secret *corev1.Secret) {
	secret.Type = mcSecret.Spec.Template.Type
	secret.Data = mcSecret.Spec.Template.Data
	secret.StringData = mcSecret.Spec.Template.StringData
	secret.Labels = mcSecret.Spec.Template.Metadata.Labels
	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	secret.Labels[vzconst.VerrazzanoManagedLabelKey] = constants.LabelVerrazzanoManagedDefault
	secret.Annotations = mcSecret.Spec.Template.Metadata.Annotations
}
