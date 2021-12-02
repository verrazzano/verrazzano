// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"context"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/certificate"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

// createCertSecret creates the secret with cert data used by Istio MTLS
func createCertSecret(log *zap.SugaredLogger, client clipkg.Client) error {
	pemData, err := createCerts(log)
	if err != nil {
		return err
	}
	return createSecret(log, client, pemData)
}

// createCert creates the cert used by Istio MTLS
func createCerts(log *zap.SugaredLogger) (*certificate.CertPemData, error) {
	const (
		country = "US"
		org     = "Oracle Corporation"
		state   = "CA"
	)
	rootConfig := certificate.CertConfig{
		CountryName:         country,
		OrgName:             org,
		StateOrProvinceName: state,
		CommonName:          "Root CA",
		NotBefore:           time.Now(),
		NotAfter:            time.Now().AddDate(10, 0, 0),
	}
	intermConfig := certificate.CertConfig{
		CountryName:         country,
		OrgName:             org,
		StateOrProvinceName: state,
		CommonName:          "Intermediate CA",
		NotBefore:           time.Now(),
		NotAfter:            time.Now().AddDate(5, 0, 0),
	}
	pemData, err := certificate.CreateSelfSignedCert(rootConfig, intermConfig)
	if err != nil {
		log.Errorf("Failed to create Certificate for Istio: %v", err)
		return nil, err
	}
	return pemData, nil
}

func createSecret(log *zap.SugaredLogger, client clipkg.Client, pemData *certificate.CertPemData) error {
	const (
		caPem        = "ca-cert.pem"
		caKey        = "ca-key.pem"
		certChainPem = "cert-chain.pem"
		rootPem      = "root-cert.pem"
		secretName   = "cacerts"
	)
	var secret corev1.Secret
	secret.Namespace = IstioNamespace
	secret.Name = secretName

	_, err := controllerutil.CreateOrUpdate(context.TODO(), client, &secret, func() error {
		secret.Type = corev1.SecretTypeOpaque
		secret.Data = map[string][]byte{
			caPem:        pemData.IntermediateCertResult.CertPEM,
			caKey:        pemData.IntermediateCertResult.PrivateKeyPEM,
			certChainPem: pemData.CertChainPEM,
			rootPem:      pemData.RootCertResult.CertPEM,
		}
		return nil
	})
	if err != nil {
		log.Errorf("Failed to create Istio certificate secret cacerts: %v", err)
		return err
	}
	return nil
}
