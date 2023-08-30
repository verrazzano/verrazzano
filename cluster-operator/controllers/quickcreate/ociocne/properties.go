// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	"errors"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/ocne"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	//Properties contains all the properties for rendering OCI OCNE Cluster templates.
	Properties struct {
		*ocne.VersionDefaults `json:",inline"`
		*oci.Credentials      `json:",inline"` //nolint:gosec //#gosec G101
		*vmcv1alpha1.Network
		Name                           string
		Namespace                      string
		vmcv1alpha1.OCIOCNEClusterSpec `json:",inline"`
		LoadBalancerSubnet             string
		ProviderID                     string
		ExistingSubnets                []oci.Subnet
		OCIClientGetter                func(creds *oci.Credentials) (oci.Client, error)
	}
)

// NewProperties creates a new properties object based on the quick create resource.
func NewProperties(ctx context.Context, cli clipkg.Client, loader oci.CredentialsLoader, ociClientGetter func(creds *oci.Credentials) (oci.Client, error), q *vmcv1alpha1.OCNEOCIQuickCreate) (*Properties, error) {
	// Get the OCNE Versions
	versions, err := ocne.GetVersionDefaults(ctx, cli, q.Spec.OCNE.Version)
	if err != nil {
		return nil, err
	}
	// Try to load the credentials, if allowed
	creds, err := loader.GetCredentialsIfAllowed(ctx, cli, q.Spec.IdentityRef, q.Namespace)
	if err != nil {
		return nil, err
	}
	props := &Properties{
		VersionDefaults:    versions,
		Credentials:        creds,
		Name:               q.Name,
		Namespace:          q.Namespace,
		OCIOCNEClusterSpec: q.Spec,
		Network:            q.Spec.OCI.Network,
		ProviderID:         oci.ProviderID,
		OCIClientGetter:    ociClientGetter,
	}
	// If there's no OCI network, check if the network has created
	if !props.HasOCINetwork() {
		network, _ := oci.GetNetwork(ctx, cli, q)
		if err == nil {
			props.Network = network
		}
	}
	if !props.IsQuickCreate() {
		if err := props.SetExistingSubnets(ctx); err != nil {
			return nil, err
		}
	}
	// Set LoadBalancerSubnet for simple lookup. Will be empty string if the network has not created yet.
	props.LoadBalancerSubnet = oci.GetLoadBalancerSubnet(props.Network)
	return props, nil
}

func (p *Properties) ApplyTemplate(cli clipkg.Client, templates ...[]byte) error {
	applier := k8sutil.NewYAMLApplier(cli, "")
	for _, tmpl := range templates {
		if err := applier.ApplyBT(tmpl, p); err != nil {
			return err
		}
	}
	return nil
}

// HasOCINetwork returns true if the OCI Network is present
func (p *Properties) HasOCINetwork() bool {
	return p.Network != nil
}

func (p *Properties) IsQuickCreate() bool {
	return p.Network.CreateVCN
}

func (p *Properties) SetExistingSubnets(ctx context.Context) error {
	if p.Credentials == nil {
		return errors.New("no credentials")
	}
	ociClient, err := p.OCIClientGetter(p.Credentials)
	if err != nil {
		return err
	}
	var subnetList []oci.Subnet
	for _, sn := range p.Network.Subnets {
		subnet, err := ociClient.GetSubnetByID(ctx, sn.ID, string(sn.Role))
		if err != nil {
			return err
		}
		subnetList = append(subnetList, *subnet)
	}
	p.ExistingSubnets = subnetList
	return nil
}
