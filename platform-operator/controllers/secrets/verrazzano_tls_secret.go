// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *VerrazzanoSecretsReconciler) reconcileVerrazzanoTLS(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Get the verrazzano ingress secret
	caSecret := corev1.Secret{}
	err := r.Get(ctx, req.NamespacedName, &caSecret)
	if err != nil {
		// Secret should never be not found, unless we're running while installation is still underway
		zap.S().Errorf("Failed to fetch secret %s/%s: %v",
			constants.VerrazzanoSystemNamespace, constants.VerrazzanoIngressSecret, err)
		return newRequeueWithDelay(), nil
	}

	// Initialize the logger with the Secret details
	if result, err := r.initLogger(caSecret); err != nil {
		return result, err
	}

	// Get the local ca-bundle secret
	mcCASecret := corev1.Secret{}
	err = r.Get(ctx, client.ObjectKey{
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

	result, err := controllerutil.CreateOrUpdate(ctx, r.Client, &mcCASecret, func() error {
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
