// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	//OCNEProperties contains all the properties for rendering OCNE Cluster templates.
	OCNEProperties struct {
		*vmcv1alpha1.OCNEOCIQuickCreate `json:",inline"`
		*oci.Credentials                `json:",inline"` //nolint:gosec //#gosec G101
	}
)

// NewOCNE creates a new properties object based on the quick create resource.
func NewOCNE(ctx context.Context, cli clipkg.Client, loader oci.CredentialsLoader, q *vmcv1alpha1.OCNEOCIQuickCreate) (*OCNEProperties, error) {
	creds, err := loader.LoadCredentialsIfAllowed(ctx, cli, q.Spec.IdentityRef, q.Namespace)
	if err != nil {
		return nil, err
	}

	return &OCNEProperties{
		OCNEOCIQuickCreate: q,
		Credentials:        creds,
	}, nil
}

func (p *OCNEProperties) ApplyFromTemplateDirectory(cli clipkg.Client, dir string) error {
	return k8sutil.NewYAMLApplier(cli, "").ApplyDT(dir, p)
}
