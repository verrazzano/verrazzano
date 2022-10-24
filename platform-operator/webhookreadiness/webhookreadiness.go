// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhookreadiness

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"github.com/verrazzano/verrazzano/pkg/constants"
	vpoconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Platform Operator webhook endpoints
	v1beta1Endpoint  = "https://verrazzano-platform-operator-webhook.verrazzano-install.svc/validate-install-verrazzano-io-v1beta1-verrazzano"
	v1alpha1Endpoint = "https://verrazzano-platform-operator-webhook.verrazzano-install.svc/validate-install-verrazzano-io-v1alpha1-verrazzano"
	vmcEndpoint      = "https://verrazzano-platform-operator-webhook.verrazzano-install.svc/validate-clusters-verrazzano-io-v1alpha1-verrazzanomanagedcluster"
)

var webhookEndpoints = []string{
	v1beta1Endpoint, v1alpha1Endpoint, vmcEndpoint,
}

type startupProbeServer struct {
	log    *zap.SugaredLogger
	client client.Client
}

// StartStartupProbeServer to check webhook readiness
func StartStartupProbeServer(log *zap.SugaredLogger, client client.Client) error {
	log.Info("Starting platform operator readiness server")
	server, err := newStartupProbeServer(log, client)
	if err != nil {
		return err
	}
	server.run()
	return nil
}

func newStartupProbeServer(log *zap.SugaredLogger, c client.Client) (*startupProbeServer, error) {
	return &startupProbeServer{
		log:    log,
		client: c,
	}, nil
}

// run Starts the startup probe endpoint
func (r *startupProbeServer) run() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		statusCode := r.performReadinessCheck()
		r.log.Debugf("Webhook status check result: %v", statusCode)
		response.WriteHeader(statusCode)
	})
	srv := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		Addr:              fmt.Sprintf(":%d", 8081),
		Handler:           mux,
	}
	go func() {
		err := srv.ListenAndServe()
		r.log.Infof("Readiness listener shutting down: %s", err.Error())
	}()
}

func (r *startupProbeServer) performReadinessCheck() int {
	for _, endpoint := range webhookEndpoints {
		if status := r.checkWebhookEndpoint(endpoint); status != http.StatusOK {
			return status
		}
	}
	return http.StatusOK
}

// checkWebhookEndpoint - Checks an individual webhook endpoint for availability
func (r *startupProbeServer) checkWebhookEndpoint(endpoint string) int {
	r.log.Debugf("Checking webhook endpoint %s", endpoint)
	hc, err := r.buildHTTPClient()
	if err != nil {
		return http.StatusInternalServerError
	}
	request, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return http.StatusInternalServerError
	}
	response, err := hc.Do(request)
	if err != nil {
		r.log.Debugf("Error checking status of webhook %s: %s", endpoint, err)
		return http.StatusServiceUnavailable
	}
	r.log.Debugf("Webhook endpoint %s status %v", endpoint, response.StatusCode)
	return response.StatusCode
}

// buildHTTPClient - Builds an HTTP client with the Webhook CA bundle secret
func (r *startupProbeServer) buildHTTPClient() (http.Client, error) {
	caBundlePEMData, err := r.getOperatorCABundle()
	if err != nil {
		return http.Client{}, err
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(caBundlePEMData)
	hc := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    pool,
				MinVersion: tls.VersionTLS12,
			},
		},
	}
	return hc, nil
}

// getOperatorCABundle - Returns the Webhook CA bundle from the bundle secret
func (r *startupProbeServer) getOperatorCABundle() ([]byte, error) {
	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: vpoconst.OperatorCA, Namespace: constants.VerrazzanoInstallNamespace},
	}
	err := r.client.Get(context.TODO(), client.ObjectKeyFromObject(&secret), &secret)
	if err != nil {
		r.log.Errorf("Error obtaining Platform Operator CA bundle from secret: %s", err.Error())
		return nil, err
	}
	caBundlePEMData := secret.Data[vpoconst.OperatorCADataKey]
	return caBundlePEMData, nil
}
