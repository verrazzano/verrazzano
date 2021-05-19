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

// MultiClusterConfigmapValidator is a struct holding objects used during validation.
type MultiClusterConfigmapValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// InjectClient injects the client.
func (v *MultiClusterConfigmapValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *MultiClusterConfigmapValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of created or updated MultiClusterConfigmap resources.
func (v *MultiClusterConfigmapValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	p := &MultiClusterConfigMap{}
	err := v.decoder.Decode(req, p)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	log.Info("client=%v", v.client)
	log.Info("operation=%v", req.Operation)
	log.Info("object=%v", p)

	switch req.Operation {
	case k8sadmission.Create:
		return translateErrorToResponse(validateMultiClusterConfigmap(v.client, p))
	case k8sadmission.Update:
		return translateErrorToResponse(validateMultiClusterConfigmap(v.client, p))
	default:
		return admission.Allowed("")
	}
}

// validateMultiClusterConfigmap performs validation checks on the resource
func validateMultiClusterConfigmap(c client.Client, mc *MultiClusterConfigMap) error {
	return nil
}
