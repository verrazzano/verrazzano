// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certrotation

import (
	"context"
	"fmt"
	vzctrl "github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	//"sigs.k8s.io/cluster-api/cmd/clusterctl/client"
	ctrl "sigs.k8s.io/controller-runtime"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
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
	// else in case of error it will happend after 5 mintues.
	if err := r.CheckCertificateExpiration(); err != nil {
		return newRequeueWithDelay(3, 5, 5*time.Minute), nil
	} else {
		return newRequeueWithDelay(3, 5, 24*time.Hour), err
	}
	return ctrl.Result{}, nil
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

func (sw *CertificateRotationManagerReconciler) CheckCertificateExpiration() error {
	status := false
	var err error
	var certsList []string
	if certsList, err = sw.GetCertificateList(); err != nil {
		return err
	}
	for i := range certsList {
		secret := certsList[i]
		sw.log.Debugf("secret/certificate found %v", secret)
		sec, secdata := sw.GetSecretData(secret)
		if secdata == nil {
			return fmt.Errorf("an error occurred obtaining certificate data for %s", secret)
		}
		status, err = sw.ValidateCertDate(secdata)
		sw.log.Debugf("cert data expiry status for secret %v", secret)
		if err != nil {
			return fmt.Errorf("an error while validating the certificate secret data")
		}
		if status {
			err = sw.DeleteSecret(sec)
			if err != nil {
				return fmt.Errorf("an error deleting the certificate")
			}
			err = sw.RolloutRestartDeployment()
			if err != nil {
				return fmt.Errorf("an error occurred restarting the deployment %v in namespace %v", sw.TargetDeployment, sw.TargetNamespace)
			}
		}
	}
	return nil
}

func (sw *CertificateRotationManagerReconciler) GetCertificateList() ([]string, error) {
	certificates := make([]string, 0)
	secretList := corev1.SecretList{}
	listOptions := &clipkg.ListOptions{Namespace: sw.WatchNamespace}
	err := sw.List(context.TODO(), &secretList, listOptions)
	if err != nil {
		return nil, fmt.Errorf("an error while listing the certificate sceret")
	}
	for _, secret := range secretList.Items {
		if secret.Type == corev1.SecretTypeTLS {
			certificates = append(certificates, secret.Name)
		}
	}
	if len(certificates) > 0 {
		return certificates, nil
	}
	return nil, fmt.Errorf("no certificate found in namespace %v", sw.WatchNamespace)
}
