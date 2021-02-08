// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// VerrazzanoManagedClusterReconciler reconciles a VerrazzanoManagedCluster object
type VerrazzanoManagedClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const resourceKind = "VerrazzanoManagedCluster"

// +kubebuilder:rbac:groups=clusters.verrazzano.io,resources=verrazzanomanagedclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clusters.verrazzano.io,resources=verrazzanomanagedclusters/status,verbs=get;update;patch

// Reconcile reconciles a VerrazzanoManagedCluster object
func (r *VerrazzanoManagedClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	log := zap.S().With("resource", fmt.Sprintf("%s:%s", req.Namespace, req.Name))
	log.Info("Reconciler called")
	vmc := &clustersv1alpha1.VerrazzanoManagedCluster{}

	err := r.Get(ctx, req.NamespacedName, vmc)
	if err != nil {
		// If the resource is not found, that means all of the finalizers have been removed,
		// and the verrazzano resource has been deleted, so there is nothing left to do.
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		// Error getting the verrazzano resource - don't requeue.
		log.Errorf("Failed to fetch %s resource: %v", resourceKind, err)
		return reconcile.Result{}, err
	}

	// If the VerrazzanoManagedCluster is being deleted then return success
	if vmc.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	// Reconcile the cluster-id
	clusterID := generateClusterID(vmc.Name, vmc.Namespace)
	if vmc.Spec.ClusterID != clusterID {
		log.Infof("Updating ClusterID from %q to %q on %s resource %s in namespace %s", vmc.Spec.ClusterID, clusterID, resourceKind, vmc.Name, vmc.Namespace)
		vmc.Spec.ClusterID = clusterID
		err := r.Update(ctx, vmc)
		if err != nil {
			log.Errorf("Failed to update %s resource %s in namespace %s: %v", resourceKind, vmc.Name, vmc.Namespace, err)
			return ctrl.Result{}, err
		}
	}

	// Reconcile the service account
	err = reconcileServiceAccount(vmc, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func reconcileServiceAccount(r *VerrazzanoManagedClusterReconciler, vmc *clustersv1alpha1.VerrazzanoManagedCluster, namespace string) error {
	saNew := createServiceAccount(vmc.Spec.ClusterID, namespace)

	// Does the service account exist?
	sa, err := r.Client.Get()

}

// Generate the value of the cluster identifier
func generateClusterID(clusterName string, namespace string) string {
	return fmt.Sprintf("%s-%s", clusterName, namespace)
}

// Create a service account object
func createServiceAccount(name string, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Secrets:                      nil,
		ImagePullSecrets:             nil,
		AutomountServiceAccountToken: nil,
	}
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoManagedClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.VerrazzanoManagedCluster{}).
		Complete(r)
}
