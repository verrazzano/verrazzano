// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"time"

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

	// The only secret we care about (for now) is the verrazzano ingress tls secret (verrazzano-tls)
	if req.Name != constants.VerrazzanoIngressSecret || req.Namespace != constants.VerrazzanoSystemNamespace {
		return ctrl.Result{}, nil
	}

	// Get the verrazzano ingress secret
	caSecret := corev1.Secret{}
	err := r.Get(context.TODO(), req.NamespacedName, &caSecret)
	if err != nil {
		// Secret should never be not found, unless we're running while installation is still underway
		zap.S().Errorf("Failed to fetch secret %s/%s: %v",
			constants.VerrazzanoSystemNamespace, constants.VerrazzanoIngressSecret, err)
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
		mcCASecret.Data["ca-bundle"] = caSecret.Data["ca.crt"]
		return nil
	})
	if err != nil {
		r.log.Errorf("Failed to create or update secret %s/%s: %v",
			constants.VerrazzanoMultiClusterNamespace, constants.VerrazzanoLocalCABundleSecret, err)
		return newRequeueWithDelay(), nil
	}

	r.log.Infof("Created or updated secret %s/%s (result: %v)",
		constants.VerrazzanoMultiClusterNamespace, constants.VerrazzanoLocalCABundleSecret, result)
	return ctrl.Result{}, nil
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(3, 5, time.Second)
}
