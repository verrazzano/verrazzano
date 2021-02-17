// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificates

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	adminv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestSetupCertificates tests that the certificates needed for webhooks are created
// GIVEN an output directory for certificates
//  WHEN I call SetupCertificates
//  THEN all the needed certificate artifacts are created
func TestSetupCertificates(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "certs")
	if err != nil {
		assert.Nil(err, "error should not be returned creating temporary directory")
	}
	defer os.RemoveAll(dir)
	caBundle, err := SetupCertificates(dir)
	assert.Nil(err, "error should not be returned setting up certificates")
	assert.NotNil(caBundle, "CA bundle should be returned")
	crtFile := fmt.Sprintf("%s/%s", dir, "tls.crt")
	keyFile := fmt.Sprintf("%s/%s", dir, "tls.key")
	assert.FileExists(crtFile, dir, "tls.crt", "expected tls.crt file not found")
	assert.FileExists(keyFile, dir, "tls.key", "expected tls.key file not found")

	crtBytes, err := ioutil.ReadFile(crtFile)
	if assert.NoError(err) {
		block, _ := pem.Decode(crtBytes)
		assert.NotEmptyf(block, "failed to decode PEM block containing public key")
		assert.Equal("CERTIFICATE", block.Type)
		cert, err := x509.ParseCertificate(block.Bytes)
		if assert.NoError(err) {
			assert.NotEmpty(cert.DNSNames, "Certificate DNSNames SAN field should not be empty")
			assert.Equal("verrazzano-application-operator.verrazzano-system.svc", cert.DNSNames[0])
		}
	}
}

// TestSetupCertificatesFail tests that the certificates needed for webhooks are not created
// GIVEN an invalid output directory for certificates
//  WHEN I call SetupCertificates
//  THEN all the needed certificate artifacts are not created
func TestSetupCertificatesFail(t *testing.T) {
	assert := assert.New(t)

	_, err := SetupCertificates("bad-dir")
	assert.Error(err, "error should be returned setting up certificates")
}

// TestUpdateValidatingWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-platform-operator
// validatingWebhookConfiguration resource.
// GIVEN a validatingWebhookConfiguration resource with the CA Bundle set
//  WHEN I call UpdateValidatingnWebhookConfiguration
//  THEN the validatingWebhookConfiguration resource set the CA Bundle as expected
func TestUpdateValidatingWebhookConfiguration(t *testing.T) {
	assert := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/validate-oam-verrazzano-io-v1alpha1-ingresstrait"
	service := adminv1beta1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1beta1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: ValidatingWebhookName,
		},
		Webhooks: []adminv1beta1.ValidatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1beta1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	assert.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateValidatingWebhookConfiguration(kubeClient, &caCert)
	assert.Nil(err, "error should not be returned updating validation webhook configuration")

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Get(context.TODO(), ValidatingWebhookName, metav1.GetOptions{})
	assert.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateValidatingWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-platform-operator validatingWebhookConfiguration resource.
// GIVEN an invalid validatingWebhookConfiguration resource with the CA Bundle set
//  WHEN I call UpdateValidatingnWebhookConfiguration
//  THEN the validatingWebhookConfiguration resource will fail to be updated
func TestUpdateValidatingWebhookConfigurationFail(t *testing.T) {
	assert := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/validate-oam-verrazzano-io-v1alpha1-ingresstrait"
	service := adminv1beta1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1beta1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "InvalidName",
		},
		Webhooks: []adminv1beta1.ValidatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1beta1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1beta1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	assert.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateValidatingWebhookConfiguration(kubeClient, &caCert)
	assert.Error(err, "error should be returned updating validation webhook configuration")
}

// TestUpdateAppConfigMutatingWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-application-operator
// mutatingWebhookConfiguration resource.
// GIVEN a mutatingWebhookConfiguration resource with the CA Bundle set
//  WHEN I call UpdateAppConfigMutatingWebhookConfiguration
//  THEN the mutatingWebhookConfiguration resource set the CA Bundle as expected
func TestUpdateAppConfigMutatingWebhookConfiguration(t *testing.T) {
	assert := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/appconfig-defaulter"
	service := adminv1beta1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: AppConfigMutatingWebhookName,
		},
		Webhooks: []adminv1beta1.MutatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1beta1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	assert.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateAppConfigMutatingWebhookConfiguration(kubeClient, &caCert)
	assert.Nil(err, "error should not be returned updating validation webhook configuration")

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.TODO(), AppConfigMutatingWebhookName, metav1.GetOptions{})
	assert.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateAppConfigMutatingWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-application-operator mutatingWebhookConfiguration resource.
// GIVEN an invalid mutatingWebhookConfiguration resource with the CA Bundle set
//  WHEN I call UpdateAppConfigMutatingWebhookConfiguration
//  THEN the mutatingWebhookConfiguration resource will fail to be updated
func TestUpdateAppConfigMutatingWebhookConfigurationFail(t *testing.T) {
	assert := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/appconfig-defaulter"
	service := adminv1beta1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "InvalidName",
		},
		Webhooks: []adminv1beta1.MutatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1beta1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	assert.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateAppConfigMutatingWebhookConfiguration(kubeClient, &caCert)
	assert.Error(err, "error should be returned updating validation webhook configuration")
}

// TestUpdateIstioMutatingWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-application-operator
// mutatingWebhookConfiguration resource.
// GIVEN a mutatingWebhookConfiguration resource with the CA Bundle set
//  WHEN I call UpdateIstioMutatingWebhookConfiguration
//  THEN the mutatingWebhookConfiguration resource set the CA Bundle as expected
func TestUpdateIstioMutatingWebhookConfiguration(t *testing.T) {
	assert := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/istio-defaulter"
	service := adminv1beta1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: IstioMutatingWebhookName,
		},
		Webhooks: []adminv1beta1.MutatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1beta1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	assert.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateIstioMutatingWebhookConfiguration(kubeClient, &caCert)
	assert.Nil(err, "error should not be returned updating validation webhook configuration")

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.TODO(), IstioMutatingWebhookName, metav1.GetOptions{})
	assert.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateIstioMutatingWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-application-operator mutatingWebhookConfiguration resource.
// GIVEN an invalid mutatingWebhookConfiguration resource with the CA Bundle set
//  WHEN I call UpdateIstioMutatingWebhookConfiguration
//  THEN the mutatingWebhookConfiguration resource will fail to be updated
func TestUpdateIstioMutatingWebhookConfigurationFail(t *testing.T) {
	assert := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/istio-defaulter"
	service := adminv1beta1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1beta1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "InvalidName",
		},
		Webhooks: []adminv1beta1.MutatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1beta1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	assert.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateIstioMutatingWebhookConfiguration(kubeClient, &caCert)
	assert.Error(err, "error should be returned updating validation webhook configuration")
}
