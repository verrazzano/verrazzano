// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"net/http"

	k8sadmission "k8s.io/api/admission/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// MultiClusterComponentValidator is a struct holding objects used during validation.
type MultiClusterComponentValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

// InjectClient injects the client.
func (v *MultiClusterComponentValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

// InjectDecoder injects the decoder.
func (v *MultiClusterComponentValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}

// Handle performs validation of created or updated MultiClusterComponent resources.
func (v *MultiClusterComponentValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	mcc := &v1alpha1.MultiClusterComponent{}
	err := v.decoder.Decode(req, mcc)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case k8sadmission.Create, k8sadmission.Update:
		return translateErrorToResponse(validateMultiClusterComponent(v.client, mcc))
	default:
		return admission.Allowed("")
	}
}

// validateMultiClusterComponent performs validation checks on the resource
func validateMultiClusterComponent(c client.Client, mcc *v1alpha1.MultiClusterComponent) error {
	if len(mcc.Spec.Placement.Clusters) == 0 {
		return fmt.Errorf("One or more target clusters must be provided")
	}
	if !isLocalClusterManagedCluster(c) {
		if err := validateTargetClustersExist(c, mcc.Spec.Placement); err != nil {
			return err
		}
	}
	return nil
}
