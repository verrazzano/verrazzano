// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificatewatcher

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"

	"time"
)

const (
	controllerName     = "CertificateWatcher"
	expiry             = 1 * 7 * 24 * time.Hour // one week as grace period
	channelBufferSize  = 100
	componentNamespace = "verrazaano-install"
	componentName      = "certificatewatcher"
)

// CertificateWatcher periodically checks the expiry of Certs.
// If vpo certs are going to expire, then restarts the vpo-webhook
type CertificateWatcher struct {
	client           clipkg.Client
	tickTime         time.Duration
	log              vzlog.VerrazzanoLogger
	watchNamespace   string
	targetNamespace  string
	targetDeployment string
	shutdown         chan int // The channel on which shutdown signals are sent/received
}

var certficateWatcher *CertificateWatcher

// CertificateWatcher - instantiate a CertificateWatcher context
func NewCertificateWatcher(c clipkg.Client, tick time.Duration, secretNamespace, targetNamespace, targetDeployment string) (*CertificateWatcher, error) {
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

	certficateWatcher = &CertificateWatcher{
		client:           c,
		tickTime:         tick,
		log:              log,
		watchNamespace:   secretNamespace,
		targetNamespace:  targetNamespace,
		targetDeployment: targetDeployment,
	}
	return certficateWatcher, nil
}

// Start starts the CertificateWatcher if it is not already running.
// It is safe to call Start multiple times, additional goroutines will not be created
func (sw *CertificateWatcher) Start() {
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
				if err := sw.WatchCertificates([]string{"verrazzano-platform-operator-ca", "verrazzano-platform-operator-tls"}); err != nil {
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

func (sw *CertificateWatcher) WatchCertificates(secrets []string) error {
	status := false
	var err error
	for i := range secrets {
		secret := secrets[i]
		sw.log.Debugf("secret/certificate found %v", secret)
		sec, secdata := sw.GetSecretData(secret)
		if secdata == nil {
			return fmt.Errorf("an error while listing the certificate secret data")
		}
		status, err = sw.ValidateCertDate(secdata)
		sw.log.Debugf("cert data expiry status for secret %v", secret)
		if err != nil {
			return fmt.Errorf("an error while validating the secret data")
		}
		if status {
			err = sw.DeleteSecret(sec)
			if err != nil {
				return fmt.Errorf("an error deleting the certificate")
			}
			err = sw.RolloutRestartDeployment()
			if err != nil {
				return fmt.Errorf("either deployment not found or an error while restarting the deployment")
			}
		}
	}
	return nil
}
