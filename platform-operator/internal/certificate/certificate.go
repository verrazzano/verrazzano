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

// CreateWebhookCertificates creates the needed certificates for the validating webhook
func CreateWebhookCertificates(certDir string) (*bytes.Buffer, error) {
	var caPEM, serverCertPEM, serverPrivKeyPEM *bytes.Buffer

	commonName := fmt.Sprintf("%s.%s.svc", OperatorName, OperatorNamespace)
	serialNumber, err := newSerialNumber()
	if err != nil {
		return nil, err
	}

	// CA config
	ca := &x509.Certificate{
		DNSNames:     []string{commonName},
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// CA private key
	caPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// Self signed CA certificate
	caBytes, err := x509.CreateCertificate(cryptorand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode CA cert
	caPEM = new(bytes.Buffer)
	_ = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	serialNumber, err = newSerialNumber()
	if err != nil {
		return nil, err
	}

	// server cert config
	cert := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		IsCA:         true,
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// server private key
	serverPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	// sign the server cert
	serverCertBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, ca, &serverPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, err
	}

	// PEM encode the server cert and key
	serverCertPEM = new(bytes.Buffer)
	_ = pem.Encode(serverCertPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertBytes,
	})

	serverPrivKeyPEM = new(bytes.Buffer)
	_ = pem.Encode(serverPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	})

	err = os.MkdirAll(certDir, 0666)
	if err != nil {
		return nil, err
	}

	err = writeFile(fmt.Sprintf("%s/tls.crt", certDir), serverCertPEM)
	if err != nil {
		return nil, err
	}

	err = writeFile(fmt.Sprintf("%s/tls.key", certDir), serverPrivKeyPEM)
	if err != nil {
		return nil, err
	}

	return caPEM, nil
}

// newSerialNumber returns a new random serial number suitable for use in a certificate.
func newSerialNumber() (*big.Int, error) {
	// A serial number can be up to 20 octets in size.
	return cryptorand.Int(cryptorand.Reader, new(big.Int).Lsh(big.NewInt(1), 8*20))
}

// writeFile writes data in the file at the given path
func writeFile(filepath string, pem *bytes.Buffer) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(pem.Bytes())
	if err != nil {
		return err
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
