// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/certificate"
	adminv1 "k8s.io/api/admissionregistration/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestUpdateValidatingnWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-cluster-operator
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

	wh, err := createExpectedValidatingWebhook(kubeClient, certificate.WebhookName)
	asserts.Nilf(err, "error should not be returned creating validation webhook configuration: %v", err)
	asserts.NotEmpty(wh)

	err = updateValidatingWebhookConfiguration(kubeClient, certificate.WebhookName)
	asserts.Nilf(err, "error should not be returned updating validation webhook configuration: %v", err)

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), "verrazzano-cluster-operator-webhook", metav1.GetOptions{})
	asserts.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateValidatingnWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-cluster-operator validatingWebhookConfiguration resource.
// GIVEN an invalid validatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call updateValidatingWebhookConfiguration
//	THEN the validatingWebhookConfiguration resource will fail to be updated
func TestUpdateValidatingnWebhookConfigurationFail(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	_, _, err := createExpectedCASecret(kubeClient)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	_, err = createInvalidExpectedValidatingWebhook(kubeClient, certificate.WebhookName)
	asserts.Nil(err, "error should not be returned creating validation webhook configuration")

	err = updateValidatingWebhookConfiguration(kubeClient, certificate.WebhookName)
	asserts.Error(err, "error should be returned updating validation webhook configuration")
}

func createExpectedCASecret(kubeClient *fake.Clientset) (*v1.Secret, bytes.Buffer, error) {
	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")

	caSecret := v1.Secret{}
	caSecret.Name = certificate.OperatorCA
	caSecret.Type = v1.SecretTypeTLS
	caSecret.Namespace = certificate.WebhookNamespace
	caSecret.Data = make(map[string][]byte)
	caSecret.Data[certificate.CertKey] = caCert.Bytes()
	caSecret.Data[certificate.PrivKey] = caCert.Bytes()

	newSecret, err := kubeClient.CoreV1().Secrets(certificate.WebhookNamespace).Create(context.TODO(), &caSecret, metav1.CreateOptions{})
	return newSecret, caCert, err
}

func createExpectedValidatingWebhook(kubeClient *fake.Clientset, whName string) (*adminv1.ValidatingWebhookConfiguration, error) {
	pathInstall := "/validate-clusters-verrazzano-io-v1alpha1-verrazzanomanagedcluster"
	serviceInstall := adminv1.ServiceReference{
		Name:      whName,
		Namespace: certificate.WebhookNamespace,
		Path:      &pathInstall,
	}

	webhook := adminv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: whName,
		},
		Webhooks: []adminv1.ValidatingWebhook{
			{
				Name: "install.verrazzano.io",
				ClientConfig: adminv1.WebhookClientConfig{
					Service: &serviceInstall,
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
	return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
}

func createInvalidExpectedValidatingWebhook(kubeClient *fake.Clientset, whName string) (*adminv1.ValidatingWebhookConfiguration, error) {
	path := "/validate-clusters-verrazzano-io-v1alpha1-verrazzanomanagedcluster"
	service := adminv1.ServiceReference{
		Name:      whName,
		Namespace: certificate.WebhookNamespace,
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
	return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
}
