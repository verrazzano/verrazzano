// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package certificate

import (
	"bytes"
	"context"
	"errors"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	fakeapiExt "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	fake2 "k8s.io/client-go/kubernetes/typed/admissionregistration/v1/fake"
	testing2 "k8s.io/client-go/testing"
	"os"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"strings"
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

	// create temp dir for certs
	tempDir, err := os.MkdirTemp("", "certs")
	t.Logf("Using temp dir %s", tempDir)
	defer func() {
		err := os.RemoveAll(tempDir)
		if err != nil {
			t.Logf("Error removing tempdir %s", tempDir)
		}
	}()
	asserts.Nil(err)

	err = CreateWebhookCertificates(zap.S(), client, tempDir)
	asserts.Nil(err)

	// Verify generated certs
	cert1 := validateFile(asserts, tempDir+"/tls.crt", "-----BEGIN CERTIFICATE-----")
	key1 := validateFile(asserts, tempDir+"/tls.key", "-----BEGIN RSA PRIVATE KEY-----")

	// Verify generated secrets
	var secret *v1.Secret
	secret, err = client.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorCA, metav1.GetOptions{})
	_, err2 := client.CoreV1().Secrets(OperatorNamespace).Get(context.TODO(), OperatorTLS, metav1.GetOptions{})
	asserts.Nil(err, "error should not be returned setting up certificates")
	asserts.Nil(err2, "error should not be returned setting up certificates")
	asserts.NotEmpty(string(secret.Data[certKey]))

	// create temp dir for certs
	tempDir2, err := os.MkdirTemp("", "certs")
	t.Logf("Using temp dir %s", tempDir2)
	defer func() {
		err := os.RemoveAll(tempDir2)
		if err != nil {
			t.Logf("Error removing tempdir %s", tempDir2)
		}
	}()
	asserts.Nil(err)

	// Call it again, should create new certs in location with identical contents
	err = CreateWebhookCertificates(zap.S(), client, tempDir2)
	asserts.Nil(err)
	asserts.Nil(err)

	// Verify generated certs
	cert2 := validateFile(asserts, tempDir+"/tls.crt", "-----BEGIN CERTIFICATE-----")
	key2 := validateFile(asserts, tempDir+"/tls.key", "-----BEGIN RSA PRIVATE KEY-----")

	asserts.Equalf(cert1, cert2, "Certs did not match")
	asserts.Equalf(key1, key2, "Keys did not match")
}

func validateFile(asserts *assert.Assertions, certFile string, certPrefix string) []byte {
	file, err := ioutil.ReadFile(certFile)
	asserts.Nilf(err, "Error reading file", certFile)
	asserts.True(strings.Contains(string(file), certPrefix))
	return file
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

	caSecret, caCert, err := createExpectedCASecret(kubeClient, asserts)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	kubeClient.CoreV1().Secrets(OperatorNamespace).Create(context.TODO(), caSecret, metav1.CreateOptions{})

	wh, err := createExpectedValidatingWebhook(kubeClient, OperatorName)
	asserts.Nilf(err, "error should not be returned creating validation webhook configuration: %v", err)
	asserts.NotEmpty(wh)

	err = UpdateValidatingWebhookConfiguration(kubeClient, OperatorName)
	asserts.Nilf(err, "error should not be returned updating validation webhook configuration: %v", err)

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

	_, _, err := createExpectedCASecret(kubeClient, asserts)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	_, err = createInvalidExpectedValidatingWebhook(kubeClient, OperatorName)
	asserts.Nil(err, "error should not be returned creating validation webhook configuration")

	err = UpdateValidatingWebhookConfiguration(kubeClient, OperatorName)
	asserts.Error(err, "error should be returned updating validation webhook configuration")
}

