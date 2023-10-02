// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oke

import (
	"context"
	_ "embed"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	finalizerKey = "verrazzano.io/oci-oke-cluster"
)

var (
	//go:embed template/cluster/cluster.goyaml
	clusterTemplate []byte
	//go:embed template/nodes/nodes.goyaml
	nodesTemplate []byte
)

type ClusterReconciler struct {
	*controller.Base
	Scheme            *runtime.Scheme
	CredentialsLoader oci.CredentialsLoader
	OCIClientGetter   func(credentials *oci.Credentials) (oci.Client, error)
}

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	q := &vmcv1alpha1.OKEQuickCreate{}
	err := r.Get(ctx, req.NamespacedName, q)
	// if cluster not found, no work to be done
	if apierrors.IsNotFound(err) {
		return controller.RequeueDelay(), nil
	}
	if err != nil {
		return controller.RequeueDelay(), err
	}
	if err := r.SetNewResourceLogger(q); err != nil {
		return controller.RequeueDelay(), err
	}
	return r.reconcile(ctx, q)
}

func (r *ClusterReconciler) reconcile(ctx context.Context, q *vmcv1alpha1.OKEQuickCreate) (ctrl.Result, error) {
	// If quick create is completed, or being deleted, clean up the quick create
	if !q.GetDeletionTimestamp().IsZero() || q.Status.Phase == vmcv1alpha1.QuickCreatePhaseComplete {
		return ctrl.Result{}, r.Cleanup(ctx, q, finalizerKey)
	}
	// Add any finalizers if they are not present
	if isMissingFinalizer(q) {
		return r.SetFinalizers(ctx, q, finalizerKey)
	}
	return r.syncCluster(ctx, q)
}

func (r *ClusterReconciler) syncCluster(ctx context.Context, q *vmcv1alpha1.OKEQuickCreate) (ctrl.Result, error) {
	props, err := NewProperties(ctx, r.Client, r.CredentialsLoader, r.OCIClientGetter, q)
	if err != nil {
		return controller.RequeueDelay(), err
	}
	// If provisioning has not successfully started, attempt to create the OKE control plane
	if shouldProvision(q) {
		if err := controller.ApplyTemplates(r.Client, props, q.Namespace, clusterTemplate); err != nil {
			return controller.RequeueDelay(), err
		}
		q.Status = vmcv1alpha1.OKEQuickCreateStatus{
			Phase: vmcv1alpha1.QuickCreatePhaseProvisioning,
		}
		r.Log.Oncef("provisioning OKE cluster: %s/%s", q.Namespace, q.Name)
		return r.UpdateStatus(ctx, q)
	}
	// If OCI Network is loaded, create the nodes and update phase to completed
	if props.HasOCINetwork() {
		if err := controller.ApplyTemplates(r.Client, props, q.Namespace, nodesTemplate); err != nil {
			return controller.RequeueDelay(), err
		}
		q.Status.Phase = vmcv1alpha1.QuickCreatePhaseComplete
		r.Log.Oncef("completed provisioning OKE cluster: %s/%s", q.Namespace, q.Name)
		return r.UpdateStatus(ctx, q)
	}
	r.Log.Progressf("waiting for OKE cluster infrastructure: %s/%s", q.Namespace, q.Name)
	// Quick Create is not complete yet, requeue
	return controller.RequeueDelay(), nil
}

func isMissingFinalizer(q *vmcv1alpha1.OKEQuickCreate) bool {
	return !vzstring.SliceContainsString(q.GetFinalizers(), finalizerKey)
}

func shouldProvision(q *vmcv1alpha1.OKEQuickCreate) bool {
	return q.Status.Phase == ""
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vmcv1alpha1.OKEQuickCreate{}).
		Complete(r)
}
