// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OCNEControlPlaneProvider   = "OCNEControlPlane"
	OCNEInfrastructureProvider = "OCICluster"
	OKEControlPlaneProvider    = "OCIManagedControlPlane"
	OKEInfrastructureProvider  = "OCIManagedCluster"
)

var GVKCAPICluster = schema.GroupVersionKind{
	Group:   "cluster.x-k8s.io",
	Version: "v1beta1",
	Kind:    "Cluster",
}

// GetClusterClassFromCluster returns the ClusterClass associated with the provided CAPI Cluster.
func GetClusterClassFromCluster(ctx context.Context, cli clipkg.Client, cluster *v1beta1.Cluster) (*v1beta1.ClusterClass, error) {
	// get the ClusterClass name, avoiding nil pointer exceptions
	if cluster == nil {
		return nil, fmt.Errorf("cannot get ClusterClass from a nil Cluster")
	}
	var clusterClassName string
	if cluster.Spec.Topology != nil {
		clusterClassName = cluster.Spec.Topology.Class
	}
	if clusterClassName == "" {
		return nil, fmt.Errorf("CAPI Cluster %s/%s does not reference a ClusterClass", cluster.Namespace, cluster.Name)
	}

	// Retrieve the ClusterClass
	clusterClassNamespacedName := types.NamespacedName{
		Name:      clusterClassName,
		Namespace: cluster.GetNamespace(),
	}
	clusterClass := &v1beta1.ClusterClass{}
	if err := cli.Get(ctx, clusterClassNamespacedName, clusterClass); err != nil {
		return nil, err
	}
	return clusterClass, nil
}
