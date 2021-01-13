// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificates

import (
	"bytes"
	"context"
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
	assert.FileExists(fmt.Sprintf("%s/%s", dir, "tls.crt"), "expected tls.crt file not found")
	assert.FileExists(fmt.Sprintf("%s/%s", dir, "tls.key"), "expected tls.key file not found")
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

// TestUpdateMutatingWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-application-operator
// validatingWebhookConfiguration resource.
// GIVEN a validatingWebhookConfiguration resource with the CA Bundle set
//  WHEN I call UpdateValidatingnWebhookConfiguration
//  THEN the validatingWebhookConfiguration resource set the CA Bundle as expected
func TestUpdateMutatingWebhookConfiguration(t *testing.T) {
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
			Name: MutatingWebhookName,
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

	err = UpdateMutatingWebhookConfiguration(kubeClient, &caCert)
	assert.Nil(err, "error should not be returned updating validation webhook configuration")

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(context.TODO(), MutatingWebhookName, metav1.GetOptions{})
	assert.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateMutatingWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-application-operator validatingWebhookConfiguration resource.
// GIVEN an invalid validatingWebhookConfiguration resource with the CA Bundle set
//  WHEN I call UpdateValidatingnWebhookConfiguration
//  THEN the validatingWebhookConfiguration resource will fail to be updated
func TestUpdateMutatingWebhookConfigurationFail(t *testing.T) {
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

	err = UpdateMutatingWebhookConfiguration(kubeClient, &caCert)
	assert.Error(err, "error should be returned updating validation webhook configuration")
}
