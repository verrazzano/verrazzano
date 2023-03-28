// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certrotation

import (
	"context"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
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
	TargetNamespace  string
	TargetDeployment string
	CompareWindow    time.Duration // should be in no of hours
}

// SetupWithManager creates a new controller and adds it to the manager
func (r *CertificateRotationManagerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(r.createComponentCertificatePredicate()).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: 1,
		}).
		Complete(r)
}

// Reconcile the Certificate Secret
func (r *CertificateRotationManagerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if ctx == nil {
		ctx = context.TODO()
	}
	if result, err := r.initLogger(componentNamespace); err != nil {
		return result, err
	}
	// If no error during certification checks, then next reconcile will happen
	// every alternative day.
	// else in case of error it will happen after 5 mintues.
	err := r.CheckCertificateExpiration(ctx)
	if err != nil {
		return newRequeueWithDelay(3, 5, time.Minute), nil
	}
	return newRequeueWithDelay(3, 24, time.Hour), err

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

func (r *CertificateRotationManagerReconciler) CheckCertificateExpiration(ctx context.Context) error {
	mustRotate := false
	var err error
	var certsList []string
	if certsList, err = r.getCertSecretList(ctx); err != nil {
		return err
	}
	for i := range certsList {
		secret := certsList[i]
		r.log.Debugf("secret/certificate found %v", secret)
		sec, secdata := r.GetSecretData(ctx, secret)
		if secdata == nil {
			return r.log.ErrorfNewErr("an error occurred obtaining certificate data for %s", secret)
		}
		mustRotate, err = r.ValidateCertDate(secdata)
		r.log.Debugf("cert data expiry status for secret %v is %v", secret, mustRotate)
		if err != nil {
			return r.log.ErrorfNewErr("an error while validating the certificate secret data")
		}
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
	return nil, r.log.ErrorfNewErr("no certificate found in namespace %v", r.WatchNamespace)
}

func (r *CertificateRotationManagerReconciler) createComponentCertificatePredicate() predicate.Predicate {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return r.isComponentNamespace(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return r.isComponentNamespace(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return r.isComponentNamespace(e.ObjectNew)
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return r.isComponentNamespace(genericEvent.Object)
		},
	}
}

func (r *CertificateRotationManagerReconciler) isComponentNamespace(o clipkg.Object) bool {
	secret := o.(*corev1.Secret)
	return secret.Namespace == r.WatchNamespace
}
