// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"context"
	"fmt"

	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// VerrazzanoManagedClusterReconciler reconciles a VerrazzanoManagedCluster object
type VerrazzanoManagedClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    *zap.SugaredLogger
}

// +kubebuilder:rbac:groups=clusters.verrazzano.io,resources=verrazzanomanagedclusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clusters.verrazzano.io,resources=verrazzanomanagedclusters/status,verbs=get;update;patch

// Reconcile reconciles a VerrazzanoManagedCluster object
func (r *VerrazzanoManagedClusterReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.TODO()
	log := zap.S().With("resource", fmt.Sprintf("%s:%s", req.Namespace, req.Name))
	r.log = log
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
		log.Errorf("Failed to fetch resource: %v", err)
		return reconcile.Result{}, err
	}

	// If the VerrazzanoManagedCluster is being deleted then return success
	if vmc.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	// Reconcile the service account
	err = reconcileServiceAccount(r, vmc, req.Namespace)
	if err != nil {
		log.Info("Failed to reconcile the ServiceAccount")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func reconcileServiceAccount(r *VerrazzanoManagedClusterReconciler, vmc *clustersv1alpha1.VerrazzanoManagedCluster, namespace string) error {
	saNew := createServiceAccount(generateServiceAccountName(vmc.Name), namespace)

	// Does the service account exist?
	sa := corev1.ServiceAccount{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: saNew.Name, Namespace: saNew.Namespace}, &sa)
	if err != nil {
		if errors.IsNotFound(err) {
			// Create the SA
			r.log.Infof("Creating ServiceAccount %s in namespace %s", saNew.Name, saNew.Namespace)
			err2 := r.Client.Create(context.TODO(), saNew)
			if err2 != nil {
				return err2
			}
		} else {
			return err
		}
	}

	// Does the VerrazzanoManagedCluster object contain the service account name?
	if vmc.Spec.ServiceAccount != saNew.Name {
		r.log.Infof("Updating ServiceAccount from %q to %q", vmc.Spec.ServiceAccount, saNew.Name)
		vmc.Spec.ServiceAccount = saNew.Name
		err = r.Update(context.TODO(), vmc)
		if err != nil {
			return err
		}
	}

	return nil
}

// Generate the service account name
func generateServiceAccountName(clusterName string) string {
	return fmt.Sprintf("%s-managed-cluster", clusterName)
}

// Create a service account object
func createServiceAccount(name string, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoManagedClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.VerrazzanoManagedCluster{}).
		Complete(r)
}
