// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificates

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	adminv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestSetupCertificates tests that the certificates needed for webhooks are created
// GIVEN an output directory for certificates
//
//	WHEN I call SetupCertificates
//	THEN all the needed certificate artifacts are created
func TestSetupCertificates(t *testing.T) {
	a := assert.New(t)

	dir, err := os.MkdirTemp("", "certs")
	if err != nil {
		a.Nil(err, "error should not be returned creating temporary directory")
	}
	defer os.RemoveAll(dir)
	caBundle, err := SetupCertificates(dir)
	a.Nil(err, "error should not be returned setting up certificates")
	a.NotNil(caBundle, "CA bundle should be returned")
	crtFile := fmt.Sprintf("%s/%s", dir, "tls.crt")
	keyFile := fmt.Sprintf("%s/%s", dir, "tls.key")
	a.FileExists(crtFile, dir, "tls.crt", "expected tls.crt file not found")
	a.FileExists(keyFile, dir, "tls.key", "expected tls.key file not found")

	crtBytes, err := os.ReadFile(crtFile)
	if a.NoError(err) {
		block, _ := pem.Decode(crtBytes)
		a.NotEmptyf(block, "failed to decode PEM block containing public key")
		a.Equal("CERTIFICATE", block.Type)
		cert, err := x509.ParseCertificate(block.Bytes)
		if a.NoError(err) {
			a.NotEmpty(cert.DNSNames, "Certificate DNSNames SAN field should not be empty")
			a.Equal("verrazzano-application-operator.verrazzano-system.svc", cert.DNSNames[0])
		}
	}
}

// TestSetupCertificatesFail tests that the certificates needed for webhooks are not created
// GIVEN an invalid output directory for certificates
//
//	WHEN I call SetupCertificates
//	THEN all the needed certificate artifacts are not created
func TestSetupCertificatesFail(t *testing.T) {
	a := assert.New(t)

	_, err := SetupCertificates("bad-dir")
	a.Error(err, "error should be returned setting up certificates")
}

// TestUpdateValidatingWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-platform-operator
// validatingWebhookConfiguration resource.
// GIVEN a validatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call UpdateValidatingnWebhookConfiguration
//	THEN the validatingWebhookConfiguration resource set the CA Bundle as expected
func TestUpdateValidatingWebhookConfiguration(t *testing.T) {
	a := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/validate-oam-verrazzano-io-v1alpha1-ingresstrait"
	service := adminv1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: IngressTraitValidatingWebhookName,
		},
		Webhooks: []adminv1.ValidatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	a.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateValidatingWebhookConfiguration(kubeClient, &caCert, IngressTraitValidatingWebhookName)
	a.Nil(err, "error should not be returned updating validation webhook configuration")

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), IngressTraitValidatingWebhookName, metav1.GetOptions{})
	a.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateValidatingWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-platform-operator validatingWebhookConfiguration resource.
// GIVEN an invalid validatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call UpdateValidatingnWebhookConfiguration
//	THEN the validatingWebhookConfiguration resource will fail to be updated
func TestUpdateValidatingWebhookConfigurationFail(t *testing.T) {
	a := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/validate-oam-verrazzano-io-v1alpha1-ingresstrait"
	service := adminv1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "InvalidName",
		},
		Webhooks: []adminv1.ValidatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	a.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateValidatingWebhookConfiguration(kubeClient, &caCert, IngressTraitValidatingWebhookName)
	a.Error(err, "error should be returned updating validation webhook configuration")
}

// TestUpdateAppConfigMutatingWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-application-operator
// mutatingWebhookConfiguration resource.
// GIVEN a mutatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call UpdateAppConfigMutatingWebhookConfiguration
//	THEN the mutatingWebhookConfiguration resource set the CA Bundle as expected
func TestUpdateAppConfigMutatingWebhookConfiguration(t *testing.T) {
	a := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/appconfig-defaulter"
	service := adminv1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: AppConfigMutatingWebhookName,
		},
		Webhooks: []adminv1.MutatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	a.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateMutatingWebhookConfiguration(kubeClient, &caCert, AppConfigMutatingWebhookName)
	a.Nil(err, "error should not be returned updating validation webhook configuration")

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), AppConfigMutatingWebhookName, metav1.GetOptions{})
	a.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateAppConfigMutatingWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-application-operator mutatingWebhookConfiguration resource.
// GIVEN an invalid mutatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call UpdateAppConfigMutatingWebhookConfiguration
//	THEN the mutatingWebhookConfiguration resource will fail to be updated
func TestUpdateAppConfigMutatingWebhookConfigurationFail(t *testing.T) {
	a := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/appconfig-defaulter"
	service := adminv1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "InvalidName",
		},
		Webhooks: []adminv1.MutatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	a.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateMutatingWebhookConfiguration(kubeClient, &caCert, AppConfigMutatingWebhookName)
	a.Error(err, "error should be returned updating validation webhook configuration")
}

// TestUpdateIstioMutatingWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-application-operator
// mutatingWebhookConfiguration resource.
// GIVEN a mutatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call UpdateIstioMutatingWebhookConfiguration
//	THEN the mutatingWebhookConfiguration resource set the CA Bundle as expected
func TestUpdateIstioMutatingWebhookConfiguration(t *testing.T) {
	a := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/istio-defaulter"
	service := adminv1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: IstioMutatingWebhookName,
		},
		Webhooks: []adminv1.MutatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	a.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateMutatingWebhookConfiguration(kubeClient, &caCert, IstioMutatingWebhookName)
	a.Nil(err, "error should not be returned updating validation webhook configuration")

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), IstioMutatingWebhookName, metav1.GetOptions{})
	a.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateIstioMutatingWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-application-operator mutatingWebhookConfiguration resource.
// GIVEN an invalid mutatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call UpdateIstioMutatingWebhookConfiguration
//	THEN the mutatingWebhookConfiguration resource will fail to be updated
func TestUpdateIstioMutatingWebhookConfigurationFail(t *testing.T) {
	a := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/istio-defaulter"
	service := adminv1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "InvalidName",
		},
		Webhooks: []adminv1.MutatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	a.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateMutatingWebhookConfiguration(kubeClient, &caCert, IstioMutatingWebhookName)
	a.Error(err, "error should be returned updating validation webhook configuration")
}
