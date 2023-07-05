// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"github.com/oracle/oci-go-sdk/v53/common"
	"github.com/oracle/oci-go-sdk/v53/core"
	"go.uber.org/zap"
	"os"
	"strings"
)

const (
	subnetPrivate = "private"
	subnetPublic  = "public"
)

// Client interface for OCI Clients
type OCIClient interface {
	GetSubnetByID(ctx context.Context, subnetID string, log *zap.SugaredLogger) (*core.Subnet, error)
	GetImageIDByName(ctx context.Context, compartmentID, displayName, operatingSystem, operatingSystemVersion string, log *zap.SugaredLogger) (string, error)
	GetVcnIDByName(ctx context.Context, compartmentID, displayName string, log *zap.SugaredLogger) (string, error)
	GetSubnetIDByName(ctx context.Context, compartmentID, vcnID, displayName string, log *zap.SugaredLogger) (string, error)
	GetSubnetCIDRByName(ctx context.Context, compartmentID, vcnID, displayName string, log *zap.SugaredLogger) (string, error)
	GetNsgIDByName(ctx context.Context, compartmentID, vcnID, displayName string, log *zap.SugaredLogger) (string, error)
	UpdateNSG(ctx context.Context, nsgID string, rule *SecurityRuleDetails, log *zap.SugaredLogger) error
}

// ClientImpl OCI Client implementation
type ClientImpl struct {
	vnClient      core.VirtualNetworkClient
	computeClient core.ComputeClient
}

type SecurityRuleDetails struct {
	Protocol    string
	Description string
	Source      string
	IsStateless bool
	TCPPortMax  int
	TCPPortMin  int
}

