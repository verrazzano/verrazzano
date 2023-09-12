// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"errors"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci"
	ocinetwork "github.com/verrazzano/verrazzano/cluster-operator/controllers/quickcreate/controller/oci/network"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

type Values struct {
	Name      string
	Namespace string
	*oci.Credentials
	Network         *vmcv1alpha1.Network
	ExistingSubnets []oci.Subnet
	OCIClientGetter func(creds *oci.Credentials) (oci.Client, error)
}

func (o *Values) SetCommonValues(ctx context.Context, cli clipkg.Client, q clipkg.Object, gvk schema.GroupVersionKind) error {
	if !o.HasOCINetwork() {
		network, err := ocinetwork.GetNetwork(ctx, cli, q, gvk)
		if err == nil {
			o.Network = network
		}

	}
	if !o.IsQuickCreate() {
		if err := o.SetExistingSubnets(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (o *Values) HasOCINetwork() bool {
	return o.Network != nil && len(o.Network.Subnets) > 0
}

func (o *Values) IsQuickCreate() bool {
	if o.Network == nil {
		return true
	}
	return o.Network.CreateVCN
}

func (o *Values) SetExistingSubnets(ctx context.Context) error {
	if o.Credentials == nil {
		return errors.New("no credentials")
	}
	ociClient, err := o.OCIClientGetter(o.Credentials)
	if err != nil {
		return err
	}
	var subnetList []oci.Subnet
	for _, sn := range o.Network.Subnets {
		subnet, err := ociClient.GetSubnetByID(ctx, sn.ID, string(sn.Role))
		if err != nil {
			return err
		}
		subnetList = append(subnetList, *subnet)
	}
	o.ExistingSubnets = subnetList
	return nil
}

func (o *Values) GetSubnetNameForRole(role vmcv1alpha1.SubnetRole) string {
	for _, subnet := range o.ExistingSubnets {
		if subnet.Role == string(role) {
			return subnet.Name
		}
	}
	return ""
}
