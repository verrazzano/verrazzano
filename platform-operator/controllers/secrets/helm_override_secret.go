// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// reconcileHelmOverrideSecret looks through the Verrazzano CR for the Secret
// if the request is from the same namespace as the CR
func (r *VerrazzanoSecretsReconciler) reconcileHelmOverrideSecret(ctx context.Context, req ctrl.Request, vz *installv1alpha1.Verrazzano) (ctrl.Result, error) {

	secret := &corev1.Secret{}
	if vz.Namespace == req.Namespace {
		// Get the secret if the request namespace matches verrazzano namespace
		if err := r.Get(ctx, req.NamespacedName, secret); err != nil {
			zap.S().Errorf("Failed to fetch Secret in Verrazzano CR namespace: %v", err)
			// Do not retry if secret is deleted
			if errors.IsNotFound(err) {
				return reconcile.Result{}, nil
			}
			return newRequeueWithDelay(), err
		}

		if result, err := r.initLogger(*secret); err != nil {
			return result, err
		}

		componentCtx, err := spi.NewContext(r.log, r.Client, vz, false)
		if err != nil {
			r.log.Errorf("Failed to construct component context: %v", err)
			return newRequeueWithDelay(), err
		}
		if componentName, ok := controllers.VzContainsResource(componentCtx, secret); ok {
			if secret.DeletionTimestamp.IsZero() {

				// Add finalizer if not added
				if !controllerutil.ContainsFinalizer(secret, constants.KubeFinalizer) {
					secret.Finalizers = append(secret.Finalizers, constants.KubeFinalizer)
					err := r.Update(context.TODO(), secret)
					if err != nil {
						return newRequeueWithDelay(), nil
					}
					return ctrl.Result{Requeue: true}, nil
				}

			} else {
				// Requeue as other finalizers haven't been removed
				if len(secret.Finalizers) > 1 {
					return newRequeueWithDelay(), nil
				}

				controllerutil.RemoveFinalizer(secret, constants.KubeFinalizer)
				err := r.Update(context.TODO(), secret)
				if err != nil {
					return newRequeueWithDelay(), err
				}
			}

			err := controllers.UpdateVerrazzanoForHelmOverrides(r.Client, componentCtx, componentName)
			if err != nil {
				r.log.Errorf("Failed to reconcile Secret: %v", err)
				return newRequeueWithDelay(), err
			}
			r.log.Infof("Updated Verrazzano Resource")
		}
	}
	return ctrl.Result{}, nil
}
