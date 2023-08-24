// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	Properties struct {
		*vmcv1alpha1.OCNEOCIQuickCreate `json:",inline"`
		oci.OCICredentials              `json:",inline"`
	}
)

func NewProperties(ctx context.Context, cli clipkg.Client, q *vmcv1alpha1.OCNEOCIQuickCreate) (*Properties, error) {
	props := &Properties{}

	return props, nil
}
