// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// reconcileInstallOverrideSecret looks through the Verrazzano CR for the Secret
// if the request is from the same namespace as the CR
func (r *VerrazzanoSecretsReconciler) reconcileInstallOverrideSecret(ctx context.Context, req ctrl.Request, vz *installv1alpha1.Verrazzano) (ctrl.Result, error) {

	secret := &corev1.Secret{}
	if vz.Namespace == req.Namespace {
		// Get the secret if the request namespace matches verrazzano namespace
		if err := r.Get(ctx, req.NamespacedName, secret); err != nil {
			// Do not retry if secret is deleted
			if errors.IsNotFound(err) {
				if err := controllers.ProcDeletedOverride(r.Client, vz, req.Name, constants.SecretKind); err != nil {
					// Do not return an error as it's most likely due to timing
					return newRequeueWithDelay(), nil
				}
				return reconcile.Result{}, nil
			}
			zap.S().Errorf("Failed to fetch Secret in Verrazzano CR namespace: %v", err)
			return newRequeueWithDelay(), err
		}

		if result, err := r.initLogger(*secret); err != nil {
			return result, err
		}

		componentCtx, err := spi.NewContext(r.log, r.Client, vz, nil, false)
		if err != nil {
			r.log.Errorf("Failed to construct component context: %v", err)
			return newRequeueWithDelay(), err
		}

		if componentName, ok := controllers.VzContainsResource(componentCtx, secret.Name, secret.Kind); ok {
			if secret.DeletionTimestamp.IsZero() {

				// Add finalizer if not added
				if !controllerutil.ContainsFinalizer(secret, constants.OverridesFinalizer) {
					secret.Finalizers = append(secret.Finalizers, constants.OverridesFinalizer)
					err := r.Update(context.TODO(), secret)
					if err != nil {
						return newRequeueWithDelay(), nil
					}
					return reconcile.Result{Requeue: true}, nil
				}

			} else {
				// Requeue as other finalizers haven't been removed
				if secret.Finalizers != nil && !controllerutil.ContainsFinalizer(secret, constants.OverridesFinalizer) {
					return reconcile.Result{Requeue: true}, nil
				}

				controllerutil.RemoveFinalizer(secret, constants.OverridesFinalizer)
				err := r.Update(context.TODO(), secret)
				if err != nil {
					return newRequeueWithDelay(), err
				}
			}

			err := controllers.UpdateVerrazzanoForInstallOverrides(r.Client, componentCtx, componentName)
			if err != nil {
				r.log.ErrorfThrottled("Failed to reconcile Secret: %v", err)
				return newRequeueWithDelay(), err
			}
			r.log.Infof("Updated Verrazzano Resource")
		}
	}
	return ctrl.Result{}, nil
}
