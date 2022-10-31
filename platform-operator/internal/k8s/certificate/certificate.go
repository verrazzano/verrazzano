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
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"math/big"
	"os"
	"time"

	adminv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

	certKey = "tls.crt"
	privKey = "tls.key"

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

	log.Debugf("Creating certs dir %s", certDir)
	err = os.MkdirAll(certDir, 0666)
	if err != nil {
		log.Errorf("Mkdir error %v", err)
		return err
	}

	certFile := fmt.Sprintf("%s/tls.crt", certDir)
	log.Debugf("Writing file %s", certFile)
	err = writeFile(certFile, serverPEM)
	if err != nil {
		log.Errorf("Error 2 %v", err)
		return err
	}

	keyFile := fmt.Sprintf("%s/tls.key", certDir)
	log.Debugf("Writing file %s", keyFile)
	err = writeFile(keyFile, serverKeyPEM)
	if err != nil {
		log.Errorf("Error 3 %v", err)
		return err
	}

	return nil
}

func createTLSCert(log *zap.SugaredLogger, kubeClient kubernetes.Interface, commonName string, ca *x509.Certificate, caKey *rsa.PrivateKey) ([]byte, []byte, error) {
	const certName = OperatorTLS
	secretsClient := kubeClient.CoreV1().Secrets(OperatorNamespace)
	existingSecret, err := secretsClient.Get(context.TODO(), certName, metav1.GetOptions{})
	if err == nil {
		log.Infof("Secret %s exists, using...", certName)
		return existingSecret.Data[certKey], existingSecret.Data[privKey], nil
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
	webhookCrt.Name = certName
	webhookCrt.Type = v1.SecretTypeTLS
	webhookCrt.Data = make(map[string][]byte)
	webhookCrt.Data["tls.crt"] = serverPEMBytes
	webhookCrt.Data[privKey] = serverKeyPEMBytes

	_, createError := secretsClient.Create(context.TODO(), &webhookCrt, metav1.CreateOptions{})
	if createError != nil {
		if errors.IsAlreadyExists(createError) {
			log.Infof("Operator CA secret %s already exists, skipping", OperatorCA)
			existingSecret, err := secretsClient.Get(context.TODO(), certName, metav1.GetOptions{})
			if err != nil {
				return []byte{}, []byte{}, err
			}
			log.Infof("Secret %s exists, using...", certName)
			return existingSecret.Data[certKey], existingSecret.Data[privKey], nil
		}
	}

	return serverPEMBytes, serverKeyPEMBytes, nil
}

func createCACert(log *zap.SugaredLogger, kubeClient kubernetes.Interface, commonName string) (*x509.Certificate, *rsa.PrivateKey, error) {
	const certName = OperatorCA

	var webhookCA v1.Secret
	caKeyPEMBytes := []byte{}
	var ca *x509.Certificate

	webhookCA.Namespace = OperatorNamespace
	webhookCA.Name = certName
	webhookCA.Type = v1.SecretTypeTLS

	secretsClient := kubeClient.CoreV1().Secrets(OperatorNamespace)
	existingSecret, err := secretsClient.Get(context.TODO(), certName, metav1.GetOptions{})
	if err == nil {
		log.Infof("CA secret %s exists, using...", certName)
		cert, err := decodeCertificate(existingSecret.Data[certKey])
		if err != nil {
			return nil, nil, err
		}
		key, err := decodeKey(existingSecret.Data[privKey])
		if err != nil {
			return nil, nil, err
		}
		return cert, key, err
	}
	if !errors.IsNotFound(err) {
		return nil, nil, err
	}

	log.Infof("Creating CA secret %s", certName)
	serialNumber, err := newSerialNumber()
	if err != nil {
		return nil, nil, err
	}

	// CA config
	ca = &x509.Certificate{
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

	caPEMBytes := caPEM.Bytes()

	caKeyPEMBytes = caKeyPEM.Bytes()

	webhookCA.Data = make(map[string][]byte)
	webhookCA.Data[certKey] = caPEMBytes
	webhookCA.Data[privKey] = caKeyPEMBytes

	_, createError := secretsClient.Create(context.TODO(), &webhookCA, metav1.CreateOptions{})
	if createError != nil {
		if errors.IsAlreadyExists(createError) {
			log.Infof("Operator CA secret %s already exists, skipping", certName)
			existingSecret, err := secretsClient.Get(context.TODO(), certName, metav1.GetOptions{})
			if err != nil {
				return nil, nil, err
			}
			caPEMBytes = existingSecret.Data[certKey]
			cert, err := decodeCertificate(caPEMBytes)
			caKeyPEMBytes = existingSecret.Data[privKey]
			if err != nil {
				return nil, nil, err
			}
			key, err := decodeKey(existingSecret.Data[privKey])
			if err != nil {
				return nil, nil, err
			}
			return cert, key, nil
		}
		return nil, nil, err
	}
	return ca, caPrivKey, nil
}

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

func decodeKey(certBytes []byte) (*rsa.PrivateKey, error) {
	p, _ := pem.Decode(certBytes)
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
func writeFile(filepath string, pemData []byte) error {
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

// DeleteValidatingWebhookConfiguration deletes a validating webhook configuration
func DeleteValidatingWebhookConfiguration(kubeClient kubernetes.Interface, name string) error {
	_, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	err = kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.TODO(), name, metav1.DeleteOptions{})

	return err
}

// UpdateValidatingWebhookConfiguration sets the CABundle
func UpdateValidatingWebhookConfiguration(kubeClient kubernetes.Interface, name string) error {
	validatingWebhook, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	caSecret, errX := kubeClient.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorCA, metav1.GetOptions{})
	if errX != nil {
		return errX
	}
	crt := caSecret.Data["tls.crt"]

	for i := range validatingWebhook.Webhooks {
		validatingWebhook.Webhooks[i].ClientConfig.CABundle = crt
	}

	_, err = kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(context.TODO(), validatingWebhook, metav1.UpdateOptions{})
	return err
}

// UpdateConversionWebhookConfiguration sets the conversion webhook for the Verrazzano resource
func UpdateConversionWebhookConfiguration(apiextClient *apiextensionsv1client.ApiextensionsV1Client, kubeClient kubernetes.Interface) error {
	crd, err := apiextClient.CustomResourceDefinitions().Get(context.TODO(), CRDName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	convertPath := "/convert"
	var webhookPort int32 = 443
	caSecret, err := kubeClient.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorCA, metav1.GetOptions{})
	if err != nil {
		return err
	}
	crt := caSecret.Data["tls.crt"]

	crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{
		Strategy: apiextensionsv1.WebhookConverter,
		Webhook: &apiextensionsv1.WebhookConversion{
			ClientConfig: &apiextensionsv1.WebhookClientConfig{
				Service: &apiextensionsv1.ServiceReference{
					Name:      OperatorName,
					Namespace: OperatorNamespace,
					Path:      &convertPath,
					Port:      &webhookPort,
				},
				CABundle: crt,
			},
			ConversionReviewVersions: []string{"v1beta1"},
		},
	}
	_, err = apiextClient.CustomResourceDefinitions().Update(context.TODO(), crd, metav1.UpdateOptions{})
	return err
}

// UpdateMutatingWebhookConfiguration sets the CABundle
func UpdateMutatingWebhookConfiguration(kubeClient kubernetes.Interface, name string) error {
	var webhook *adminv1.MutatingWebhookConfiguration
	webhook, err := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	caSecret, err := kubeClient.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorCA, metav1.GetOptions{})
	if err != nil {
		return err
	}
	crt := caSecret.Data["tls.crt"]
	if err != nil {
		return err
	}
	for i := range webhook.Webhooks {
		webhook.Webhooks[i].ClientConfig.CABundle = crt
	}
	_, err = kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.TODO(), webhook, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
