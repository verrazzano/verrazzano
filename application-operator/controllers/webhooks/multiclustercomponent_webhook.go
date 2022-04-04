// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"net/http"

	"github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	k8sadmission "k8s.io/api/admission/v1"
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

	if mcc.ObjectMeta.DeletionTimestamp.IsZero() {
		switch req.Operation {
		case k8sadmission.Create, k8sadmission.Update:
			err = validateMultiClusterResource(v.client, mcc)
			if err != nil {
				return admission.Denied(err.Error())
			}
			err = validateNamespaceInProject(v.client, mcc.Namespace)
			if err != nil {
				return admission.Denied(err.Error())
			}
		}
	}
	return admission.Allowed("")
}
