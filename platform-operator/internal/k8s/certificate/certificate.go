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
	"fmt"
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
)

// CreateWebhookCertificates creates the needed certificates for the validating webhook
func CreateWebhookCertificates(certDir string, kubeClient *kubernetes.Clientset) error {

	commonName := fmt.Sprintf("%s.%s.svc", OperatorName, OperatorNamespace)
	serialNumber, err := newSerialNumber()
	if err != nil {
		return err
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
		return err
	}

	// Self signed CA certificate
	caBytes, err := x509.CreateCertificate(cryptorand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	serialNumber, err = newSerialNumber()
	if err != nil {
		return err
	}

	// server cert config
	cert := &x509.Certificate{
		DNSNames:     []string{commonName},
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
		return err
	}

	// sign the server cert
	serverCertBytes, err := x509.CreateCertificate(cryptorand.Reader, cert, ca, &serverPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return err
	}

	var webhookCA v1.Secret
	webhookCA.Namespace = OperatorNamespace
	webhookCA.Name = OperatorCA
	webhookCA.Type = v1.SecretTypeTLS
	webhookCA.Data = make(map[string][]byte)
	webhookCA.Data["tls.crt"] = caBytes
	webhookCA.Data["tls.key"] = x509.MarshalPKCS1PrivateKey(caPrivKey)

	_, err = kubeClient.CoreV1().Secrets(OperatorNamespace).Create(context.TODO(), &webhookCA, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	var webhookCrt v1.Secret
	webhookCrt.Namespace = OperatorNamespace
	webhookCrt.Name = OperatorTLS
	webhookCrt.Type = v1.SecretTypeTLS
	webhookCrt.Data = make(map[string][]byte)
	webhookCrt.Data["tls.crt"] = serverCertBytes
	webhookCrt.Data["tls.key"] = x509.MarshalPKCS1PrivateKey(serverPrivKey)

	_, err = kubeClient.CoreV1().Secrets(OperatorNamespace).Create(context.TODO(), &webhookCrt, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
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

// DeleteValidatingWebhookConfiguration deletes a validating webhook configuration
func DeleteValidatingWebhookConfiguration(kubeClient kubernetes.Interface, name string) error {
	_, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	err = kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.TODO(), name, metav1.DeleteOptions{})

	return err
}

// UpdateValidatingnWebhookConfiguration sets the CABundle
func UpdateValidatingnWebhookConfiguration(kubeClient kubernetes.Interface, name string) error {
	validatingWebhook, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	caSecret, err := kubeClient.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorCA, metav1.GetOptions{})
	if err != nil {
		return err
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
