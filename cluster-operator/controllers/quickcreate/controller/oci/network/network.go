// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package network

import (
	"context"
	"errors"
	vmcv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	GVKOCICluster = schema.GroupVersionKind{
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta2",
		Kind:    "ocicluster",
	}
	GVKOCIManagedCluster = schema.GroupVersionKind{
		Group:   "infrastructure.cluster.x-k8s.io",
		Version: "v1beta2",
		Kind:    "ocimanagedcluster",
	}
)

func GetNetwork(ctx context.Context, cli clipkg.Client, o clipkg.Object, gvk schema.GroupVersionKind) (*vmcv1alpha1.Network, error) {
	network := &vmcv1alpha1.Network{}
	ociCluster := &unstructured.Unstructured{}
	ociCluster.SetGroupVersionKind(gvk)
	err := cli.Get(ctx, types.NamespacedName{
		Namespace: o.GetNamespace(),
		Name:      o.GetName(),
	}, ociCluster)
	if err != nil {
		return nil, err
	}
	// Get VCN id and VCN subnets from the OCI Cluster resource
	vcnID, found, err := unstructured.NestedString(ociCluster.Object, "spec", "networkSpec", "vcn", "id")
	if err != nil || !found {
		return nil, errors.New("waiting for VCN to be created")
	}
	network.VCN = vcnID
	subnets, found, err := unstructured.NestedSlice(ociCluster.Object, "spec", "networkSpec", "vcn", "subnets")
	if err != nil || !found {
		return nil, errors.New("waiting for subnets to be created")
	}

	// For each subnet in the VCN subnet list, identify its role and populate the subnet id in the cluster state
	for _, subnet := range subnets {
		subnetObject, ok := subnet.(map[string]interface{})
		if !ok {
			return nil, errors.New("subnet is creating")
		}

		// Get nested subnet Role and id from the subnet object
		subnetRole, found, err := unstructured.NestedString(subnetObject, "role")
		if err != nil || !found {
			return nil, errors.New("waiting for subnet role to be populated")
		}
		subnetID, found, err := unstructured.NestedString(subnetObject, "id")
		if err != nil || !found {
			return nil, errors.New("waiting for subnet id to be populated")
		}

		network.Subnets = append(network.Subnets, vmcv1alpha1.Subnet{
			Role: vmcv1alpha1.SubnetRole(subnetRole),
			ID:   subnetID,
		})
	}
	// No network can be loaded currently
	return network, nil
}

func GetLoadBalancerSubnet(network *vmcv1alpha1.Network) string {
	if network == nil {
		return ""
	}
	for _, subnet := range network.Subnets {
		if subnet.Role == vmcv1alpha1.SubnetRoleServiceLB {
			return subnet.ID
		}
	}
	return ""
}
