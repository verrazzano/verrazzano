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
	mcs := &MultiClusterSecret{}
	err := v.decoder.Decode(req, mcs)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case k8sadmission.Create, k8sadmission.Update:
		return translateErrorToResponse(validateMultiClusterSecret(v.client, mcs))
	default:
		return admission.Allowed("")
	}
}

// validateMultiClusterSecret performs validation checks on the resource
func validateMultiClusterSecret(c client.Client, mcs *MultiClusterSecret) error {
	if len(mcs.Spec.Placement.Clusters) == 0 {
		return fmt.Errorf("One or more target clusters must be provided")
	}
	if isLocalClusterAdminCluster(c) {
		if err := validateTargetClustersExist(c, mcs.Spec.Placement); err != nil {
			return err
		}
	}
	return nil
}
