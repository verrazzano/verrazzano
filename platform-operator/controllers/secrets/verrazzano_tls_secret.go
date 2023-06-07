// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"fmt"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzlog "github.com/verrazzano/verrazzano/pkg/log"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"

	"github.com/verrazzano/verrazzano/platform-operator/constants"

	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const rancherDeploymentName = "rancher"

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

	// Update the Rancher TLS CA secret with the CA in Verrazzano TLS Secret
	if isVzIngressSecret {
		result, err := r.updateSecret(vzconst.RancherSystemNamespace, vzconst.RancherTLSCA,
			vzconst.RancherTLSCAKey, caKey, caSecret, false)
		if err != nil {
			return newRequeueWithDelay(), nil
		}
		// Restart Rancher pod to have the updated TLS CA secret value reflected in the pod
		if result == controllerutil.OperationResultUpdated {
			if err := r.restartRancherPod(); err != nil {
				return newRequeueWithDelay(), err
			}
		}
	}
	// Update the verrazzano-local-ca-bundle secret
	if _, err := r.updateSecret(constants.VerrazzanoMultiClusterNamespace, constants.VerrazzanoLocalCABundleSecret,
		"ca-bundle", caKey, caSecret, true); err != nil {
		return newRequeueWithDelay(), nil
	}
	return ctrl.Result{}, nil
}

func (r *VerrazzanoSecretsReconciler) updateSecret(namespace string, name string, destCAKey string,
	sourceCAKey string, sourceSecret corev1.Secret, isCreate bool) (controllerutil.OperationResult, error) {
	// Get the secret
	secret := corev1.Secret{}
	err := r.Get(context.TODO(), client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &secret)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			r.log.Errorf("Failed to fetch secret %s/%s: %v", namespace, name, err)
			return controllerutil.OperationResultNone, err
		}
		if !isCreate {
			r.log.Debugf("Secret %s/%s not found, nothing to do", namespace, name)
			return controllerutil.OperationResultNone, nil
		}
		// Secret was not found, make a new one
		secret = corev1.Secret{}
		secret.Name = name
		secret.Namespace = namespace
	}

	result, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &secret, func() error {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		zap.S().Debugf("Updating CA secret with data from %s key of %s/%s secret ", sourceCAKey,
			sourceSecret.Namespace, sourceSecret.Name)
		secret.Data[destCAKey] = sourceSecret.Data[sourceCAKey]
		return nil
	})

	if err != nil {
		r.log.ErrorfThrottled("Failed to create or update secret %s/%s: %s", name, namespace, err.Error())
		return controllerutil.OperationResultNone, err
	}

	r.log.Infof("Created or updated secret %s/%s (result: %v)", name, namespace, result)
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
		return vzlog.ConflictWithLog(fmt.Sprintf("Failed updating deployment %s/%s", deployment.Namespace, deployment.Name), err, zap.S())
	}
	r.log.Infof("Updated Rancher deployment %s/%s with restart annotation to force a pod restart",
		deployment.Namespace, deployment.Name)
	return nil
}
