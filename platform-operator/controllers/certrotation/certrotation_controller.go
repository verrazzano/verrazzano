// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certrotation

import (
	"context"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	vzcert "github.com/verrazzano/verrazzano/platform-operator/internal/k8s/certificate"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

const (
	controllerName     = "CertificateRotationManager"
	componentNamespace = constants.VerrazzanoInstallNamespace
	componentName      = "certrotation"
)

// CertificateRotationManagerReconciler reconciles certificate secrets.
type CertificateRotationManagerReconciler struct {
	clipkg.Client
	Scheme           *runtime.Scheme
	StatusUpdater    vzstatus.Updater
	log              vzlog.VerrazzanoLogger
	WatchNamespace   string
	CertificatesList []string
	TargetNamespace  string
	TargetDeployment string
	CompareWindow    time.Duration // should be in no of Seconds
	CheckPeriod      time.Duration
	lastReconcile    *time.Time
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *CertificateRotationManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).WithEventFilter(r.createComponentCertificatePredicate()).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}

// Reconcile the Certificate Secret
func (r *CertificateRotationManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lastReconcile := time.Now()
	if r.lastReconcile != nil {
		r.log.Debugf("Reconciling %v, time between reconciles: %v", req.NamespacedName, lastReconcile.Sub(*r.lastReconcile))
	}
	r.lastReconcile = &lastReconcile
	if ctx == nil {
		ctx = context.TODO()
	}
	if result, err := r.initLogger(componentNamespace); err != nil {
		return result, err
	}
	certList, err := r.getCertSecretList(ctx)
	if err != nil {
		// An error occurred requeue and try again
		return newRequeueWithDelay(30, 60, time.Second), err
	}
	if len(certList) < len(r.CertificatesList) {
		// If there are no matching certificates, we don't need to re-queue
		r.log.Debugf("No matching certificate secrets found")
		return newRequeueWithDelay(30, 60, time.Second), nil
	}

	err = r.CheckCertificateExpiration(ctx, certList)
	if err != nil {
		result := newRequeueWithDelay(5, 10, time.Second)
		r.log.Debugf("Error while checking certificate status: %v", result.RequeueAfter)
		return result, err
	}

	result := ctrl.Result{
		Requeue:      true,
		RequeueAfter: r.CheckPeriod, // Check period is in duration already
	}
	r.log.Infof("Successfully validated webhook certificates: %v", result.RequeueAfter)
	return result, nil

}

// initialize secret logger
func (r *CertificateRotationManagerReconciler) initLogger(secretNamespace string) (ctrl.Result, error) {
	// Get the resource logger needed to log message using 'progress' and 'once' methods
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           componentName,
		Namespace:      componentNamespace,
		ID:             secretNamespace,
		Generation:     0,
		ControllerName: controllerName,
	})
	if err != nil {
		zap.S().Errorf("Failed to create resource logger for CertificateRotationManager controller: %v", err)
		return newRequeueWithDelay(3, 5, 5*time.Second), err
	}
	r.log = log
	return ctrl.Result{}, nil
}

// Create a new Result that will cause a reconcile requeue after a short delay
func newRequeueWithDelay(min, max int, unit time.Duration) ctrl.Result {
	return vzctrl.NewRequeueWithDelay(min, max, unit)
}

func (r *CertificateRotationManagerReconciler) CheckCertificateExpiration(ctx context.Context, certsList []string) error {
	mustRotate := false
	var err error
	status := make(map[string]bool)

	for i := range certsList {
		secret := certsList[i]
		r.log.Debugf("secret/certificate found %v", secret)
		sec, secdata := r.GetSecretData(ctx, secret)
		if secdata == nil {
			return r.log.ErrorfNewErr("an error occurred obtaining certificate data for %s", secret)
		}
		mustRotate, err = r.ValidateCertDate(secdata)
		if err != nil {
			return r.log.ErrorfNewErr("an error while validating the certificate secret data")
		}
		r.log.Debugf("cert data expiry status for secret %v is %v", secret, mustRotate)
		status[secret] = mustRotate
		if mustRotate {
			err = r.DeleteSecret(ctx, sec)
			if err != nil {
				return r.log.ErrorfNewErr("an error deleting the certificate")
			}
		}
	}
	for s := range status {
		if status[s] {
			err = r.RolloutRestartDeployment(ctx)
			if err != nil {
				return err
			}
			break
		}
	}
	return nil
}

func (r *CertificateRotationManagerReconciler) getCertSecretList(ctx context.Context) ([]string, error) {
	certificates := make([]string, 0)
	secretList := corev1.SecretList{}
	listOptions := &clipkg.ListOptions{Namespace: r.WatchNamespace}
	err := r.List(ctx, &secretList, listOptions)
	if err != nil {
		return nil, r.log.ErrorfNewErr("an error while listing the certificate sceret")
	}
	for _, secret := range secretList.Items {
		if secret.Type == corev1.SecretTypeTLS {
			certificates = append(certificates, secret.Name)
		}
	}
	if len(certificates) > 0 {
		return certificates, nil
	}
	r.log.Debugf("No certificates found in namespace %v", r.WatchNamespace)
	return certificates, nil
}

func (r *CertificateRotationManagerReconciler) createComponentCertificatePredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return r.isCertificateSecret(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.isCertificateSecret(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.isCertificateSecret(e.ObjectNew)
		},
	}
}

func (r *CertificateRotationManagerReconciler) isCertificateSecret(o clipkg.Object) bool {
	certificate := o.(*corev1.Secret)
	return certificate.Labels[vzcert.OperatorCertLabelKey] == vzcert.OperatorCertLabel &&
		certificate.Labels[vzcert.OperatorCertLabelKey] != ""
}