// TestUpdateConversionWebhookConfiguration tests that the CA Bundle is updated in the verrazzano-platform-operator
// ConversionWebhookConfiguration resource.
// GIVEN a call to UpdateConversionWebhookConfiguration
//
//	WHEN the webhook CA bundle is present
//	THEN the CRD is updated with a Webhook converter configuration for the v1beta1 review versions with the correct CA Bundle
func TestUpdateConversionWebhookConfiguration(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	_, caCert, err := createExpectedCASecret(kubeClient, asserts)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	apiExtClient := fakeapiExt.NewSimpleClientset().ApiextensionsV1()
	_, err = apiExtClient.CustomResourceDefinitions().Create(context.TODO(), &v12.CustomResourceDefinition{
		ObjectMeta: controllerruntime.ObjectMeta{Name: CRDName},
	}, metav1.CreateOptions{})
	asserts.Nilf(err, "Unexpected error creating mock CRD: %v", err)

	err = UpdateConversionWebhookConfiguration(apiExtClient, kubeClient)
	asserts.Nilf(err, "Unexpected error returned updating validation webhook configuration: %v", err)

	updatedCRD, err := apiExtClient.CustomResourceDefinitions().Get(context.TODO(), CRDName, metav1.GetOptions{})
	asserts.Nilf(err, "Unexpected error getting updated CRD: %v", err)

	asserts.Equal(caCert.Bytes(), updatedCRD.Spec.Conversion.Webhook.ClientConfig.CABundle, "Expected CA bundle name did not match")
	asserts.Equal(OperatorName, updatedCRD.Spec.Conversion.Webhook.ClientConfig.Service.Name)
	asserts.Equal(OperatorNamespace, updatedCRD.Spec.Conversion.Webhook.ClientConfig.Service.Namespace)
	asserts.Equal("/convert", *updatedCRD.Spec.Conversion.Webhook.ClientConfig.Service.Path)
	asserts.Equal(int32(443), *updatedCRD.Spec.Conversion.Webhook.ClientConfig.Service.Port)
	asserts.Equal(v12.WebhookConverter, updatedCRD.Spec.Conversion.Strategy)
	asserts.Equal([]string{"v1beta1"}, updatedCRD.Spec.Conversion.Webhook.ConversionReviewVersions)
}

// TestUpdateMutatingWebhookConfiguration tests that the CA Bundle is updated the specified MutatingWebhook configuration
// GIVEN a call to UpdateMutatingWebhookConfiguration
//
//	WHEN with the webhook CA bundle secret exists
//	THEN the MutatingWebhook configuration sets the CA bundle on all webhook client configurations
func TestUpdateMutatingWebhookConfiguration(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	_, caCert, err := createExpectedCASecret(kubeClient, asserts)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	_, err = createExpectedMutatingWebhook(kubeClient)
	asserts.Nilf(err, "Unexpected error creating expected webhook configuration")

	err = UpdateMutatingWebhookConfiguration(kubeClient, constants.MysqlBackupMutatingWebhookName)
	asserts.Nilf(err, "Unexpected error returned from UpdateMutatingWebhookConfiguration: %v", err)

	updatedWebhook, _ := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), constants.MysqlBackupMutatingWebhookName, metav1.GetOptions{})
	asserts.Equal(caCert.Bytes(), updatedWebhook.Webhooks[0].ClientConfig.CABundle, "Expected CA bundle name did not match")
	asserts.Equal(caCert.Bytes(), updatedWebhook.Webhooks[1].ClientConfig.CABundle, "Expected CA bundle name did not match")
}

// TestUpdateMutatingWebhookConfigurationNoCASecret tests that
// GIVEN a call to UpdateMutatingWebhookConfiguration
//
//	WHEN with the webhook CA bundle secret does not exist but the webhook does
//	THEN an error is returned
func TestUpdateMutatingWebhookConfigurationNoCASecret(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	_, err := createExpectedMutatingWebhook(kubeClient)
	asserts.Nilf(err, "Unexpected error creating expected webhook configuration", err)

	err = UpdateMutatingWebhookConfiguration(kubeClient, constants.MysqlBackupMutatingWebhookName)
	asserts.NotNil(err, "No error returned when webhook doesn't exist")
}

// TestUpdateMutatingWebhookConfigurationNoWebhook tests that
// GIVEN a call to UpdateMutatingWebhookConfiguration
//
//	WHEN with the webhook CA bundle secret exists but the webhook does not
//	THEN an error is returned
func TestUpdateMutatingWebhookConfigurationNoWebhook(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	_, _, err := createExpectedCASecret(kubeClient, asserts)
	asserts.Nilf(err, "Unexpected error creating expected CA secret", err)

	err = UpdateMutatingWebhookConfiguration(kubeClient, constants.MysqlBackupMutatingWebhookName)
	asserts.NotNil(err, "No error returned when webhook doesn't exist")
}

