// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (r *VerrazzanoSecretsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// One secret we care about is the verrazzano ingress tls secret (verrazzano-tls)
	if req.Name == constants.VerrazzanoIngressSecret && req.Namespace == constants.VerrazzanoSystemNamespace {
		return r.reconcileVerrazzanoTLS(ctx, req)
	}

	// TODO List:
	// 1. Get the Verrazzano CR and verify that the Namespace of it and the request align
	//      a) i.e. vz.Namespace == req.Namespace
	// 2. Verify that the Secret exists as a helm override (use vzconfig.vzContainsResources)
	vz := &installv1alpha1.Verrazzano{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: constants.DefaultNamespace}, vz); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch Verrazzano resource: %v", err)
		return newRequeueWithDelay(), err
	}

	res, err := r.reconcileHelmOverrideSecret(ctx, req, vz)
	if err != nil {
		zap.S().Errorf("Failed to reconcile Secret: %v", err)
		return newRequeueWithDelay(), err
	}

	return res, nil

}

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
		zap.S().Errorf("Failed to create resource logger for VerrazzanoSecrets controller", err)
		return newRequeueWithDelay(), err
	}
	r.log = log
	return ctrl.Result{}, nil
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(3, 5, time.Second)
}
