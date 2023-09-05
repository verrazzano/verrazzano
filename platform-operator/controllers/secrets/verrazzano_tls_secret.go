// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"fmt"
	"time"

	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const rancherDeploymentName = "rancher"
const mcCABundleKey = "ca-bundle"

var fetchSecretFailureTemplate = "Failed to fetch secret %s/%s: %v"

// reconcileVerrazzanoTLS - Update the Verrazzano private CA bundle
func (r *VerrazzanoSecretsReconciler) reconcileVerrazzanoTLS(ctx context.Context, secret types.NamespacedName, caKey string) (ctrl.Result, error) {

	// Get the secret
	caSecret := corev1.Secret{}
	if err := r.Get(ctx, secret, &caSecret); err != nil {
		if apierrors.IsNotFound(err) {
			// Secret may have been deleted, skip reconcile
			zap.S().Infof("Secret %s does not exist, skipping reconcile", secret)
			return ctrl.Result{}, nil
		}
		// Secret should never be not found, unless we're running while installation is still underway
		zap.S().Errorf(fetchSecretFailureTemplate,
			secret.Namespace, secret.Name, err)
		return newRequeueWithDelay(), nil
	}
	zap.S().Debugf("Fetched secret %s/%s ", secret.Namespace, secret.Name)

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	if result, err := r.initLogger(secret, &caSecret); err != nil {
		return result, err
	}

	// Update the Verrazzano private CA bundle; the source of truth from a VZ perspective
	_, err := r.updateSecret(vzconst.VerrazzanoSystemNamespace, vzconst.PrivateCABundle,
		vzconst.CABundleKey, caKey, &caSecret, false)
	if err != nil {
		return newRequeueWithDelay(), nil
	}
	return ctrl.Result{}, nil
}

// reconcileVerrazzanoCABundleCopies - The Verrazzano private CA secret has changed. Propagate that change into the following:
//   - The Rancher TLS CA secret and restart Rancher deployment
//   - The multi-cluster verrazzano-local-ca-bundle secret which maintains a copy of the local CA bundle to sync with remote clusters in the multi-cluster case
//
// Certs issued from ACME issuers do not populate the "ca.crt" field in leaf cert secrets.  In those scenarios those copies are set up
// once during VZ resource reconciliation until if/when the VZ issuer configuration is changed.
//
// These copies are only maintained when private CA configurations are involved; self-signed, custom CA, and Let's Encrypt staging configurations
func (r *VerrazzanoSecretsReconciler) reconcileVerrazzanoCABundleCopies() (ctrl.Result, error) {
	// Use private bundle secret to update copies
	privateBundleSecret := &corev1.Secret{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: vzconst.PrivateCABundle, Namespace: vzconst.VerrazzanoSystemNamespace}, privateBundleSecret)
	if otherErr := client.IgnoreNotFound(err); otherErr != nil {
		return newRequeueWithDelay(), otherErr
	}

	// Get the resource logger needed to log message using 'progress' and 'once' methods
	if result, err := r.initLogger(client.ObjectKeyFromObject(privateBundleSecret), privateBundleSecret); err != nil {
		return result, err
	}

	// Update the Rancher TLS CA secret
	result, err := r.updateSecret(vzconst.RancherSystemNamespace, vzconst.RancherTLSCA,
		vzconst.RancherTLSCAKey, vzconst.CABundleKey, privateBundleSecret, false)
	if err != nil {
		return newRequeueWithDelay(), nil
	}

	// Restart Rancher pod to have the updated CA secret value reflected in the pod
	if result == controllerutil.OperationResultUpdated {
		if err := r.restartRancherPod(); err != nil {
			return newRequeueWithDelay(), err
		}
	}

	if !r.multiclusterNamespaceExists() {
		// Multicluster namespace doesn't exist yet, nothing to do so requeue
		return newRequeueWithDelay(), nil
	}

	// Update the verrazzano-local-ca-bundle secret
	if _, err := r.updateSecret(constants.VerrazzanoMultiClusterNamespace, constants.VerrazzanoLocalCABundleSecret,
		mcCABundleKey, vzconst.CABundleKey, privateBundleSecret, true); err != nil {
		return newRequeueWithDelay(), nil
	}
	return ctrl.Result{}, nil
}

func (r *VerrazzanoSecretsReconciler) updateSecret(namespace string, name string, destCAKey string,
	sourceCAKey string, sourceSecret *corev1.Secret, isCreateAllowed bool) (controllerutil.OperationResult, error) {
	// Get the secret
	secret := corev1.Secret{}
	err := r.Get(context.TODO(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &secret)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			r.log.Errorf(fetchSecretFailureTemplate, namespace, name, err)
			return controllerutil.OperationResultNone, err
		}
		if !isCreateAllowed {
			if r.log != nil {
				r.log.Debugf("Secret %s/%s not found, nothing to do", namespace, name)
			}
			return controllerutil.OperationResultNone, nil
		}
		// Secret was not found, make a new one
		secret = corev1.Secret{}
		secret.Name = name
		secret.Namespace = namespace
	}

	result, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		// We only want to update the target secret IFF the secret/key in the source secret/key exist;
		// we are keeping private CA bundles in sync on rotation only, the modules manage the lifecycle
		// of the target secrets on reconcile of the VZ CR
		sourceBundle, exists := sourceSecret.Data[sourceCAKey]
		if !exists && !isCreateAllowed {
			zap.S().Debugf("Source key %s does not exist in secret %s/%s, nothing to do ", sourceCAKey,
				sourceSecret.Namespace, sourceSecret.Name)
			return nil
		}
		zap.S().Debugf("Updating CA secret with data from %s key of %s/%s secret ", sourceCAKey,
			sourceSecret.Namespace, sourceSecret.Name)
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data[destCAKey] = sourceBundle
		return nil
	})

	if err != nil {
		r.log.ErrorfThrottled("Failed to create or update secret %s/%s: %s", name, namespace, err.Error())
		return controllerutil.OperationResultNone, err
	}

	r.log.Debugf("Created or updated secret %s/%s (result: %v)", name, namespace, result)
	return result, nil
}

// restartRancherPod adds an annotation to the Rancher deployment template to restart the Rancher pods
func (r *VerrazzanoSecretsReconciler) restartRancherPod() error {
	deployment := appsv1.Deployment{}
	if err := r.Get(context.TODO(), types.NamespacedName{Namespace: vzconst.RancherSystemNamespace,
		Name: rancherDeploymentName}, &deployment); err != nil {
		if apierrors.IsNotFound(err) {
			r.log.Debugf("Rancher deployment %s/%s not found, nothing to do",
				vzconst.RancherSystemNamespace, rancherDeploymentName)
			return nil
		}
		r.log.ErrorfThrottled("Failed getting Rancher deployment %s/%s to restart pod: %v",
			vzconst.RancherSystemNamespace, rancherDeploymentName, err)
		return err
	}

	// annotate the deployment to do a restart of the pod
	if deployment.Spec.Template.ObjectMeta.Annotations == nil {
		deployment.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	deployment.Spec.Template.ObjectMeta.Annotations[vzconst.VerrazzanoRestartAnnotation] = time.Now().String()

	if err := r.Update(context.TODO(), &deployment); err != nil {
		return log.ConflictWithLog(fmt.Sprintf("Failed updating deployment %s/%s", deployment.Namespace, deployment.Name), err, zap.S())
	}
	r.log.Infof("Updated Rancher deployment %s/%s with restart annotation to force a pod restart",
		deployment.Namespace, deployment.Name)
	return nil
}
