// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/internal/certificates"
	adminv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestUpdateValidatingnWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-application-operator
// validatingWebhookConfiguration resource.
// GIVEN a validatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call updateValidatingWebhookConfiguration
//	THEN the validatingWebhookConfiguration resource set the CA Bundle as expected
func TestUpdateValidatingnWebhookConfiguration(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	_, caCert, err := createExpectedCASecret(kubeClient)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	wh, err := createExpectedValidatingWebhook(kubeClient)
	asserts.Nilf(err, "error should not be returned creating validation webhook configuration: %v", err)
	asserts.NotEmpty(wh)

	err = updateValidatingWebhookConfiguration(kubeClient, certificates.VerrazzanoProjectValidatingWebhookName)
	asserts.Nilf(err, "error should not be returned updating validation webhook configuration: %v", err)

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), certificates.VerrazzanoProjectValidatingWebhookName, metav1.GetOptions{})
	asserts.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateValidatingnWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-application-operator validatingWebhookConfiguration resource.
// GIVEN an invalid validatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call updateValidatingWebhookConfiguration
//	THEN the validatingWebhookConfiguration resource will fail to be updated
func TestUpdateValidatingnWebhookConfigurationFail(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	_, _, err := createExpectedCASecret(kubeClient)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	_, err = createInvalidExpectedValidatingWebhook(kubeClient, certificates.VerrazzanoProjectValidatingWebhookName)
	asserts.Nil(err, "error should not be returned creating validation webhook configuration")

	err = updateValidatingWebhookConfiguration(kubeClient, certificates.VerrazzanoProjectValidatingWebhookName)
	asserts.Error(err, "error should be returned updating validation webhook configuration")
}

// TestUpdateMutatingWebhookConfiguration tests that the CA Bundle is updated the specified MutatingWebhook configuration
// GIVEN a call to updateMutatingWebhookConfiguration
//
//	WHEN with the webhook CA bundle secret exists
//	THEN the MutatingWebhook configuration sets the CA bundle on all webhook client configurations
func TestUpdateMutatingWebhookConfiguration(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	_, caCert, err := createExpectedCASecret(kubeClient)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	_, err = createExpectedMutatingWebhook(kubeClient)
	asserts.Nilf(err, "Unexpected error creating expected webhook configuration")

	err = updateMutatingWebhookConfiguration(kubeClient, certificates.IstioMutatingWebhookName)
	asserts.Nilf(err, "Unexpected error returned from updateMutatingWebhookConfiguration: %v", err)

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), certificates.IstioMutatingWebhookName, metav1.GetOptions{})
	asserts.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
	asserts.Equal(caCert.Bytes(), updatedWebhook.Webhooks[1].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateMutatingWebhookConfigurationNoCASecret tests that
// GIVEN a call to updateMutatingWebhookConfiguration
//
//	WHEN with the webhook CA bundle secret does not exist but the webhook does
//	THEN an error is returned
func TestUpdateMutatingWebhookConfigurationNoCASecret(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	_, err := createExpectedMutatingWebhook(kubeClient)
	asserts.Nilf(err, "Unexpected error creating expected webhook configuration", err)

	err = updateMutatingWebhookConfiguration(kubeClient, certificates.IstioMutatingWebhookName)
	asserts.NotNil(err, "No error returned when webhook doesn't exist")
}

// TestUpdateMutatingWebhookConfigurationNoWebhook tests that
// GIVEN a call to updateMutatingWebhookConfiguration
//
//	WHEN with the webhook CA bundle secret exists but the webhook does not
//	THEN an error is returned
func TestUpdateMutatingWebhookConfigurationNoWebhook(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	_, _, err := createExpectedCASecret(kubeClient)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	err = updateMutatingWebhookConfiguration(kubeClient, certificates.IstioMutatingWebhookName)
	asserts.NotNil(err, "No error returned when webhook doesn't exist")
}

func createExpectedCASecret(kubeClient *fake.Clientset) (*v1.Secret, bytes.Buffer, error) {
	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")

	caSecret := v1.Secret{}
	caSecret.Name = certificates.OperatorCA
	caSecret.Type = v1.SecretTypeTLS
	caSecret.Namespace = certificates.OperatorNamespace
	caSecret.Data = make(map[string][]byte)
	caSecret.Data[certificates.CertKey] = caCert.Bytes()
	caSecret.Data[certificates.PrivKey] = caCert.Bytes()

	newSecret, err := kubeClient.CoreV1().Secrets(certificates.OperatorNamespace).Create(context.TODO(), &caSecret, metav1.CreateOptions{})
	return newSecret, caCert, err
}

func createExpectedMutatingWebhook(kubeClient *fake.Clientset) (*adminv1.MutatingWebhookConfiguration, error) {
	webhook := adminv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: certificates.IstioMutatingWebhookName,
		},
		Webhooks: []adminv1.MutatingWebhook{
			{Name: "webhook1"},
			{Name: "webhook2"},
		},
	}
	return kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
}

func createExpectedValidatingWebhook(kubeClient *fake.Clientset) (*adminv1.ValidatingWebhookConfiguration, error) {
	path := "/validate-clusters-verrazzano-io-v1alpha1-verrazzanoproject"
	service := adminv1.ServiceReference{
		Name:      certificates.VerrazzanoProjectValidatingWebhookName,
		Namespace: certificates.OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: certificates.VerrazzanoProjectValidatingWebhookName,
		},
		Webhooks: []adminv1.ValidatingWebhook{
			{
				Name: "verrazzano-clusters-verrazzanoproject-validator.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}
	return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
}

func createInvalidExpectedValidatingWebhook(kubeClient *fake.Clientset, whName string) (*adminv1.ValidatingWebhookConfiguration, error) {
	path := "/validate-clusters-verrazzano-io-v1alpha1-verrazzanoproject"
	service := adminv1.ServiceReference{
		Name:      certificates.VerrazzanoProjectValidatingWebhookName,
		Namespace: certificates.OperatorNamespace,
		Path:      &path,
	}
	webhook := adminv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "InvalidName",
		},
		Webhooks: []adminv1.ValidatingWebhook{
			{
				Name: "verrazzano-clusters-verrazzanoproject-validator.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &service,
				},
			},
		},
	}
	return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
}
