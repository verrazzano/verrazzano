// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"
	"github.com/verrazzano/verrazzano/application-operator/internal/certificates"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// updateValidatingWebhookConfiguration sets the CABundle
func updateValidatingWebhookConfiguration(kubeClient kubernetes.Interface, name string) error {
	validatingWebhook, err := kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	caSecret, errX := kubeClient.CoreV1().Secrets(certificates.OperatorNamespace).Get(context.TODO(), certificates.OperatorCA, metav1.GetOptions{})
	if errX != nil {
		return errX
	}

	crt := caSecret.Data[certificates.CertKey]
	for i := range validatingWebhook.Webhooks {
		validatingWebhook.Webhooks[i].ClientConfig.CABundle = crt
	}

	_, err = kubeClient.AdmissionregistrationV1().ValidatingWebhookConfigurations().Update(context.TODO(), validatingWebhook, metav1.UpdateOptions{})
	return err
}

// updateMutatingWebhookConfiguration sets the CABundle
func updateMutatingWebhookConfiguration(kubeClient kubernetes.Interface, name string) error {
	mutatingWebhook, err := kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	caSecret, errX := kubeClient.CoreV1().Secrets(certificates.OperatorNamespace).Get(context.TODO(), certificates.OperatorCA, metav1.GetOptions{})
	if errX != nil {
		return errX
	}

	crt := caSecret.Data[certificates.CertKey]
	for i := range mutatingWebhook.Webhooks {
		mutatingWebhook.Webhooks[i].ClientConfig.CABundle = crt
	}

	_, err = kubeClient.AdmissionregistrationV1().MutatingWebhookConfigurations().Update(context.TODO(), mutatingWebhook, metav1.UpdateOptions{})
	return err
}
