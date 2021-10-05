// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"time"

	adminv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// OperatorName is the resource name for the Verrazzano platform operator
	OperatorName = "verrazzano-platform-operator"
	// OperatorNamespace is the resource namespace for the Verrazzano platform operator
	OperatorNamespace = "verrazzano-install"
)

// Config specifies the certificate configuration
type Config struct {
	// CertDir is the directory where the certificate should be written
	CertDir string

	// CommonName is the certificate common name
	CommonName string

	// NotBefore time when certificate is valid
	NotBefore time.Time

	// NotAfter time when certificate is valid
	NotAfter time.Time

	// FilenameRootCert is the filename in CertDir where the root cert PEM will be written
	FilenameRootCert string

	// FilenameRootPrivKey is the filename in CertDir where the root private key PEM will be written
	FilenameRootPrivKey string

	// FilenameIntermediateCert is the filename in CertDir where the intermediate cert PEM will be written
	FilenameIntermediateCert string

	// FilenameIntermediatePrivKey is the filename in CertDir where the intermediate private key PEM will be written
	FilenameIntermediatePrivKey string

	// FilenameCertChain is the filename in CertDir where that cert chain PEM (ca cert + root cert) will be written
	FilenameCertChain string
}

// CreateSelfSignedCert creates the needed certificates for the validating webhook
func CreateSelfSignedCert(config Config) (*bytes.Buffer, error) {
	serialNumber, err := newSerialNumber()
	if err != nil {
		return nil, err
	}

	// create the root certificate info needed to create the certificate
	rootCertInfo := &x509.Certificate{
		DNSNames:     []string{config.CommonName},
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: config.CommonName,
		},
		NotBefore:             config.NotBefore,
		NotAfter:              config.NotAfter,
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create root private key
	rootPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// PEM encode root private key
	rootPrivKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(rootPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(rootPrivKey),
	})

	// create root self-signed CA certificate
	rootCertBytes, err := x509.CreateCertificate(cryptorand.Reader, rootCertInfo, rootCertInfo, &rootPrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode root cert
	rootCertPEM := new(bytes.Buffer)
	_ = pem.Encode(rootCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: rootCertBytes,
	})

	serialNumber, err = newSerialNumber()
	if err != nil {
		return nil, err
	}

	// create the intermediate certificate info needed to create the certificate
	intermediateCertInfo := &x509.Certificate{
		DNSNames:     []string{config.CommonName},
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: config.CommonName,
		},
		NotBefore:    config.NotBefore,
		NotAfter:     config.NotAfter,
		IsCA:         true,
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// generate intermediate cert private key
	intermediatePrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// PEM encode the intermediate private key
	intermediatePrivKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(intermediatePrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(intermediatePrivKey),
	})

	// sign the intermediate cert with the root cert
	intermediateCertBytes, err := x509.CreateCertificate(cryptorand.Reader, intermediateCertInfo, rootCertInfo, &intermediatePrivKey.PublicKey, rootPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode the intermediate cert and key
	intermediateCertPEM := new(bytes.Buffer)
	_ = pem.Encode(intermediateCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: intermediateCertBytes,
	})

	// Write the files that are specified in the config
	certDir := config.CertDir
	err = os.MkdirAll(certDir, 0666)
	if err != nil {
		return nil, err
	}

	if len(config.FilenameRootPrivKey) > 0 {
		err = writeFile(fmt.Sprintf("%s/%s", certDir, config.FilenameRootPrivKey), rootPrivKeyPEM)
		if err != nil {
			return nil, err
		}
	}
	if len(config.FilenameRootCert) > 0 {
		err = writeFile(fmt.Sprintf("%s/%s", certDir, config.FilenameRootCert), rootCertPEM)
		if err != nil {
			return nil, err
		}
	}
	if len(config.FilenameIntermediateCert) > 0 {
		err = writeFile(fmt.Sprintf("%s/%s", certDir, config.FilenameIntermediateCert), intermediateCertPEM)
		if err != nil {
			return nil, err
		}
	}
	if len(config.FilenameIntermediatePrivKey) > 0 {
		err = writeFile(fmt.Sprintf("%s/%s", certDir, config.FilenameIntermediatePrivKey), intermediatePrivKeyPEM)
		if err != nil {
			return nil, err
		}
	}
	if len(config.FilenameCertChain) > 0 {
		err = writeFile(fmt.Sprintf("%s/%s", certDir, config.FilenameCertChain), rootCertPEM, intermediateCertPEM)
		if err != nil {
			return nil, err
		}
	}
	return rootCertPEM, nil
}

// newSerialNumber returns a new random serial number suitable for use in a certificate.
func newSerialNumber() (*big.Int, error) {
	// A serial number can be up to 20 octets in size.
	return cryptorand.Int(cryptorand.Reader, new(big.Int).Lsh(big.NewInt(1), 8*20))
}

// writeFile writes data in the file at the given path
func writeFile(filepath string, pems ...*bytes.Buffer) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, pem := range pems {
		_, err = f.Write(pem.Bytes())
		if err != nil {
			return err
		}
	}
	return nil
}

// UpdateValidatingnWebhookConfiguration sets the CABundle
func UpdateValidatingnWebhookConfiguration(kubeClient kubernetes.Interface, caCert *bytes.Buffer) error {
	var validatingWebhook *adminv1beta1.ValidatingWebhookConfiguration
	validatingWebhook, err := kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(context.TODO(), OperatorName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if len(validatingWebhook.Webhooks) != 2 {
		return fmt.Errorf("Expected 2 webhooks in %s ValidatingWebhookConfiguration, but found %v", OperatorName, len(validatingWebhook.Webhooks))
	}
	validatingWebhook.Webhooks[0].ClientConfig.CABundle = caCert.Bytes()
	validatingWebhook.Webhooks[1].ClientConfig.CABundle = caCert.Bytes()
	_, err = kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Update(context.TODO(), validatingWebhook, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// CreateWebhookCertConfig creates the config needed to create platform operator webhook certificate
func CreateWebhookCertConfig(certDir string) Config {
	return Config{
		CertDir:                     certDir,
		CommonName:                  fmt.Sprintf("%s.%s.svc", OperatorName, OperatorNamespace),
		FilenameIntermediateCert:    "tls.crt",
		FilenameIntermediatePrivKey: "tls.key",
		NotBefore:                   time.Now(),
		NotAfter:                    time.Now().AddDate(1, 0, 0),
	}
}

// CreateIstioCertConfig creates the config needed to create an Istio certificate
func CreateIstioCertConfig(certDir string) Config {
	return Config{
		CertDir:                     certDir,
		CommonName:                  fmt.Sprintf("%s.%s.svc", OperatorName, OperatorNamespace),
		FilenameIntermediateCert:    "tls.crt",
		FilenameIntermediatePrivKey: "tls.key",
		NotBefore:                   time.Now(),
		NotAfter:                    time.Now().AddDate(1, 0, 0),
	}
}
