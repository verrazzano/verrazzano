// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package secrets

import (
	"context"
	"time"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/issuer"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// VerrazzanoSecretsReconciler reconciles secrets.
// One part of the controller is for the verrazzano-tls secret. The controller
// ensures that a copy of the ca.crt secret (admin CA bundle) is copied to a secret
// in the verrazzano-mc namespace, so that managed clusters can fetch it.
// This controller also manages install override sources from the Verrazzano CR
type VerrazzanoSecretsReconciler struct {
	client.Client
	DynamicClient dynamic.Interface
	Scheme        *runtime.Scheme
	log           vzlog.VerrazzanoLogger
	StatusUpdater vzstatus.Updater
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *VerrazzanoSecretsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		Complete(r)
}

// Reconcile the Secret
func (r *VerrazzanoSecretsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// One secret we care about is the verrazzano ingress tls secret (verrazzano-tls)
	if ctx == nil {
		ctx = context.TODO()
	}

	// Get the VZ
	vzList, result, err := r.getVZ(ctx)
	if err != nil {
		return result, err
	}

	// Nothing to do if no VZ returned or if it is being deleted
	if vzList == nil || len(vzList.Items) == 0 || vzList.Items[0].DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	// Get the effective CR to access the ClusterIssuer configuration
	vz := &vzList.Items[0]
	effectiveCR, err := transform.GetEffectiveCR(vz)
	if err != nil {
		zap.S().Errorf("Failed to get the effective CR for %s/%s: %s", vz.Namespace, vz.Name, err.Error())
		return newRequeueWithDelay(), err
	}

	// Renew all certificates issued by ClusterIssuer when it's secret changes
	clusterIssuer := effectiveCR.Spec.Components.ClusterIssuer
	if isClusterIssuerSecret(req.NamespacedName, clusterIssuer) {
		zap.S().Debugf("Reconciling ClusterIssuer secret %s/%s", req.Namespace, req.Name)
		if result, err = r.renewClusterIssuerCertificates(req, vz); err != nil {
			zap.S().Errorf("Failed to new all certificates issued by ClusterIssuer %s: %s", vzconst.VerrazzanoClusterIssuerName, err.Error())
			return result, err
		}
		return r.reconcileVerrazzanoTLS(ctx, req.NamespacedName, corev1.TLSCertKey)
	}

	// Handle changes to the verrazzano-tls-ca secret
	if isVerrazzanoPrivateCABundle(req.NamespacedName) {
		zap.S().Debugf("Reconciling changes to secret %s/%s", req.Namespace, req.Name)
		return r.reconcileVerrazzanoCABundleCopies()
	}

	// updates capi credential if an ocne cloud credential has been updated
	if config.Get().CloudCredentialWatchEnabled {
		caSecret := corev1.Secret{}
		if err := r.Get(ctx, req.NamespacedName, &caSecret); err != nil {
			zap.S().Errorf("Failed to get Secret: %v", err)
			return newRequeueWithDelay(), err
		}
		if isOCNECloudCredential(&caSecret) {
			// check if ocne cluster is using the updated secret and update cluster's copy of secret if necessary
			if err := r.checkClusterCredentials(&caSecret); err != nil {
				zap.S().Errorf("Failed to update Secret: %v", err)
				return newRequeueWithDelay(), err
			}
		}
	}

	res, err := r.reconcileInstallOverrideSecret(ctx, req, vz)
	if err != nil {
		zap.S().Errorf("Failed to reconcile Secret: %v", err)
		return newRequeueWithDelay(), err
	}
	return res, nil
}

func (r *VerrazzanoSecretsReconciler) getVZ(ctx context.Context) (*installv1alpha1.VerrazzanoList, ctrl.Result, error) {
	vzList := &installv1alpha1.VerrazzanoList{}
	err := r.List(ctx, vzList)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, reconcile.Result{}, nil
		}
		zap.S().Errorf("Failed to fetch Verrazzano resource: %v", err)
		return nil, newRequeueWithDelay(), err
	}
	return vzList, ctrl.Result{}, nil
}

// initialize secret logger
func (r *VerrazzanoSecretsReconciler) initLogger(nsName types.NamespacedName, obj client.Object) (ctrl.Result, error) {
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           nsName.Name,
		Namespace:      nsName.Namespace,
		ID:             string(obj.GetUID()),
		Generation:     obj.GetGeneration(),
		ControllerName: "secrets",
	})
	if err != nil {
		zap.S().Errorf("Failed to create resource logger for VerrazzanoSecrets controller: %v", err)
		return newRequeueWithDelay(), err
	}
	r.log = log
	return ctrl.Result{}, nil
}

func isVerrazzanoPrivateCABundle(secretName types.NamespacedName) bool {
	return secretName.Name == vzconst.PrivateCABundle && secretName.Namespace == constants.VerrazzanoSystemNamespace
}

func isClusterIssuerSecret(secretName types.NamespacedName, clusterIssuer *installv1alpha1.ClusterIssuerComponent) bool {
	if clusterIssuer == nil || clusterIssuer.CA == nil {
		return false
	}
	return secretName.Name == clusterIssuer.CA.SecretName && secretName.Namespace == clusterIssuer.ClusterResourceNamespace
}

func (r *VerrazzanoSecretsReconciler) multiclusterNamespaceExists() bool {
	ns := corev1.Namespace{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: constants.VerrazzanoMultiClusterNamespace}, &ns)
	if err == nil {
		return true
	}
	if !apierrors.IsNotFound(err) {
		r.log.ErrorfThrottled("Unexpected error checking for namespace %s: %v", constants.VerrazzanoMultiClusterNamespace, err)
	}
	r.log.Debugf("Namespace %s does not exist, nothing to do", constants.VerrazzanoMultiClusterNamespace)
	return false
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay() ctrl.Result {
	return vzctrl.NewRequeueWithDelay(3, 5, time.Second)
}

func (r *VerrazzanoSecretsReconciler) renewClusterIssuerCertificates(req ctrl.Request, obj client.Object) (ctrl.Result, error) {
	if result, err := r.initLogger(req.NamespacedName, obj); err != nil {
		return result, err
	}

	// List the certificates
	certList := &certv1.CertificateList{}
	if err := r.List(context.TODO(), certList); err != nil {
		return newRequeueWithDelay(), err
	}

	cmClient, err := issuer.GetCMClientFunc()()
	if err != nil {
		return newRequeueWithDelay(), err
	}

	// Renew each certificate that was issued by the Verrazzano ClusterIssuer
	for i, cert := range certList.Items {
		if cert.Spec.IssuerRef.Name == vzconst.VerrazzanoClusterIssuerName {
			// Only renew the cert if the renewal time is in the past
			if cert.Status.RenewalTime == nil || time.Now().Before(cert.Status.RenewalTime.Time) {
				continue
			}

			r.log.Infof("Renewing certificate %s/%s", cert.Namespace, cert.Name)
			if err := issuer.RenewCertificate(context.TODO(), cmClient, r.log, &certList.Items[i]); err != nil {
				return newRequeueWithDelay(), err
			}
		}
	}
	return ctrl.Result{}, nil
}
