// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	finalizerKey = "verrazzano.io/oci-ocne-cluster"
)

type OCIOCNEClusterReconciler struct {
	clipkg.Client
	Scheme *runtime.Scheme
	Logger *zap.SugaredLogger
}

func (r *OCIOCNEClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	cluster := &vmcv1alpha1.OCIOCNECluster{}
	err := r.Get(ctx, req.NamespacedName, cluster)
	// if cluster not found, no work to be done
	if apierrors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	err = r.reconcile(ctx, cluster)
	if err != nil {
		return newRequeueWithDelay(), err
	}
	return ctrl.Result{}, nil
}

func (r OCIOCNEClusterReconciler) reconcile(ctx context.Context, cluster *vmcv1alpha1.OCIOCNECluster) error {
	// If cluster is being deleted, handle delete
	if !cluster.GetDeletionTimestamp().IsZero() {
		return r.delete(ctx, cluster)
	}
	// Set finalizer if not present
	if err := r.setFinalizer(ctx, cluster); err != nil {
		return err
	}
	return r.syncCluster(ctx, cluster)
}

func (r *OCIOCNEClusterReconciler) delete(ctx context.Context, cluster *vmcv1alpha1.OCIOCNECluster) error {
	if !vzstring.SliceContainsString(cluster.GetFinalizers(), finalizerKey) {
		return nil
	}
	cluster.SetFinalizers(vzstring.RemoveStringFromSlice(cluster.GetFinalizers(), finalizerKey))
	err := r.Update(ctx, cluster)
	if err != nil && !apierrors.IsConflict(err) {
		return err
	}
	return nil
}

func (r *OCIOCNEClusterReconciler) setFinalizer(ctx context.Context, cluster *vmcv1alpha1.OCIOCNECluster) error {
	if finalizers, added := vzstring.SliceAddString(cluster.GetFinalizers(), finalizerKey); added {
		cluster.SetFinalizers(finalizers)
		if err := r.Update(ctx, cluster); err != nil {
			return err
		}
	}
	return nil
}

func (r *OCIOCNEClusterReconciler) syncCluster(ctx context.Context, cluster *vmcv1alpha1.OCIOCNECluster) error {
	return nil
}

func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second)
}