// NewClient creates a new OCI Client
func NewClient(provider common.ConfigurationProvider) (OCIClient, error) {
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

// GetImageIDByName retrieves an image OCID given an image name and a compartment id, if that image exists.
func (c *ClientImpl) GetImageIDByName(ctx context.Context, compartmentID, displayName, operatingSystem, operatingSystemVersion string, log *zap.SugaredLogger) (string, error) {
	images, err := c.computeClient.ListImages(ctx, core.ListImagesRequest{
		CompartmentId:          &compartmentID,
		OperatingSystem:        &operatingSystem,
		OperatingSystemVersion: &operatingSystemVersion,
	})
	if err != nil {
		return "", err
	}
	if len(images.Items) == 0 {
		log.Errorf("no images found for %s/%s", compartmentID, displayName)
		return "", err
	}

	for _, image := range images.Items {
		if strings.Contains(*image.DisplayName, displayName) {
			return *image.Id, nil
		}
	}
	// default return
	return *images.Items[0].Id, nil
}

// GetVcnIDByName retrieves an VCN OCID given a vcn name and a compartment id, if the vcn exists.
func (c *ClientImpl) GetVcnIDByName(ctx context.Context, compartmentID, displayName string, log *zap.SugaredLogger) (string, error) {
	vcns, err := c.vnClient.ListVcns(ctx, core.ListVcnsRequest{
		CompartmentId: &compartmentID,
		DisplayName:   &displayName,
	})
	if err != nil {
		return "", err
	}

	if len(vcns.Items) == 0 {
		log.Errorf("no vcns found for %s/%s", compartmentID, displayName)
		return "", err
	}
	return *vcns.Items[0].Id, nil
}

// GetSubnetIDByName retrieves an Subnet OCID given a subnet name and a compartment id, if the subnet exists.
func (c *ClientImpl) GetSubnetIDByName(ctx context.Context, compartmentID, vcnID, displayName string, log *zap.SugaredLogger) (string, error) {
	subnets, err := c.vnClient.ListSubnets(ctx, core.ListSubnetsRequest{
		CompartmentId: &compartmentID,
		VcnId:         &vcnID,
		DisplayName:   &displayName,
	})
	if err != nil {
		return "", err
	}

	if len(subnets.Items) == 0 {
		log.Errorf("no subnet found for %s/%s", compartmentID, displayName)
		return "", err
	}
	return *subnets.Items[0].Id, nil
}

// GetSubnetCIDRByName retrieves an Subnet CIDR block given a subnet name and a compartment id, if the subnet exists.
func (c *ClientImpl) GetSubnetCIDRByName(ctx context.Context, compartmentID, vcnID, displayName string, log *zap.SugaredLogger) (string, error) {
	subnets, err := c.vnClient.ListSubnets(ctx, core.ListSubnetsRequest{
		CompartmentId: &compartmentID,
		VcnId:         &vcnID,
		DisplayName:   &displayName,
	})
	if err != nil {
		return "", err
	}

	if len(subnets.Items) == 0 {
		log.Errorf("no subnet found for %s/%s", compartmentID, displayName)
		return "", err
	}
	return *subnets.Items[0].CidrBlock, nil
}

// GetNsgIDByName retrieves an NSG OCID given a nsg name and a compartment id, if the nsg exists.
func (c *ClientImpl) GetNsgIDByName(ctx context.Context, compartmentID, vcnID, displayName string, log *zap.SugaredLogger) (string, error) {
	nsgs, err := c.vnClient.ListNetworkSecurityGroups(ctx, core.ListNetworkSecurityGroupsRequest{
		CompartmentId: &compartmentID,
		VcnId:         &vcnID,
		DisplayName:   &displayName,
	})
	if err != nil {
		return "", err
	}

	if len(nsgs.Items) == 0 {
		log.Errorf("no nsg found for %s/%s", compartmentID, displayName)
		return "", err
	}

	return *nsgs.Items[0].Id, nil
}

// UpdateNSG retrieves an NSG OCID given a nsg name and a compartment id, if the nsg exists.
func (c *ClientImpl) UpdateNSG(ctx context.Context, nsgID string, rule *SecurityRuleDetails, log *zap.SugaredLogger) error {
	var err error

	ociCoreSecurityDetails := core.AddSecurityRuleDetails{
		Direction:   core.AddSecurityRuleDetailsDirectionIngress,
		Protocol:    &rule.Protocol,
		Description: &rule.Description,
		Source:      &rule.Source,
		SourceType:  core.AddSecurityRuleDetailsSourceTypeCidrBlock,
		IsStateless: &rule.IsStateless,
	}

	switch rule.Protocol {
	case "6":
		ociCoreSecurityDetails.TcpOptions = &core.TcpOptions{
			DestinationPortRange: &core.PortRange{
				Max: &rule.TCPPortMax,
				Min: &rule.TCPPortMin,
			},
		}
	}

	_, err = c.vnClient.AddNetworkSecurityGroupSecurityRules(ctx, core.AddNetworkSecurityGroupSecurityRulesRequest{
		NetworkSecurityGroupId: &nsgID,
		AddNetworkSecurityGroupSecurityRulesDetails: core.AddNetworkSecurityGroupSecurityRulesDetails{
			SecurityRules: []core.AddSecurityRuleDetails{
				ociCoreSecurityDetails,
			},
		},
	})

	if err != nil {
		log.Error("unable to update nsg '%s':  %v", nsgID, zap.Error(err))
		return err
	}

	return nil
}

// GetSubnetByID retrieves a subnet given that subnet's Id.
func (c *ClientImpl) GetSubnetByID(ctx context.Context, subnetID string, log *zap.SugaredLogger) (*core.Subnet, error) {
	response, err := c.vnClient.GetSubnet(ctx, core.GetSubnetRequest{
		SubnetId:        &subnetID,
		RequestMetadata: common.RequestMetadata{},
	})
	if err != nil {
		return nil, err
	}

	subnet := response.Subnet
	return &subnet, nil
}

// SubnetAccess returns public or private, depending on a subnet's access type
func SubnetAccess(subnet core.Subnet, log *zap.SugaredLogger) string {
	if subnet.ProhibitPublicIpOnVnic != nil && subnet.ProhibitInternetIngress != nil && !*subnet.ProhibitPublicIpOnVnic && !*subnet.ProhibitInternetIngress {
		return subnetPublic
	}
	return subnetPrivate
}

func GetOCIConfigurationProvider(log *zap.SugaredLogger) common.ConfigurationProvider {
	_, err := os.Stat(OCIPrivateKeyPath)
	if err != nil {
		log.Errorf("file '%s' not found", OCIPrivateKeyPath)
		return nil
	}
	data, err := os.ReadFile(OCIPrivateKeyPath)
	if err != nil {
		log.Error("failed reading file contents: ", zap.Error(err))
		return nil
	}
	return common.NewRawConfigurationProvider(OCITenancyID, OCIUserID, OCIRegion, OCIFingerprint, strings.TrimSpace(string(data)), nil)
}
