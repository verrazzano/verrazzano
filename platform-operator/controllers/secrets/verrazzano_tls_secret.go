// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"

	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// reconcileVerrazzanoTLS reconciles secret containing the admin ca bundle in the Multi Cluster namespace
func (r *VerrazzanoSecretsReconciler) reconcileVerrazzanoTLS(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	isVzIngressSecret := isVerrazzanoIngressSecretName(req.NamespacedName)
	isAddnlTLSSecret := isAdditionalTLSSecretName(req.NamespacedName)

	caKey := "ca.crt"
	if isAddnlTLSSecret {
		caKey = vzconst.AdditionalTLSCAKey
	}

	if isVzIngressSecret && r.additionalTLSSecretExists() {
		// When the additional TLS secret exists, it is considered the source of truth - ignore
		// reconciles for the VZ Ingress TLS secret in that case
		return ctrl.Result{}, nil
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
	zap.S().Debugf("Fetched secret %s/%s ", req.NamespacedName.Namespace, req.NamespacedName.Name)

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	if result, err := r.initLogger(caSecret); err != nil {
		return result, err
	}

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
		zap.S().Debugf("Updating MC CA secret with data from %s key of %s/%s secret ", caKey, caSecret.Namespace, caSecret.Name)
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
