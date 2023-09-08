// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fake

import (
	"context"
	"errors"
	"github.com/oracle/oci-go-sdk/v53/core"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	CredentialsLoaderImpl struct {
		Credentials *oci.Credentials
	}
	ClientImpl struct {
		VCN *core.Vcn
	}
)

func (c *CredentialsLoaderImpl) GetCredentialsIfAllowed(_ context.Context, _ clipkg.Client, _ types.NamespacedName, _ string) (*oci.Credentials, error) {
	return c.Credentials, nil
}

func (c *ClientImpl) GetSubnetByID(_ context.Context, id, role string) (*oci.Subnet, error) {
	return &oci.Subnet{
		ID:   id,
		Role: role,
		Name: role,
		CIDR: "10.0.0.0/16",
		Type: "public",
	}, nil
}

func (c *ClientImpl) GetVCNByID(_ context.Context, id string) (*core.Vcn, error) {
	if id == *c.VCN.Id {
		return c.VCN, nil
	}
	return nil, errors.New("vcn not found")
}
