// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
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

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	// OperatorName is the resource name for the Verrazzano platform operator
	OperatorName    = "verrazzano-platform-operator-webhook"
	OldOperatorName = "verrazzano-platform-operator"
	OperatorCA      = "verrazzano-platform-operator-ca"
	OperatorTLS     = "verrazzano-platform-operator-tls"

	// OperatorNamespace is the resource namespace for the Verrazzano platform operator
	OperatorNamespace = "verrazzano-install"
	CRDName           = "verrazzanos.install.verrazzano.io"

	CertKey = "tls.crt"
	PrivKey = "tls.key"

	certYearsValid = 3
)

// CreateWebhookCertificates creates the needed certificates for the validating webhook
func CreateWebhookCertificates(log *zap.SugaredLogger, kubeClient kubernetes.Interface, certDir string) error {

	commonName := fmt.Sprintf("%s.%s.svc", OperatorName, OperatorNamespace)
	ca, caKey, err := createCACert(log, kubeClient, commonName)
	if err != nil {
		return err
	}

	serverPEM, serverKeyPEM, err := createTLSCert(log, kubeClient, commonName, ca, caKey)
	if err != nil {
		return err
	}

	log.Debugf("Creating certs dir %s", certDir)
	if err := os.MkdirAll(certDir, 0666); err != nil {
		log.Errorf("Mkdir error %v", err)
		return err
	}

	if err := writeFile(log, fmt.Sprintf("%s/%s", certDir, CertKey), serverPEM); err != nil {
		log.Errorf("Error writing cert file: %v", err)
		return err
	}

	if err := writeFile(log, fmt.Sprintf("%s/%s", certDir, PrivKey), serverKeyPEM); err != nil {
		log.Errorf("Error 3 %v", err)
		return err
	}

	return nil
}

func createTLSCert(log *zap.SugaredLogger, kubeClient kubernetes.Interface, commonName string, ca *x509.Certificate, caKey *rsa.PrivateKey) ([]byte, []byte, error) {
	secretsClient := kubeClient.CoreV1().Secrets(OperatorNamespace)
	existingSecret, err := secretsClient.Get(context.TODO(), OperatorTLS, metav1.GetOptions{})
	if err == nil {
		log.Infof("Secret %s exists, using...", OperatorTLS)
		return existingSecret.Data[CertKey], existingSecret.Data[PrivKey], nil
	}
	if !errors.IsNotFound(err) {
		return []byte{}, []byte{}, err
	}

	serialNumber, err := newSerialNumber()
	if err != nil {
		return []byte{}, []byte{}, err
	}

	// server cert config
	cert := &x509.Certificate{
		DNSNames:     []string{commonName},
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(certYearsValid, 0, 0),
		IsCA:         false,
		SubjectKeyId: []byte{1, 2, 3, 4, 6},
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	// server private key
	serverPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return []byte{}, []byte{}, err
	}

	// sign the server cert
	serverCertBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, ca, &serverPrivKey.PublicKey, caKey)
	if err != nil {
		return []byte{}, []byte{}, err
	}

	return createTLSCertSecretIfNecesary(log, secretsClient, serverCertBytes, serverPrivKey)
}

func createTLSCertSecretIfNecesary(log *zap.SugaredLogger, secretsClient corev1.SecretInterface,
	serverCertBytes []byte, serverPrivKey *rsa.PrivateKey) ([]byte, []byte, error) {
	// PEM encode Server cert
	serverPEM := new(bytes.Buffer)
	_ = pem.Encode(serverPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: serverCertBytes,
	})

	// PEM encode Server cert
	serverKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(serverKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(serverPrivKey),
	})

	serverPEMBytes := serverPEM.Bytes()

	serverKeyPEMBytes := serverKeyPEM.Bytes()

	var webhookCrt v1.Secret
	webhookCrt.Namespace = OperatorNamespace
	webhookCrt.Name = OperatorTLS
	webhookCrt.Type = v1.SecretTypeTLS
	webhookCrt.Data = make(map[string][]byte)
	webhookCrt.Data[CertKey] = serverPEMBytes
	webhookCrt.Data[PrivKey] = serverKeyPEMBytes

	_, createError := secretsClient.Create(context.TODO(), &webhookCrt, metav1.CreateOptions{})
	if createError != nil {
		if errors.IsAlreadyExists(createError) {
			log.Infof("Operator CA secret %s already exists, skipping", OperatorCA)
			existingSecret, err := secretsClient.Get(context.TODO(), OperatorTLS, metav1.GetOptions{})
			if err != nil {
				return []byte{}, []byte{}, err
			}
			log.Infof("Secret %s exists, using...", OperatorTLS)
			return existingSecret.Data[CertKey], existingSecret.Data[PrivKey], nil
		}
		return []byte{}, []byte{}, createError
	}

	return serverPEMBytes, serverKeyPEMBytes, nil
}

