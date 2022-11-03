// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"bytes"
	"context"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"testing"

	"github.com/stretchr/testify/assert"
	adminv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// TestCreateWebhookCertificates tests that the certificates needed for webhooks are created
// GIVEN an output directory for certificates
//
//	WHEN I call CreateWebhookCertificates
//	THEN all the needed certificate artifacts are created
func TestCreateWebhookCertificates(t *testing.T) {
	asserts := assert.New(t)
	client := fake.NewSimpleClientset()
	_ = CreateWebhookCertificates(zap.S(), client, "/etc/webhook/certs")
	var secret *v1.Secret
	secret, err := client.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorCA, metav1.GetOptions{})
	_, err2 := client.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorTLS, metav1.GetOptions{})
	asserts.Nil(err, "error should not be returned setting up certificates")
	asserts.Nil(err2, "error should not be returned setting up certificates")
	asserts.NotEmpty(string(secret.Data[certKey]))
}

// TestUpdateValidatingnWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-platform-operator
// validatingWebhookConfiguration resource.
// GIVEN a validatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call UpdateValidatingWebhookConfiguration
//	THEN the validatingWebhookConfiguration resource set the CA Bundle as expected
func TestUpdateValidatingnWebhookConfiguration(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")

	caSecret := v1.Secret{}
	caSecret.Name = OperatorCA
	caSecret.Type = v1.SecretTypeTLS
	caSecret.Namespace = OperatorNamespace
	caSecret.Data = make(map[string][]byte)
	caSecret.Data[certKey] = caCert.Bytes()
	caSecret.Data[privKey] = caCert.Bytes()

	kubeClient.CoreV1().Secrets(OperatorNamespace).Create(context.TODO(), &caSecret, metav1.CreateOptions{})

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

	wh, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
	asserts.Nil(err, "error should not be returned creating validation webhook configuration")
	asserts.NotEmpty(wh)

	err = UpdateValidatingWebhookConfiguration(kubeClient, OperatorName)
	asserts.Nil(err, "error should not be returned updating validation webhook configuration")

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), "verrazzano-platform-operator-webhook", metav1.GetOptions{})
	asserts.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateValidatingnWebhookConfigurationFail tests that the CA Bundle is not updated in the
// verrazzano-platform-operator validatingWebhookConfiguration resource.
// GIVEN an invalid validatingWebhookConfiguration resource with the CA Bundle set
//
//	WHEN I call UpdateValidatingWebhookConfiguration
//	THEN the validatingWebhookConfiguration resource will fail to be updated
func TestUpdateValidatingnWebhookConfigurationFail(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")
	caSecret := v1.Secret{}
	caSecret.Name = OperatorCA
	caSecret.Type = v1.SecretTypeTLS
	caSecret.Namespace = OperatorNamespace
	caSecret.Data = make(map[string][]byte)
	caSecret.Data[certKey] = caCert.Bytes()
	caSecret.Data[privKey] = caCert.Bytes()

	kubeClient.CoreV1().Secrets(OperatorNamespace).Create(context.TODO(), &caSecret, metav1.CreateOptions{})

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
	asserts.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateValidatingWebhookConfiguration(kubeClient, OperatorName)
	asserts.Error(err, "error should be returned updating validation webhook configuration")
}
