// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package generate

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	istioTLSSecret = "istio-certs"
	rootCertFile   = "root-cert.pem"
	certChainFile  = "cert-chain.pem"
	certKeyFile    = "key.pem"
)

// GenerateSecret continuously attempts to generate a secret once a minute
// this function should never exit and will continuously fail if the secret cannot be created
func GenerateSecret(client client.Client, log *zap.SugaredLogger) {
	// Look to create or update the secret every minute
	for _ = range time.Tick(time.Minute) {
		certDir := "/etc/istio-certs"
		rootCert, err := os.ReadFile(fmt.Sprintf("%s/root-cert.pem", certDir))
		if err != nil {
			log.Errorf("Failed to read the root certificate file: %v", err)
			continue
		}
		certChain, err := os.ReadFile(fmt.Sprintf("%s/cert-chain.pem", certDir))
		if err != nil {
			log.Errorf("Failed to read the certificate chain file: %v", err)
			continue
		}
		key, err := os.ReadFile(fmt.Sprintf("%s/key.pem", certDir))
		if err != nil {
			log.Errorf("Failed to read the certificate key file: %v", err)
			continue
		}

		tlsSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: constants.PrometheusOperatorNamespace,
				Name:      istioTLSSecret,
			},
		}
		_, err = controllerutil.CreateOrUpdate(context.TODO(), client, &tlsSecret, func() error {
			tlsSecret.Data = map[string][]byte{
				rootCertFile:  rootCert,
				certChainFile: certChain,
				certKeyFile:   key,
			}
			return nil
		})
		if err != nil {
			log.Errorf("Failed to update the secret: %v", err)
		}
	}
}