func createCACert(log *zap.SugaredLogger, kubeClient kubernetes.Interface, commonName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	secretsClient := kubeClient.CoreV1().Secrets(OperatorNamespace)
	existingSecret, err := secretsClient.Get(context.TODO(), OperatorCA, metav1.GetOptions{})
	if err == nil {
		log.Infof("CA secret %s exists, using...", OperatorCA)
		return decodeExistingSecretData(existingSecret)
	}
	if !errors.IsNotFound(err) {
		return nil, nil, err
	}

	log.Infof("Creating CA secret %s", OperatorCA)
	serialNumber, err := newSerialNumber()
	if err != nil {
		return nil, nil, err
	}

	// CA config
	ca := &x509.Certificate{
		DNSNames:     []string{commonName},
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(certYearsValid, 0, 0),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// CA private key
	caPrivKey, err := rsa.GenerateKey(cryptorand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	// Self signed CA certificate
	caBytes, err := x509.CreateCertificate(cryptorand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	return createCACertSecretIfNecessary(log, secretsClient, ca, caPrivKey, caBytes)
}

func createCACertSecretIfNecessary(log *zap.SugaredLogger, secretsClient corev1.SecretInterface, ca *x509.Certificate,
	caPrivKey *rsa.PrivateKey, caBytes []byte) (*x509.Certificate, *rsa.PrivateKey, error) {
	caPEMBytes, caKeyPEMBytes := encodeCABytes(caBytes, caPrivKey)

	webhookCA := v1.Secret{}
	webhookCA.Namespace = OperatorNamespace
	webhookCA.Name = OperatorCA
	webhookCA.Type = v1.SecretTypeTLS

	webhookCA.Data = make(map[string][]byte)
	webhookCA.Data[CertKey] = caPEMBytes
	webhookCA.Data[PrivKey] = caKeyPEMBytes

	_, createError := secretsClient.Create(context.TODO(), &webhookCA, metav1.CreateOptions{})
	if createError != nil {
		if errors.IsAlreadyExists(createError) {
			log.Infof("Operator CA secret %s already exists, using existing secret", OperatorCA)
			existingSecret, err := secretsClient.Get(context.TODO(), OperatorCA, metav1.GetOptions{})
			if err != nil {
				return nil, nil, err
			}
			return decodeExistingSecretData(existingSecret)
		}
		return nil, nil, createError
	}
	return ca, caPrivKey, nil
}

// encodeCABytes PEM-encode the certificate and Key data
func encodeCABytes(caBytes []byte, caPrivKey *rsa.PrivateKey) ([]byte, []byte) {
	// PEM encode CA cert
	caPEM := new(bytes.Buffer)
	_ = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	})

	// PEM encode CA cert
	caKeyPEM := new(bytes.Buffer)
	_ = pem.Encode(caKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	})

	return caPEM.Bytes(), caKeyPEM.Bytes()
}

// decodeExistingSecretData Decode existing secret data into their X509 and RSA objects
func decodeExistingSecretData(secret *v1.Secret) (*x509.Certificate, *rsa.PrivateKey, error) {
	cert, err := decodeCertificate(secret.Data[CertKey])
	if err != nil {
		return nil, nil, err
	}
	key, err := decodeKey(secret.Data[PrivKey])
	if err != nil {
		return nil, nil, err
	}
	return cert, key, err
}

// decodeCertificate Decode certificate PEM data
func decodeCertificate(certBytes []byte) (*x509.Certificate, error) {
	p, _ := pem.Decode(certBytes)
	if p == nil {
		return nil, fmt.Errorf("Unable to decode certificate")
	}
	certificate, err := x509.ParseCertificate(p.Bytes)
	if err != nil {
		return nil, err
	}
	return certificate, nil
}

// decodeKey - Decode private Key PEM data
func decodeKey(keyBytes []byte) (*rsa.PrivateKey, error) {
	p, _ := pem.Decode(keyBytes)
	if p == nil {
		return nil, fmt.Errorf("Unable to decode certificate")
	}
	key, err := x509.ParsePKCS1PrivateKey(p.Bytes)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// newSerialNumber returns a new random serial number suitable for use in a certificate.
func newSerialNumber() (*big.Int, error) {
	// A serial number can be up to 20 octets in size.
	return cryptorand.Int(cryptorand.Reader, new(big.Int).Lsh(big.NewInt(1), 8*20))
}

// writeFile writes data in the file at the given path
func writeFile(log *zap.SugaredLogger, filepath string, pemData []byte) error {
	log.Debugf("Writing file %s", filepath)
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.Write(pemData)
	if err != nil {
		return err
	}

	return nil
}
