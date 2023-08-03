// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

		// The next 2 blocks ensure that we keep the copies of any private CA bundles in play in sync with the source(s);
		// verrazzano-system/verrazzano-tls-ca is created/updated during VZ reconcile, but in the self-signed case that
		// can be rotated based on the cert duration.  So we keep verrazzano-system/verrazzano-tls-ca
		// in sync with any rotation updates, and then keep any upstream copies from that in sync when that
		// reconciles

		// Ingress secret was updated, if there's a CA crt update the verrazzano-tls-ca copy; this will trigger
		// a reconcile of that secret which should have us come through and update those copies
		// - Cert-Manager rotates the CA cert in the self-signed/custom CA case causing it to be updated in leaf cert secret
		if isVerrazzanoIngressSecretName(req.NamespacedName) || isVerrazzanoPrivateCABundle(req.NamespacedName) {
			return r.reconcileVerrazzanoTLS(ctx, req, vz)
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

func isVerrazzanoIngressSecretName(secretName types.NamespacedName) bool {
	return secretName.Name == constants.VerrazzanoIngressSecret && secretName.Namespace == constants.VerrazzanoSystemNamespace
}

func isVerrazzanoPrivateCABundle(secretName types.NamespacedName) bool {
	return secretName.Name == vzconst.PrivateCABundle && secretName.Namespace == constants.VerrazzanoSystemNamespace
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
	r.log.Debugf("Namespace %s does not exist, nothing to do", constants.VerrazzanoMultiClusterNamespace)
	return false
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(3, 5, time.Second)
}