// TestDeleteValidatingWebhookConfiguration tests that
// GIVEN a call to DeleteValidatingWebhookConfiguration
//
//	WHEN the webhook does exist
//	THEN no error is returned
func TestDeleteValidatingWebhookConfiguration(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()

	const webhookName = "foo"
	_, err := createExpectedValidatingWebhook(kubeClient, webhookName)
	asserts.Nilf(err, "Unexpected error creating expected webhook configuration")

	err = DeleteValidatingWebhookConfiguration(kubeClient, webhookName)
	asserts.Nilf(err, "Unexpected error when deleting webhook", err)

	wh, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), webhookName, metav1.GetOptions{})
	asserts.NotNil(err, "Did not get expected error after delete")
	asserts.True(errors2.IsNotFound(err), "Did not get IsNotFound error after delete")
	asserts.Nilf(wh, "Webhook reference should be nil after delete")
}

// TestDeleteValidatingWebhookConfigurationDoesNotExist tests that
// GIVEN a call to DeleteValidatingWebhookConfiguration
//
//	WHEN the webhook does NOT exist
//	THEN no error is returned
func TestDeleteValidatingWebhookConfigurationDoesNotExist(t *testing.T) {
	asserts := assert.New(t)
	kubeClient := fake.NewSimpleClientset()
	err := DeleteValidatingWebhookConfiguration(kubeClient, "foo")
	asserts.Nilf(err, "Unexpected error when deleting webhook that does not exist", err)
}

// TestDeleteValidatingWebhookConfigurationErrorOnGet tests that
// GIVEN a call to DeleteValidatingWebhookConfiguration
//
//	WHEN the webhook Get() operation returns an unexpected error
//	THEN that error is returned
func TestDeleteValidatingWebhookConfigurationErrorOnGet(t *testing.T) {
	asserts := assert.New(t)

	kubeClient := fake.NewSimpleClientset()
	kubeClient.AdmissionregistrationV1().(*fake2.FakeAdmissionregistrationV1).
		PrependReactor("get", "validatingwebhookconfigurations",
			func(action testing2.Action) (handled bool, ret runtime.Object, err error) {
				return true, nil, errors.New("Error deleting validating webhook")
			})

	err := DeleteValidatingWebhookConfiguration(kubeClient, "foo")
	asserts.NotNilf(err, "No expected error returned when deleting webhook", err)
}

func createExpectedCASecret(kubeClient *fake.Clientset, asserts *assert.Assertions) (*v1.Secret, bytes.Buffer, error) {
	var caCert bytes.Buffer
	caCert.WriteString("Fake CABundle")

	caSecret := v1.Secret{}
	caSecret.Name = OperatorCA
	caSecret.Type = v1.SecretTypeTLS
	caSecret.Namespace = OperatorNamespace
	caSecret.Data = make(map[string][]byte)
	caSecret.Data[certKey] = caCert.Bytes()
	caSecret.Data[privKey] = caCert.Bytes()

	newSecret, err := kubeClient.CoreV1().Secrets(OperatorNamespace).Create(context.TODO(), &caSecret, metav1.CreateOptions{})
	return newSecret, caCert, err
}

func createExpectedMutatingWebhook(kubeClient *fake.Clientset) (*adminv1.MutatingWebhookConfiguration, error) {
	webhook := adminv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: constants.MysqlBackupMutatingWebhookName,
		},
		Webhooks: []adminv1.MutatingWebhook{
			{Name: "webhook1"},
			{Name: "webhook2"},
		},
	}
	return kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
}

func createExpectedValidatingWebhook(kubeClient *fake.Clientset, whName string) (*adminv1.ValidatingWebhookConfiguration, error) {
	pathInstall := "/validate-install-verrazzano-io-v1alpha1-verrazzano"
	serviceInstall := adminv1.ServiceReference{
		Name:      whName,
		Namespace: OperatorNamespace,
		Path:      &pathInstall,
	}
	pathClusters := "/validate-clusters-verrazzano-io-v1alpha1-verrazzanomanagedcluster"
	serviceClusters := adminv1.ServiceReference{
		Name:      whName,
		Namespace: OperatorNamespace,
		Path:      &pathClusters,
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
	return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
}

func createInvalidExpectedValidatingWebhook(kubeClient *fake.Clientset, whName string) (*adminv1.ValidatingWebhookConfiguration, error) {
	path := "/validate-install-verrazzano-io-v1alpha1-verrazzano"
	service := adminv1.ServiceReference{
		Name:      whName,
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
	return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(context.TODO(), &webhook, metav1.CreateOptions{})
}
