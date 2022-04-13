// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

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
	if req.Name == constants.VerrazzanoIngressSecret && req.Namespace == constants.VerrazzanoSystemNamespace {
		zap.S().Info("Reconciling VerrazzanoAdminCA")

		// Get the verrazzano ingress secret
		caSecret := corev1.Secret{}
		if err := r.Get(context.TODO(), req.NamespacedName, &caSecret); err != nil {
			// Secret should never be not found, unless we're running before it's been created
			zap.S().Errorf("Failed to fetch Verrazzano ingress secret: %v", err)
			return newRequeueWithDelay(), nil
		}
		zap.S().Info("Got admin secret")

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
		}
		zap.S().Info("Got logger")

		r.log = log

		mcCASecret := corev1.Secret{}
		mcCASecret.Data = make(map[string][]byte)
		mcCASecret.Data["ca-bundle"] = caSecret.Data["ca.crt"]
		mcCASecret.Name = constants.MCAdminCASecret
		mcCASecret.Namespace = constants.VerrazzanoMultiClusterNamespace

		_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, &mcCASecret, func() error { return nil })
		if err != nil {
			r.log.Errorf("Failed to create or update MC admin ca-bundle secret: %v", err)
			return newRequeueWithDelay(), nil
		}
		zap.S().Info("Created or updated MC admin ca-bundle secret")

		// The resource has been reconciled.
		r.log.Infof("Successfully reconciled Verrazzano ingress secret")
	} else {
		zap.S().Infof("Ignoring reconcile for secret: %v", req.NamespacedName)
	}

	return ctrl.Result{}, nil
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(3, 5, time.Second)
}
