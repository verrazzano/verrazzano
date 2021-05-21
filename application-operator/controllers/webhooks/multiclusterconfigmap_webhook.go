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
	mccm := &v1alpha1.MultiClusterConfigMap{}
	err := v.decoder.Decode(req, mccm)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	switch req.Operation {
	case k8sadmission.Create, k8sadmission.Update:
		return translateErrorToResponse(validateMultiClusterConfigmap(v.client, mccm))
	default:
		return admission.Allowed("")
	}
}

// validateMultiClusterConfigmap performs validation checks on the resource
func validateMultiClusterConfigmap(c client.Client, mccm *v1alpha1.MultiClusterConfigMap) error {
	if len(mccm.Spec.Placement.Clusters) == 0 {
		return fmt.Errorf("One or more target clusters must be provided")
	}
	if !isLocalClusterManagedCluster(c) {
		if err := validateTargetClustersExist(c, mccm.Spec.Placement); err != nil {
			return err
		}
	}
	return nil
}
