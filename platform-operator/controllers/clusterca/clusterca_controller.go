// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusterca

import (
	"context"
	"time"

	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// VerrazzanoAdminCAReconciler reconciles a the Admin CA bundle by copying it
// to a secret in the verrazzano-mc namespace, so managed clusters can pull it.
type VerrazzanoAdminCAReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	log    vzlog.VerrazzanoLogger
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoAdminCAReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Complete(r)
}

func (r *VerrazzanoAdminCAReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Determine if we're interested in this secret
	if req.Name == constants.VerrazzanoIngressSecret && req.Namespace == constants.VerrazzanoSystemNamespace {
		// Get the verrazzano ingress secret
		caSecret := corev1.Secret{}
		if err := r.Get(context.TODO(), req.NamespacedName, &caSecret); err != nil {
			// Secret should never be not found, unless we're running before it's been created
			r.log.Errorf("Failed to fetch Verrazzano Admin CA secret: %v", err)
			return newRequeueWithDelay(), nil
		}

		mcCASecret := corev1.Secret{}
		mcCASecret.Data["ca-bundle"] = caSecret.Data["ca.crt"]
		mcCASecret.Name = constants.MCAdminCASecret
		mcCASecret.Namespace = constants.VerrazzanoMultiClusterNamespace

		_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &mcCASecret, func() error { return nil })
		if err != nil {
			r.log.Errorf("Failed to create or update MC Admin CA secret: %v", err)
			return newRequeueWithDelay(), nil
		}

		// The resource has been reconciled.
		r.log.Oncef("Successfully reconciled Admin CA secret")
	}

	return ctrl.Result{}, nil
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(2, 3, time.Second)
}
