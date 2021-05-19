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
	p := &MultiClusterSecret{}
	err := v.decoder.Decode(req, p)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	log.Info("client=%v", v.client)
	log.Info("operation=%v", req.Operation)
	log.Info("object=%v", p)

	switch req.Operation {
	case k8sadmission.Create:
		return translateErrorToResponse(validateMultiClusterSecret(v.client, p))
	case k8sadmission.Update:
		return translateErrorToResponse(validateMultiClusterSecret(v.client, p))
	default:
		return admission.Allowed("")
	}
}

// validateMultiClusterSecret performs validation checks on the resource
func validateMultiClusterSecret(c client.Client, mc *MultiClusterSecret) error {
	return nil
}
