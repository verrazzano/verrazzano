// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

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
	adminv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestCreateWebhookCertificates tests that the certificates needed for webhooks are created
// GIVEN an output directory for certificates
//  WHEN I call CreateWebhookCertificates
//  THEN all the needed certificate artifacts are created
func TestCreateWebhookCertificates(t *testing.T) {
	assert := assert.New(t)

	dir, err := ioutil.TempDir("", "certs")
	if err != nil {
		assert.Nil(err, "error should not be returned creating temporary directory")
	}
	defer os.RemoveAll(dir)
	caBundle, err := CreateWebhookCertificates(dir)
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
			assert.Equal("verrazzano-platform-operator.verrazzano-install.svc", cert.DNSNames[0])
		}
	}
}

// TestCreateWebhookCertificatesFail tests that the certificates needed for webhooks are not created
// GIVEN an invalid output directory for certificates
//  WHEN I call CreateWebhookCertificates
//  THEN all the needed certificate artifacts are not created
func TestCreateWebhookCertificatesFail(t *testing.T) {
	assert := assert.New(t)

	_, err := CreateWebhookCertificates("/bad-dir")
	assert.Error(err, "error should be returned setting up certificates")
}

// TestUpdateValidatingnWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-platform-operator
// validatingWebhookConfiguration resource.
// GIVEN a validatingWebhookConfiguration resource with the CA Bundle set
//  WHEN I call UpdateValidatingnWebhookConfiguration
//  THEN the validatingWebhookConfiguration resource set the CA Bundle as expected
func TestUpdateValidatingnWebhookConfiguration(t *testing.T) {
	assert := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	pathInstall := "/validate-install-verrazzano-io-v1alpha1-verrazzano"
	serviceInstall := adminv1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &pathInstall,
	}
	pathClusters := "/validate-clusters-verrazzano-io-v1alpha1-verrazzanomanagedcluster"
	serviceClusters := adminv1.ServiceReference{
		Name:      OperatorName,
		Namespace: OperatorNamespace,
		Path:      &pathClusters,
	}
	webhook := adminv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: OperatorName,
		},
		Webhooks: []adminv1.ValidatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &serviceInstall,
				},
			},
			{
				Name: "clusters.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &serviceClusters,
				},
			},
			{
				Name: "install.verrazzano.io.v1beta",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &serviceInstall,
				},
			},
		},
	}

	_, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	assert.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateValidatingnWebhookConfiguration(kubeClient, &caCert)
	assert.Nil(err, "error should not be returned updating validation webhook configuration")

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), "verrazzano-platform-operator", metav1.GetOptions{})
	assert.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateValidatingnWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-platform-operator validatingWebhookConfiguration resource.
// GIVEN an invalid validatingWebhookConfiguration resource with the CA Bundle set
//  WHEN I call UpdateValidatingnWebhookConfiguration
//  THEN the validatingWebhookConfiguration resource will fail to be updated
func TestUpdateValidatingnWebhookConfigurationFail(t *testing.T) {
	assert := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	path := "/validate-install-verrazzano-io-v1alpha1-verrazzano"
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
	assert.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateValidatingnWebhookConfiguration(kubeClient, &caCert)
	assert.Error(err, "error should be returned updating validation webhook configuration")
}
