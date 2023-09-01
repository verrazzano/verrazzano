// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"github.com/oracle/oci-go-sdk/v53/common"
	"github.com/oracle/oci-go-sdk/v53/core"
)

const (
	subnetPrivate = "private"
	subnetPublic  = "public"
)

// Client interface for OCI Clients
type (
	Client interface {
		GetSubnetByID(ctx context.Context, id, role string) (*Subnet, error)
		GetVCNByID(ctx context.Context, id string) (*core.Vcn, error)
		GetImageByID(ctx context.Context, id string) (*core.Image, error)
	}
	// ClientImpl OCI Client implementation
	ClientImpl struct {
		vnClient      core.VirtualNetworkClient
		computeClient core.ComputeClient
	}
	Subnet struct {
		ID   string
		Role string
		Name string
		CIDR string
		Type string
	}
)

// NewClient creates a new OCI Client
func NewClient(creds *Credentials) (Client, error) {
	provider, err := creds.AsConfigurationProvider()
	if err != nil {
		return nil, err
	}
	net, err := core.NewVirtualNetworkClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	compute, err := core.NewComputeClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	return &ClientImpl{
		vnClient:      net,
		computeClient: compute,
	}, nil
}

// GetSubnetByID retrieves a subnet given that subnet's ID.
func (c *ClientImpl) GetSubnetByID(ctx context.Context, subnetID, role string) (*Subnet, error) {
	response, err := c.vnClient.GetSubnet(ctx, core.GetSubnetRequest{
		SubnetId:        &subnetID,
		RequestMetadata: common.RequestMetadata{},
	})
	if err != nil {
		return nil, err
	}

	sn := response.Subnet
	return &Subnet{
		ID:   subnetID,
		CIDR: *sn.CidrBlock,
		Type: subnetAccess(sn),
		Name: role,
		Role: role,
	}, nil
}

func (c *ClientImpl) GetVCNByID(ctx context.Context, id string) (*core.Vcn, error) {
	response, err := c.vnClient.GetVcn(ctx, core.GetVcnRequest{
		VcnId: &id,
	})
	if err != nil {
		return nil, err
	}
	return &response.Vcn, nil
}

func (c *ClientImpl) GetImageByID(ctx context.Context, id string) (*core.Image, error) {
	response, err := c.computeClient.GetImage(ctx, core.GetImageRequest{
		ImageId: &id,
	})
	if err != nil {
		return nil, err
	}
	return &response.Image, nil
}

// subnetAccess returns public or private, depending on a subnet's access type
func subnetAccess(subnet core.Subnet) string {
	if subnet.ProhibitPublicIpOnVnic != nil && subnet.ProhibitInternetIngress != nil && !*subnet.ProhibitPublicIpOnVnic && !*subnet.ProhibitInternetIngress {
		return subnetPublic
	}
	return subnetPrivate
}
