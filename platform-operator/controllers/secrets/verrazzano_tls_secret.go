// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"fmt"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
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
func (r *VerrazzanoSecretsReconciler) reconcileVerrazzanoTLS(ctx context.Context, req ctrl.Request, vz *vzapi.Verrazzano) (ctrl.Result, error) {

	if vz.Status.State != vzapi.VzStateReady {
		vzlog.DefaultLogger().Progressf("Verrazzano state is %s, CA secrets reconciling paused", vz.Status.State)
		return vzctrl.NewRequeueWithDelay(10, 30, time.Second), nil
	}

	isVzIngressSecret := isVerrazzanoIngressSecretName(req.NamespacedName)
	isAddnlTLSSecret := isAdditionalTLSSecretName(req.NamespacedName)

	caKey := "ca.crt"
	if isAddnlTLSSecret {
		caKey = vzconst.AdditionalTLSCAKey
	}

	// Get the secret
	caSecret := corev1.Secret{}
	if err := r.Get(context.TODO(), req.NamespacedName, &caSecret); err != nil {
		if apierrors.IsNotFound(err) {
			// Secret may have been deleted, skip reconcile
			zap.S().Infof("Secret %s does not exist, skipping reconcile", req.NamespacedName)
			return ctrl.Result{}, nil
		}
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

	componentCtx, err := spi.NewContext(r.log, r.Client, vz, nil, false)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	if isVzIngressSecret && r.isLetsEncryptStaging(componentCtx.EffectiveCR()) {
		// When Let's Encrypt staging is configured, do not update the tls-ca secret
		r.log.Infof("Using Let's Encrypt staging, skipping update of Rancher bundle")
		return ctrl.Result{}, nil
	}

	if !r.multiclusterNamespaceExists() {
		// Multicluster namespace doesn't exist yet, nothing to do so requeue
		return newRequeueWithDelay(), nil
	}

	// Update the Rancher TLS CA secret with the CA in Verrazzano TLS Secret
	if isVzIngressSecret {
		// Update the Verrazzano private CA bundle; source of truth from a VZ perspective
		_, err := r.updateSecret(vzconst.VerrazzanoSystemNamespace, vzconst.PrivateCABundle,
			vzconst.CABundleKey, caKey, caSecret, false)
		if err != nil {
			return newRequeueWithDelay(), nil
		}

		// Update the Rancher copy and bounce the deployment if necessary
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

func (r *VerrazzanoSecretsReconciler) isLetsEncryptStaging(effectiveCR *vzapi.Verrazzano) bool {
	if isLEConfig, _ := vzcr.IsLetsEncryptConfig(effectiveCR); isLEConfig {
		return vzcr.IsLetsEncryptStagingEnv(*effectiveCR.Spec.Components.ClusterIssuer.LetsEncrypt)
	}
	return false
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
		return log.ConflictWithLog(fmt.Sprintf("Failed updating deployment %s/%s", deployment.Namespace, deployment.Name), err, zap.S())
	}
	r.log.Infof("Updated Rancher deployment %s/%s with restart annotation to force a pod restart",
		deployment.Namespace, deployment.Name)
	return nil
}
