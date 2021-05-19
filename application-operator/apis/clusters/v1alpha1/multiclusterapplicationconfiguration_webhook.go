// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"context"
	"fmt"
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
	mcac := &MultiClusterApplicationConfiguration{}
	err := v.decoder.Decode(req, mcac)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case k8sadmission.Create, k8sadmission.Update:
		return translateErrorToResponse(validateMultiClusterApplicationConfiguration(v.client, mcac))
	default:
		return admission.Allowed("")
	}
}

// validateMultiClusterApplicationConfiguration performs validation checks on the resource
func validateMultiClusterApplicationConfiguration(c client.Client, mcac *MultiClusterApplicationConfiguration) error {
	if len(mcac.Spec.Placement.Clusters) == 0 {
		return fmt.Errorf("One or more target clusters must be provided")
	}
	if isLocalClusterAdminCluster(c) {
		if err := validateTargetClustersExist(c, mcac.Spec.Placement); err != nil {
			return err
		}
	}
	return nil
}
