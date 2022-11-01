// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/status"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
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
// This controller also manages install override sources from the Verrazzano CR
type VerrazzanoSecretsReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	log           vzlog.VerrazzanoLogger
	StatusUpdater vzstatus.Updater
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

	vzList := &installv1alpha1.VerrazzanoList{}
	err := r.List(ctx, vzList)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch Verrazzano resource: %v", err)
		return newRequeueWithDelay(), err
	}
	if vzList != nil && len(vzList.Items) > 0 {
		vz := &vzList.Items[0]
		// Nothing to do if the vz resource is being deleted
		if vz.DeletionTimestamp != nil {
			return ctrl.Result{}, nil
		}

		// We care about the CA secret for the cluster - this can come from the verrazzano ingress
		// tls secret (verrazzano-tls in verrazzano-system NS), OR from the tls-additional-ca in the
		// cattle-system NS (used in the Let's Encrypt staging cert case)
		if isVerrazzanoIngressSecretName(req.NamespacedName) || isAdditionalTLSSecretName(req.NamespacedName) {
			return r.reconcileVerrazzanoTLS(ctx, req)
		}

		res, err := r.reconcileInstallOverrideSecret(ctx, req, vz)
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

func (r *VerrazzanoSecretsReconciler) additionalTLSSecretExists() bool {
	sec := corev1.Secret{}
	err := r.Get(context.TODO(), client.ObjectKey{
		Namespace: vzconst.RancherSystemNamespace,
		Name:      vzconst.AdditionalTLS,
	}, &sec)
	if err != nil && apierrors.IsNotFound(err) {
		return false
	}
	return true
}

func isAdditionalTLSSecretName(secretName types.NamespacedName) bool {
	return secretName.Name == vzconst.AdditionalTLS && secretName.Namespace == vzconst.RancherSystemNamespace
}

func isVerrazzanoIngressSecretName(secretName types.NamespacedName) bool {
	return secretName.Name == constants.VerrazzanoIngressSecret && secretName.Namespace == constants.VerrazzanoSystemNamespace
}

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
