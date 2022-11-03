// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	k8sadmission "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// MultiClusterApplicationConfigurationValidator is a struct holding objects used during validation.
type MultiClusterApplicationConfigurationValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// InjectClient injects the client.
func (v *MultiClusterApplicationConfigurationValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *MultiClusterApplicationConfigurationValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of created or updated MultiClusterApplicationConfiguration resources.
func (v *MultiClusterApplicationConfigurationValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	counterMetricObject, errorCounterMetricObject, handleDurationMetricObject, zapLogForMetrics, err := metricsexporter.ExposeControllerMetrics("MultiClusterApplicationConfigurationValidator", metricsexporter.MultiClusterAppconfigPodHandleCounter, metricsexporter.MultiClusterAppconfigPodHandleError, metricsexporter.MultiClusterAppconfigPodHandleDuration)
	if err != nil {
		return admission.Response{}
	}
	handleDurationMetricObject.TimerStart()
	defer handleDurationMetricObject.TimerStop()

	mcac := &v1alpha1.MultiClusterApplicationConfiguration{}
	err = v.decoder.Decode(req, mcac)
	if err != nil {
		errorCounterMetricObject.Inc(zapLogForMetrics, err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	if mcac.ObjectMeta.DeletionTimestamp.IsZero() {
		switch req.Operation {
		case k8sadmission.Create, k8sadmission.Update:
			err = validateMultiClusterResource(v.client, mcac)
			if err != nil {
				errorCounterMetricObject.Inc(zapLogForMetrics, err)
				return admission.Denied(err.Error())
			}
			err = v.validateSecrets(mcac)
			if err != nil {
				errorCounterMetricObject.Inc(zapLogForMetrics, err)
				return admission.Denied(err.Error())
			}
		}
	}
	counterMetricObject.Inc(zapLogForMetrics, err)
	return admission.Allowed("")
}

// Validate that the secrets referenced in the MultiClusterApplicationConfiguration resource exist in the
// same namespace as the MultiClusterApplicationConfiguration resource.
func (v *MultiClusterApplicationConfigurationValidator) validateSecrets(mcac *v1alpha1.MultiClusterApplicationConfiguration) error {
	if len(mcac.Spec.Secrets) == 0 {
		return nil
	}

	secrets := corev1.SecretList{}
	listOptions := &client.ListOptions{Namespace: mcac.Namespace}
	err := v.client.List(context.TODO(), &secrets, listOptions)
	if err != nil {
		return err
	}

	var secretsNotFound []string
	for _, mcSecret := range mcac.Spec.Secrets {
		found := false
		for _, secret := range secrets.Items {
			if secret.Name == mcSecret {
				found = true
				break
			}
		}
		if !found {
			secretsNotFound = append(secretsNotFound, mcSecret)
		}
	}

	if len(secretsNotFound) != 0 {
		secretsDelimited := strings.Join(secretsNotFound, ",")
		return fmt.Errorf("secret(s) %s specified in MultiClusterApplicationConfiguration not found in namespace %s", secretsDelimited, mcac.Namespace)
	}

	return nil
}
