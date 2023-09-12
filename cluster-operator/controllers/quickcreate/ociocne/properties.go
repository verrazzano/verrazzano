// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ociocne

import (
	"context"
	"fmt"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	ocicommon "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/common"
	ocinetwork "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/network"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/ocne"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	//Properties contains all the properties for rendering OCI OCNE Cluster templates.
	Properties struct {
		ocicommon.Values
		*ocne.VersionDefaults
		vmcv1alpha1.OCIOCNEClusterSpec
		LoadBalancerSubnet string
		ProviderID         string
		DockerConfigJSON   string
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
	creds, err := loader.GetCredentialsIfAllowed(ctx, cli, q.Spec.IdentityRef.AsNamespacedName(), q.Namespace)
	if err != nil {
		return nil, err
	}
	props := &Properties{
		Values: ocicommon.Values{
			Name:            q.Name,
			Namespace:       q.Namespace,
			Credentials:     creds,
			Network:         q.Spec.OCI.Network,
			OCIClientGetter: ociClientGetter,
		},
		VersionDefaults:    versions,
		OCIOCNEClusterSpec: q.Spec,
		ProviderID:         oci.ProviderID,
	}
	if err := props.SetCommonValues(ctx, cli, q, ocinetwork.GVKOCICluster); err != nil {
		return nil, err
	}
	if props.HasImagePullSecret() {
		if err := props.SetDockerConfigJSON(ctx, cli); err != nil {
			return nil, err
		}
	}
	// Set LoadBalancerSubnet for simple lookup. Will be empty string if the network has not created yet.
	props.LoadBalancerSubnet = ocinetwork.GetLoadBalancerSubnet(props.Network)
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

func (p *Properties) IsControlPlaneOnly() bool {
	return len(p.OCI.Workers) < 1
}

func (p *Properties) HasImagePullSecret() bool {
	return p.PrivateRegistry != nil && len(p.PrivateRegistry.CredentialsSecret.Name) > 0
}

func (p *Properties) SetDockerConfigJSON(ctx context.Context, cli clipkg.Client) error {
	secret := &corev1.Secret{}
	if err := cli.Get(ctx, types.NamespacedName{
		Namespace: p.PrivateRegistry.CredentialsSecret.Namespace,
		Name:      p.PrivateRegistry.CredentialsSecret.Name,
	}, secret); err != nil {
		return err
	}
	if secret.Data == nil {
		return fmt.Errorf("failed to load private registry credentials from secret %s/%s", p.PrivateRegistry.CredentialsSecret.Namespace, p.PrivateRegistry.CredentialsSecret.Name)
	}
	dockerConfigJSON, ok := secret.Data[".dockerconfigjson"]
	if !ok {
		return fmt.Errorf("no private registry credentials found in secret %s/%s", p.PrivateRegistry.CredentialsSecret.Namespace, p.PrivateRegistry.CredentialsSecret.Name)
	}
	p.DockerConfigJSON = string(dockerConfigJSON)
	return nil
}
