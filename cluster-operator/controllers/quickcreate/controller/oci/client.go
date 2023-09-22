// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oci

import (
	"context"
	"github.com/oracle/oci-go-sdk/v53/common"
	"github.com/oracle/oci-go-sdk/v53/core"
	"github.com/oracle/oci-go-sdk/v53/identity"
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
		GetAvailabilityAndFaultDomains(ctx context.Context) ([]AvailabilityDomain, error)
	}
	// ClientImpl OCI Client implementation
	ClientImpl struct {
		tenancyID      string
		vnClient       core.VirtualNetworkClient
		identityClient identity.IdentityClient
	}
	Subnet struct {
		ID          string
		Role        string
		Name        string
		DisplayName string
		CIDR        string
		Type        string
	}
	AvailabilityDomain struct {
		Name         string
		FaultDomains []FaultDomain
	}
	FaultDomain struct {
		Name string
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
	i, err := identity.NewIdentityClientWithConfigurationProvider(provider)
	if err != nil {
		return nil, err
	}
	return &ClientImpl{
		tenancyID:      creds.Tenancy,
		vnClient:       net,
		identityClient: i,
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
		ID:          subnetID,
		CIDR:        *sn.CidrBlock,
		Type:        subnetAccess(sn),
		Name:        role,
		DisplayName: *sn.DisplayName,
		Role:        role,
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

func (c *ClientImpl) GetAvailabilityAndFaultDomains(ctx context.Context) ([]AvailabilityDomain, error) {
	ads, err := c.identityClient.ListAvailabilityDomains(ctx, identity.ListAvailabilityDomainsRequest{
		CompartmentId: &c.tenancyID,
	})
	if err != nil {
		return nil, err
	}
	var availabilityDomains []AvailabilityDomain
	for _, ad := range ads.Items {
		availabilityDomain := AvailabilityDomain{
			Name: *ad.Name,
		}
		fd, err := c.identityClient.ListFaultDomains(ctx, identity.ListFaultDomainsRequest{
			CompartmentId:      &c.tenancyID,
			AvailabilityDomain: ad.Name,
		})
		if err != nil {
			return nil, err
		}
		for _, f := range fd.Items {
			availabilityDomain.FaultDomains = append(availabilityDomain.FaultDomains, FaultDomain{
				Name: *f.Name,
			})
		}
		availabilityDomains = append(availabilityDomains, availabilityDomain)
	}
	return availabilityDomains, nil
}

// subnetAccess returns public or private, depending on a subnet's access type
func subnetAccess(subnet core.Subnet) string {
	if subnet.ProhibitPublicIpOnVnic != nil && subnet.ProhibitInternetIngress != nil && !*subnet.ProhibitPublicIpOnVnic && !*subnet.ProhibitInternetIngress {
		return subnetPublic
	}
	return subnetPrivate
}
