// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"net/http"

	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	k8sadmission "k8s.io/api/admission/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// MultiClusterSecretValidator is a struct holding objects used during validation.
type MultiClusterSecretValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// InjectClient injects the client.
func (v *MultiClusterSecretValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *MultiClusterSecretValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of created or updated MultiClusterSecret resources.
func (v *MultiClusterSecretValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	counterMetricObject, errorCounterMetricObject, handleDurationMetricObject, zapLogForMetrics, err := metricsexporter.ExposeControllerMetrics("MultiClusterSecretValidator", metricsexporter.MultiClusterSecretHandleCounter, metricsexporter.MultiClusterSecretHandleError, metricsexporter.MultiClusterSecretHandleDuration)
	if err != nil {
		return admission.Response{}
	}
	handleDurationMetricObject.TimerStart()
	defer handleDurationMetricObject.TimerStop()

	mcs := &v1alpha1.MultiClusterSecret{}
	err = v.decoder.Decode(req, mcs)
	if err != nil {
		errorCounterMetricObject.Inc(zapLogForMetrics, err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	if mcs.ObjectMeta.DeletionTimestamp.IsZero() {
		switch req.Operation {
		case k8sadmission.Create, k8sadmission.Update:
			err = validateMultiClusterResource(v.client, mcs)
			if err != nil {
				errorCounterMetricObject.Inc(zapLogForMetrics, err)
				return admission.Denied(err.Error())
			}
		}
	}
	counterMetricObject.Inc(zapLogForMetrics, err)
	return admission.Allowed("")
}
