// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	_ "embed"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/capi"
	"github.com/verrazzano/verrazzano/pkg/k8s/node"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	finalizerKey = "verrazzano.io/oci-ocne-cluster"
)

var (
	//go:embed template/addons/addons.goyaml
	addonsTemplate []byte
	//go:embed template/cluster/cluster.goyaml
	clusterTemplate []byte
	//go:embed template/cluster/nodes.goyaml
	nodesTemplate []byte
	//go:embed template/cluster/ocne.goyaml
	ocneTemplate []byte
)

type ClusterReconciler struct {
	*controller.Base
	Scheme            *runtime.Scheme
	CredentialsLoader oci.CredentialsLoader
	OCIClientGetter   func(credentials *oci.Credentials) (oci.Client, error)
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
	if err := r.SetNewResourceLogger(q); err != nil {
		return controller.RequeueDelay(), err
	}
	return r.reconcile(ctx, q)
}

func (r *ClusterReconciler) reconcile(ctx context.Context, q *vmcv1alpha1.OCNEOCIQuickCreate) (ctrl.Result, error) {
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

func (r *ClusterReconciler) syncCluster(ctx context.Context, q *vmcv1alpha1.OCNEOCIQuickCreate) (ctrl.Result, error) {
	ocne, err := NewProperties(ctx, r.Client, r.CredentialsLoader, r.OCIClientGetter, q)
	if err != nil {
		return controller.RequeueDelay(), err
	}
	// If provisioning has not successfully started, attempt to provisioning the cluster
	if shouldProvision(q) {
		if err := controller.ApplyTemplates(r.Client, ocne, q.Namespace, clusterTemplate, nodesTemplate, ocneTemplate); err != nil {
			return controller.RequeueDelay(), err
		}
		q.Status = vmcv1alpha1.OCNEOCIQuickCreateStatus{
			Phase: vmcv1alpha1.QuickCreatePhaseProvisioning,
		}
		r.Log.Oncef("provisioning OCNE OCI cluster: %s/%s", q.Namespace, q.Name)
		return r.UpdateStatus(ctx, q)
	}
	// If OCI Network is loaded, update the quick create to completed phase
	if ocne.HasOCINetwork() {
		if err := controller.ApplyTemplates(r.Client, ocne, q.Namespace, addonsTemplate); err != nil {
			return controller.RequeueDelay(), err
		}
		// If the cluster only has control plane nodes, set them for scheduling
		if ocne.IsControlPlaneOnly() {
			if err := r.setControlPlaneSchedulable(ctx, q); err != nil {
				return controller.RequeueDelay(), nil
			}
		}
		r.Log.Oncef("completed provisioning OCNE OCI cluster: %s/%s", q.Namespace, q.Name)
		q.Status.Phase = vmcv1alpha1.QuickCreatePhaseComplete
		return r.UpdateStatus(ctx, q)
	}

	r.Log.Progressf("waiting for OCNE OCI cluster infrastructure: %s/%s", q.Namespace, q.Name)
	// Quick Create is not complete yet, requeue
	return controller.RequeueDelay(), nil
}

func (r *ClusterReconciler) setControlPlaneSchedulable(ctx context.Context, q *vmcv1alpha1.OCNEOCIQuickCreate) error {
	cli, err := capi.GetClusterClient(ctx, r.Client, types.NamespacedName{
		Namespace: q.Namespace,
		Name:      q.Name,
	}, r.Scheme)
	if err != nil {
		return err
	}
	return node.SetControlPlaneScheduling(ctx, cli)
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&vmcv1alpha1.OCNEOCIQuickCreate{}).
		Complete(r)
}
