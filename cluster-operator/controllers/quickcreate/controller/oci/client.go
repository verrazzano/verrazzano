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
		GetSubnetById(ctx context.Context, id, role string) (*Subnet, error)
	}
	// ClientImpl OCI Client implementation
	ClientImpl struct {
		vnClient core.VirtualNetworkClient
	}
	Subnet struct {
		Id   string
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
	return &ClientImpl{
		vnClient: net,
	}, nil
}

// GetSubnetById retrieves a subnet given that subnet's Id.
func (c *ClientImpl) GetSubnetById(ctx context.Context, subnetId, role string) (*Subnet, error) {
	response, err := c.vnClient.GetSubnet(ctx, core.GetSubnetRequest{
		SubnetId:        &subnetId,
		RequestMetadata: common.RequestMetadata{},
	})
	if err != nil {
		return nil, err
	}

	sn := response.Subnet
	return &Subnet{
		Id:   subnetId,
		CIDR: *sn.CidrBlock,
		Type: subnetAccess(sn),
		Name: role,
		Role: role,
	}, nil
}

// subnetAccess returns public or private, depending on a subnet's access type
func subnetAccess(subnet core.Subnet) string {
	if subnet.ProhibitPublicIpOnVnic != nil && subnet.ProhibitInternetIngress != nil && !*subnet.ProhibitPublicIpOnVnic && !*subnet.ProhibitInternetIngress {
		return subnetPublic
	}
	return subnetPrivate
}
