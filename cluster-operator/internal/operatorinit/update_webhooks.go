// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operatorinit

import (
	"context"

	"github.com/verrazzano/verrazzano/cluster-operator/internal/certificate"
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
