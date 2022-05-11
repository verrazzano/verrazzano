// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"

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
		if err := r.Get(ctx, req.NamespacedName, secret); err != nil {
			zap.S().Errorf("Failed to fetch Secret in Verrazzano CR namespace: %v", err)
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
			err := controllers.UpdateVerrazzanoForHelmOverrides(r.Client, componentCtx, componentName)
			if err != nil {
				r.log.Errorf("Failed to reconcile ConfigMap: %v", err)
				return newRequeueWithDelay(), err
			}
			r.log.Infof("Updated Verrazzano Resource")
		}
	}
	return ctrl.Result{}, nil
}
