// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	finalizerKey = "verrazzano.io/oci-ocne-cluster"
	templatesDir = "template"
)

type ClusterReconciler struct {
	clipkg.Client
	Scheme            *runtime.Scheme
	Logger            *zap.SugaredLogger
	CredentialsLoader oci.CredentialsLoader
}

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	q := &vmcv1alpha1.OCNEOCIQuickCreate{}
	err := r.Get(ctx, req.NamespacedName, q)
	// if cluster not found, no work to be done
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return controller.RequeueDelay(), err
	}
	return r.reconcile(ctx, q)
}

func (r *ClusterReconciler) reconcile(ctx context.Context, q *vmcv1alpha1.OCNEOCIQuickCreate) (ctrl.Result, error) {
	// If quick create is completed, or being deleted, clean up the quick create
	if !q.GetDeletionTimestamp().IsZero() || q.Status.QuickCreateStatus.Phase == vmcv1alpha1.QuickCreatePhaseComplete {
		return ctrl.Result{}, r.delete(ctx, q)
	}
	// Add any finalizers if they are not present
	if isMissingFinalizer(q) {
		return r.setFinalizers(ctx, q)
	}
	return r.syncCluster(ctx, q)
}

func (r *ClusterReconciler) delete(ctx context.Context, q *vmcv1alpha1.OCNEOCIQuickCreate) error {
	if !vzstring.SliceContainsString(q.GetFinalizers(), finalizerKey) {
		return nil
	}
	q.SetFinalizers(vzstring.RemoveStringFromSlice(q.GetFinalizers(), finalizerKey))
	err := r.Update(ctx, q)
	if err != nil && !apierrors.IsConflict(err) {
		return err
	}
	return nil
}

func (r *ClusterReconciler) setFinalizers(ctx context.Context, q *vmcv1alpha1.OCNEOCIQuickCreate) (ctrl.Result, error) {
	q.Finalizers = append(q.GetFinalizers(), finalizerKey)
	if err := r.Update(ctx, q); err != nil {
		return controller.RequeueDelay(), err
	}
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) syncCluster(ctx context.Context, q *vmcv1alpha1.OCNEOCIQuickCreate) (ctrl.Result, error) {
	// If provisioning has not successfully started, attempt to provisioning the cluster
	if shouldProvision(q) {
		ocne, err := NewOCNE(ctx, r.Client, r.CredentialsLoader, q)
		if err != nil {
			return controller.RequeueDelay(), err
		}
		if err := ocne.ApplyFromTemplateDirectory(r.Client, templatesDir); err != nil {
			return controller.RequeueDelay(), err
		}
		q.Status.QuickCreateStatus.Phase = vmcv1alpha1.QuickCreatePhaseProvisioning
		return r.updateStatus(ctx, q)
	}
	// If provisioning has been completed, update the quick create to completed phase
	if isComplete(q) {
		q.Status.QuickCreateStatus.Phase = vmcv1alpha1.QuickCreatePhaseComplete
		return r.updateStatus(ctx, q)
	}
	// Quick Create is not complete yet, requeue
	return controller.RequeueDelay(), nil
}

func (r *ClusterReconciler) updateStatus(ctx context.Context, q *vmcv1alpha1.OCNEOCIQuickCreate) (ctrl.Result, error) {
	if err := r.Status().Update(ctx, q); err != nil {
		return controller.RequeueDelay(), err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vmcv1alpha1.OCNEOCIQuickCreate{}).
		Complete(r)
}
