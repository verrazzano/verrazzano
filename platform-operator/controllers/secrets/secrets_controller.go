// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"time"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"k8s.io/apimachinery/pkg/types"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// VerrazzanoSecretsReconciler reconciles secrets.
// Currently the only secret reconciled is the verrazzano-tls secret. The controller
// ensures that a copy of the ca.crt secret (admin CA bundle) is copied to a secret
// in the verrazzano-mc namespace, so that managed clusters can fetch it.
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

func (r *VerrazzanoSecretsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// We only care about the CA secret for the cluster - this can come from the verrazzano ingress
	// tls secret (verrazzano-tls in verrazzano-system NS), OR from the tls-additional-ca in the
	// cattle-system NS (used in the Let's Encrypt staging cert case)
	isVzIngressSecret := isVerrazzanoIngressSecretName(req.NamespacedName)
	isAddnlTLSSecret := isAdditionalTLSSecretName(req.NamespacedName)
	if !isVzIngressSecret && !isAddnlTLSSecret {
		return ctrl.Result{}, nil
	}

	caKey := "ca.crt"
	if isAddnlTLSSecret {
		caKey = vzconst.AdditionalTLSCAKey
	}

	// Get the secret
	caSecret := corev1.Secret{}
	err := r.Get(context.TODO(), req.NamespacedName, &caSecret)
	if err != nil {
		// Secret should never be not found, unless we're running while installation is still underway
		zap.S().Errorf("Failed to fetch secret %s/%s: %v",
			req.Namespace, req.Name, err)
		return newRequeueWithDelay(), nil
	}

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           caSecret.Name,
		Namespace:      caSecret.Namespace,
		ID:             string(caSecret.UID),
		Generation:     caSecret.Generation,
		ControllerName: "secrets",
	})
	if err != nil {
		zap.S().Errorf("Failed to create resource logger for VerrazzanoSecrets controller", err)
		return newRequeueWithDelay(), nil
	}
	r.log = log

	if !r.multiclusterNamespaceExists() {
		// Multicluster namespace doesn't exist yet, nothing to do so requeue
		return newRequeueWithDelay(), nil
	}

	// Get the local ca-bundle secret
	mcCASecret := corev1.Secret{}
	err = r.Get(context.TODO(), client.ObjectKey{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      constants.VerrazzanoLocalCABundleSecret,
	}, &mcCASecret)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			r.log.Errorf("Failed to fetch secret %s/%s: %v",
				constants.VerrazzanoMultiClusterNamespace, constants.VerrazzanoLocalCABundleSecret, err)
			return newRequeueWithDelay(), nil
		}
		// Secret was not found, make a new one
		mcCASecret = corev1.Secret{}
		mcCASecret.Name = constants.VerrazzanoLocalCABundleSecret
		mcCASecret.Namespace = constants.VerrazzanoMultiClusterNamespace
	}

	result, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &mcCASecret, func() error {
		if mcCASecret.Data == nil {
			mcCASecret.Data = make(map[string][]byte)
		}
		mcCASecret.Data["ca-bundle"] = caSecret.Data[caKey]
		return nil
	})

	if err != nil {
		r.log.ErrorfThrottled("Failed to create or update secret %s/%s: %s",
			constants.VerrazzanoMultiClusterNamespace, constants.VerrazzanoLocalCABundleSecret, err.Error())
		return newRequeueWithDelay(), nil
	}

	r.log.Infof("Created or updated secret %s/%s (result: %v)",
		constants.VerrazzanoMultiClusterNamespace, constants.VerrazzanoLocalCABundleSecret, result)
	return ctrl.Result{}, nil
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
