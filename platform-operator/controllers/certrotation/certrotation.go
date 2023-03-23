// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certrotation

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

const (
	controllerName     = "CertificateRotationManager"
	channelBufferSize  = 100
	componentNamespace = constants.VerrazzanoInstallNamespace
	componentName      = "certrotation"
)

// CertificateRotationManager periodically checks the expiry of Certs.
// If certs are going to expire, then restarts the target namespace deployment
// such that new certs can be regenerated
type CertificateRotationManager struct {
	client           clipkg.Client
	tickTime         time.Duration
	log              vzlog.VerrazzanoLogger
	watchNamespace   string
	targetNamespace  string
	targetDeployment string
	compareWindow    time.Duration
	shutdown         chan int // The channel on which shutdown signals are sent/received
}

var certrotationManager *CertificateRotationManager

// CertificateRotationManager - instantiate a CertificateWatcher context
func NewCertificateRotationManager(c clipkg.Client, tick time.Duration, compareWindow time.Duration, secretNamespace, targetNamespace, targetDeployment string) (*CertificateRotationManager, error) {
	log, err := vzlog.EnsureResourceLogger(&vzlog.ResourceConfig{
		Name:           componentName,
		Namespace:      componentNamespace,
		ID:             secretNamespace,
		Generation:     0,
		ControllerName: controllerName,
	})
	if err != nil {
		zap.S().Errorf("Failed to create resource logger for %s: %v", controllerName, err)
		return nil, err
	}

	certrotationManager = &CertificateRotationManager{
		client:           c,
		tickTime:         tick,
		log:              log,
		watchNamespace:   secretNamespace,
		targetNamespace:  targetNamespace,
		targetDeployment: targetDeployment,
		compareWindow:    compareWindow,
	}
	return certrotationManager, nil
}

// Start starts the CertificateWatcher if it is not already running.
// It is safe to call Start multiple times, additional goroutines will not be created
func (sw *CertificateRotationManager) Start() {
	if sw.shutdown != nil {
		// already running, so nothing to do
		return
	}
	sw.shutdown = make(chan int, channelBufferSize)
	// goroutine updates availability every p.tickTime. If a shutdown signal is received (or channel is closed),
	// the goroutine returns.
	go func() {
		var _ error
		ticker := time.NewTicker(sw.tickTime)
		for {
			select {
			case <-ticker.C:
				if err := sw.CheckCertificateExpiration(); err != nil {
					sw.log.ErrorfThrottled("Failed to sync: %v", err)
				}
			case <-sw.shutdown:
				// shutdown event causes termination
				ticker.Stop()
				return
			}
		}
	}()
}

func (sw *CertificateRotationManager) CheckCertificateExpiration() error {
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
				return fmt.Errorf("an error occurred restarting the deployment %v in namespace %v", sw.targetDeployment, sw.targetNamespace)
			}
		}
	}
	return nil
}

func (sw *CertificateRotationManager) GetCertificateList() ([]string, error) {
	certificates := make([]string, 0)
	secretList := corev1.SecretList{}
	listOptions := &clipkg.ListOptions{Namespace: sw.watchNamespace}
	err := sw.client.List(context.TODO(), &secretList, listOptions)
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
	return nil, fmt.Errorf("no certificate found in namespace %v", sw.watchNamespace)
}
