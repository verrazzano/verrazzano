// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"go.uber.org/zap"
)

// VerrazzanoSecretsReconciler reconciles secrets.
// One part of the controller is for the verrazzano-tls secret. The controller
// ensures that a copy of the ca.crt secret (admin CA bundle) is copied to a secret
// in the verrazzano-mc namespace, so that managed clusters can fetch it.
// This controller also manages Helm override sources from the Verrazzano CR
type VerrazzanoSecretsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    vzlog.VerrazzanoLogger
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoSecretsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Complete(r)
}

// Reconcile the Secret
func (r *VerrazzanoSecretsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// One secret we care about is the verrazzano ingress tls secret (verrazzano-tls)
	if ctx == nil {
		ctx = context.TODO()
	}

	if req.Name == constants.VerrazzanoIngressSecret && req.Namespace == constants.VerrazzanoSystemNamespace {
		return r.reconcileVerrazzanoTLS(ctx, req)
	}

	vzList := &installv1alpha1.VerrazzanoList{}
	listOpts := []client.ListOption{
		client.InNamespace(""),
	}
	err := r.List(ctx, vzList, listOpts...)
	if err != nil {
		if apierrors.IsNotFound(err) {
			zap.S().Infof("VZ not found Secret")
			return reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch Verrazzano resource: %v", err)
		return newRequeueWithDelay(), err
	}
	if vzList != nil && len(vzList.Items) > 0 {
		vz := &vzList.Items[0]
		res, err := r.reconcileHelmOverrideSecret(ctx, req, vz)
		if err != nil {
			zap.S().Errorf("Failed to reconcile Secret: %v", err)
			return newRequeueWithDelay(), err
		}
		return res, nil
	}

	return ctrl.Result{}, nil

}

// initialize secret logger
func (r *VerrazzanoSecretsReconciler) initLogger(secret corev1.Secret) (ctrl.Result, error) {
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           secret.Name,
		Namespace:      secret.Namespace,
		ID:             string(secret.UID),
		Generation:     secret.Generation,
		ControllerName: "secrets",
	})
	if err != nil {
		zap.S().Errorf("Failed to create resource logger for VerrazzanoSecrets controller: %v", err)
		return newRequeueWithDelay(), err
	}
	r.log = log
	return ctrl.Result{}, nil
}

// multiclusterNamespaceExists checks if the Verrazzano Multi Cluster namespace exists
func (r *VerrazzanoSecretsReconciler) multiclusterNamespaceExists() bool {
	ns := corev1.Namespace{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoMultiClusterNamespace}, &ns)
	if err == nil {
		return true
	}
	if !apierrors.IsNotFound(err) {
		r.log.ErrorfThrottled("Unexpected error checking for namespace %s: %v", constants.VerrazzanoMultiClusterNamespace, err)
	}
	r.log.Progressf("Namespace %s does not exist, nothing to do", constants.VerrazzanoMultiClusterNamespace)
	return false
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(3, 5, time.Second)
}
