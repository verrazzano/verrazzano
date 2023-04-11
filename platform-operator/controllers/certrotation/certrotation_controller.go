// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certrotation

import (
	"context"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	string2 "github.com/verrazzano/verrazzano/pkg/string"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
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
		r.log.Infof("Reconciling %v, time between reconciles: %v", req.NamespacedName, lastReconcile.Sub(*r.lastReconcile))
	}
	r.lastReconcile = &lastReconcile
	if ctx == nil {
		ctx = context.TODO()
	}
	if result, err := r.initLogger(componentNamespace); err != nil {
		return result, err
	}

	//secret := &corev1.Secret{}
	//if err := r.Get(ctx, req.NamespacedName, secret); err != nil {
	//	if errors.IsNotFound(err) {
	//		r.log.Infof("Secret %v was deleted, restart deployment to regenerate certs", req.NamespacedName)
	//		// certificate secret was deleted, rotate
	//		r.RolloutRestartDeployment(ctx)
	//		return ctrl.Result{}, nil
	//	}
	//	return newRequeueWithDelay(5, 10, time.Second), err
	//}
	//if !secret.GetDeletionTimestamp().IsZero() {
	//	r.log.Infof("Secret %v was deleted, restart deployment to regenerate certs", req.NamespacedName)
	//	// certificate secret was deleted, rotate
	//	r.RolloutRestartDeployment(ctx)
	//	return ctrl.Result{}, nil
	//}

	// If no error during certification checks, then next reconcile will happen
	// every alternative day.
	// else in case of error it will happen after 5 mintues.
	certList, err := r.getCertSecretList(ctx)
	if err != nil {
		//r.log.Debugf("Error listing certificate secrets, skipping reconcile for %v, error: %s", req.NamespacedName, err.Error())
		// An error occurred requeue and try again
		return newRequeueWithDelay(5, 10, time.Second), err
	}
	if len(certList) == 0 {
		// If there are no matching certificates, we don't need to re-queue
		r.log.Debugf("No matching certificate secrets found")
		return ctrl.Result{}, nil
	}

	err = r.CheckCertificateExpiration(ctx, certList)
	if err != nil {
		result := newRequeueWithDelay(5, 10, time.Second)
		r.log.Infof("Delay: %v", result.RequeueAfter)
		return result, err
	}

	result := ctrl.Result{
		Requeue:      true,
		RequeueAfter: r.CheckPeriod, // Check period is in duration already
	}
	r.log.Infof("Delay: %v", result.RequeueAfter)
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
	//var certsList []string
	//if certsList, err = r.getCertSecretList(ctx); err != nil {
	//	return err
	//}
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

		if mustRotate {
			err = r.DeleteSecret(ctx, sec)
			if err != nil {
				return r.log.ErrorfNewErr("an error deleting the certificate")
			}
			err = r.RolloutRestartDeployment(ctx)
			if err != nil {
				return err
			}
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
	r.log.Infof("No certificates found in namespace %v", r.WatchNamespace)
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
			return r.isCertificateSecret(e.ObjectOld)
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			namespace := genericEvent.Object.GetNamespace()
			return r.WatchNamespace == namespace
		},
	}
}

func (r *CertificateRotationManagerReconciler) isCertificateSecret(o clipkg.Object) bool {
	if o == nil {
		return false
	}
	return r.WatchNamespace == o.GetNamespace() && string2.SliceContainsString(r.CertificatesList, o.GetName())
}
