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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *VerrazzanoSecretsReconciler) reconcileHelmOverrideSecret(ctx context.Context, req ctrl.Request, vz *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	// TODO List (cont):
	// 3. Update the Verrazzano CR to start a helm upgrade command
	//      a) Update the status.ReconcileGeneration for the prometheus operator
	//	    b) as an example: vz.Status.Components["prometheus-operator"].LastReconciledGeneration = 0 (it should be component generic)
	// 4. Create unit tests for new functions

	secret := &corev1.Secret{}
	if vz.Namespace == req.Namespace {
		if err := r.Get(ctx, req.NamespacedName, secret); err != nil {
			zap.S().Errorf("Failed to fetch ConfigMap in Verrazzano CR namespace: %v", err)
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
			err := r.updateVerrazzanoForHelmOverrides(componentCtx, componentName)
			if err != nil {
				r.log.Errorf("Failed to reconcile ConfigMap: %v", err)
				return newRequeueWithDelay(), err
			}
		}
	}
	return ctrl.Result{}, nil
}

func (r *VerrazzanoSecretsReconciler) updateVerrazzanoForHelmOverrides(componentCtx spi.ComponentContext, componentName string) error {
	cr := componentCtx.ActualCR()
	_, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, cr, func() error {
		cr.Status.Components[componentName].LastReconciledGeneration = 0
		return nil
	})
	if err == nil {
		return nil
	}
	return err
}
