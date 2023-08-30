// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
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
		ProviderId                     string
	}
)

// NewProperties creates a new properties object based on the quick create resource.
func NewProperties(ctx context.Context, cli clipkg.Client, loader oci.CredentialsLoader, q *vmcv1alpha1.OCNEOCIQuickCreate) (*Properties, error) {
	// Get the OCNE Versions
	versions, err := ocne.GetVersionDefaults(ctx, cli, q.Spec.OCNE.Version)
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
		ProviderId:         oci.ProviderId,
	}
	// If there's no OCI network, check if the network has created
	if !props.HasOCINetwork() {
		network, _ := oci.GetNetwork(ctx, cli, q)
		if err == nil {
			props.Network = network
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
