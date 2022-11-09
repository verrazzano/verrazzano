// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"

	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/certificate"
	adminv1 "k8s.io/api/admissionregistration/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// deleteValidatingWebhookConfiguration deletes a validating webhook configuration
func deleteValidatingWebhookConfiguration(kubeClient kubernetes.Interface, name string) error {
	_, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Delete(context.TODO(), name, metav1.DeleteOptions{})
}

// updateValidatingWebhookConfiguration sets the CABundle
func updateValidatingWebhookConfiguration(kubeClient kubernetes.Interface, name string) error {
	validatingWebhook, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	caSecret, errX := kubeClient.CoreV1().Secrets(certificate.OperatorNamespace).Get(context.TODO(), certificate.OperatorCA, metav1.GetOptions{})
	if errX != nil {
		return errX
	}

	crt := caSecret.Data[certificate.CertKey]
	for i := range validatingWebhook.Webhooks {
		validatingWebhook.Webhooks[i].ClientConfig.CABundle = crt
	}

	_, err = kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(context.TODO(), validatingWebhook, metav1.UpdateOptions{})
	return err
}

// updateConversionWebhookConfiguration sets the conversion webhook for the Verrazzano resource
func updateConversionWebhookConfiguration(apiextClient apiextensionsv1client.ApiextensionsV1Interface, kubeClient kubernetes.Interface) error {
	crd, err := apiextClient.CustomResourceDefinitions().Get(context.TODO(), certificate.CRDName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	convertPath := "/convert"
	var webhookPort int32 = 443
	caSecret, err := kubeClient.CoreV1().Secrets(certificate.OperatorNamespace).Get(context.TODO(), certificate.OperatorCA, metav1.GetOptions{})
	if err != nil {
		return err
	}

	crt := caSecret.Data[certificate.CertKey]
	crd.Spec.Conversion = &apiextensionsv1.CustomResourceConversion{
		Strategy: apiextensionsv1.WebhookConverter,
		Webhook: &apiextensionsv1.WebhookConversion{
			ClientConfig: &apiextensionsv1.WebhookClientConfig{
				Service: &apiextensionsv1.ServiceReference{
					Name:      certificate.OperatorName,
					Namespace: certificate.OperatorNamespace,
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

// updateMutatingWebhookConfiguration sets the CABundle
func updateMutatingWebhookConfiguration(kubeClient kubernetes.Interface, name string) error {
	var webhook *adminv1.MutatingWebhookConfiguration
	webhook, err := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	caSecret, err := kubeClient.CoreV1().Secrets(certificate.OperatorNamespace).Get(context.TODO(), certificate.OperatorCA, metav1.GetOptions{})
	if err != nil {
		return err
	}
	crt := caSecret.Data[certificate.CertKey]
	for i := range webhook.Webhooks {
		webhook.Webhooks[i].ClientConfig.CABundle = crt
	}
	_, err = kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.TODO(), webhook, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}
