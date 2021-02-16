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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// VerrazzanoManagedClusterReconciler reconciles a VerrazzanoManagedCluster object
type VerrazzanoManagedClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    *zap.SugaredLogger
}

const mcRoleAndBindingName = "verrazzano-managed-cluster"

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

		// Error getting the VerrazzanoManagedCluster resource
		log.Errorf("Failed to fetch resource: %v", err)
		return reconcile.Result{}, err
	}

	// If the VerrazzanoManagedCluster is being deleted then return success
	if vmc.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	// Reconcile the service account
	err = r.reconcileServiceAccount(vmc)
	if err != nil {
		log.Infof("Failed to reconcile the ServiceAccount: %v", err)
		return ctrl.Result{}, err
	}

	err = r.reconcileManagedRoleBinding(vmc)
	if err != nil {
		log.Infof("Failed to reconcile the ServiceAccount: %v", err)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VerrazzanoManagedClusterReconciler) reconcileServiceAccount(vmc *clustersv1alpha1.VerrazzanoManagedCluster) error {
	// Create or update the service account
	_, err := r.createOrUpdateServiceAccount(context.TODO(), vmc)
	if err != nil {
		return err
	}

	// Does the VerrazzanoManagedCluster object contain the service account name?
	saName := generateManagedResourceName(vmc.Name)
	if vmc.Spec.ServiceAccount != saName {
		r.log.Infof("Updating ServiceAccount from %q to %q", vmc.Spec.ServiceAccount, saName)
		vmc.Spec.ServiceAccount = saName
		err = r.Update(context.TODO(), vmc)
		if err != nil {
			return err
		}
	}

	return nil
}

// Create or update the ServiceAccount for a VerrazzanoManagedCluster
func (r *VerrazzanoManagedClusterReconciler) createOrUpdateServiceAccount(ctx context.Context, vmc *clustersv1alpha1.VerrazzanoManagedCluster) (controllerutil.OperationResult, error) {
	var serviceAccount corev1.ServiceAccount
	serviceAccount.Namespace = vmc.Namespace
	serviceAccount.Name = generateManagedResourceName(vmc.Name)

	return controllerutil.CreateOrUpdate(ctx, r.Client, &serviceAccount, func() error {
		r.mutateServiceAccount(vmc, &serviceAccount)
		// This SetControllerReference call will trigger garbage collection i.e. the serviceAccount
		// will automatically get deleted when the VerrazzanoManagedCluster is deleted
		return controllerutil.SetControllerReference(vmc, &serviceAccount, r.Scheme)
	})
}

func (r *VerrazzanoManagedClusterReconciler) mutateServiceAccount(vmc *clustersv1alpha1.VerrazzanoManagedCluster, serviceAccount *corev1.ServiceAccount) {
	serviceAccount.Name = generateManagedResourceName(vmc.Name)
}

// Generate the common name used by all resources specific to a given managed cluster
func generateManagedResourceName(clusterName string) string {
	return fmt.Sprintf("%s-managed-cluster", clusterName)
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoManagedClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&clustersv1alpha1.VerrazzanoManagedCluster{}).
		Complete(r)
}
