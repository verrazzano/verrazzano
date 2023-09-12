// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oke

import (
	"context"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	ocicommon "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/common"
	ocinetwork "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/network"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type (
	Properties struct {
		ocicommon.Values
		vmcv1alpha1.OKEQuickCreateSpec
		WorkerNodeSubnetName string
		AvailabilityDomains  []oci.AvailabilityDomain
	}
)

func NewProperties(ctx context.Context, cli clipkg.Client, loader oci.CredentialsLoader, ociClientGetter func(creds *oci.Credentials) (oci.Client, error), q *vmcv1alpha1.OKEQuickCreate) (*Properties, error) {
	creds, err := loader.GetCredentialsIfAllowed(ctx, cli, q.Spec.IdentityRef.AsNamespacedName(), q.Namespace)
	if err != nil {
		return nil, err
	}
	props := &Properties{
		Values: ocicommon.Values{
			Name:            q.Name,
			Namespace:       q.Namespace,
			Credentials:     creds,
			Network:         q.Spec.Network.Config,
			OCIClientGetter: ociClientGetter,
		},
		OKEQuickCreateSpec: q.Spec,
	}
	ociClient, err := props.OCIClientGetter(creds)
	if err != nil {
		return nil, err
	}
	ads, err := ociClient.GetAvailabilityAndFaultDomains(ctx)
	if err != nil {
		return nil, err
	}
	props.AvailabilityDomains = ads

	if err := props.SetCommonValues(ctx, cli, q, ocinetwork.GVKOCIManagedCluster); err != nil {
		return nil, err
	}
	props.WorkerNodeSubnetName = props.GetSubnetNameForRole(vmcv1alpha1.SubnetRoleWorker)
	return props, nil
}
