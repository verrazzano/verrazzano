// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"net/http"

	k8sadmission "k8s.io/api/admission/v1beta1"
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
	p := &MultiClusterApplicationConfiguration{}
	err := v.decoder.Decode(req, p)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	log.Info("client=%v", v.client)
	log.Info("operation=%v", req.Operation)
	log.Info("object=%v", p)

	switch req.Operation {
	case k8sadmission.Create:
		return translateErrorToResponse(validateMultiClusterApplicationConfiguration(v.client, p))
	case k8sadmission.Update:
		return translateErrorToResponse(validateMultiClusterApplicationConfiguration(v.client, p))
	default:
		return admission.Allowed("")
	}
}

// validateMultiClusterApplicationConfiguration performs validation checks on the resource
func validateMultiClusterApplicationConfiguration(c client.Client, mc *MultiClusterApplicationConfiguration) error {
	return nil
}

// TODO: remove when synced up with VerrazzanoProject changes
// translateErrorToResponse translates an error to an admission.Response
func translateErrorToResponse(err error) admission.Response {
	if err == nil {
		return admission.Allowed("")
	}
	return admission.Denied(err.Error())
}
