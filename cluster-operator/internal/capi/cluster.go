// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
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

var GVKCAPIClusterClass = schema.GroupVersionKind{
	Group:   "cluster.x-k8s.io",
	Version: "v1beta1",
	Kind:    "ClusterClass",
}

// GetCluster returns the requested CAPI Cluster as an unstructured pointer.
func GetCluster(ctx context.Context, cli clipkg.Client, clusterNamespacedName types.NamespacedName) (*unstructured.Unstructured, error) {
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(GVKCAPICluster)
	if err := cli.Get(context.TODO(), clusterNamespacedName, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// GetClusterClass returns the requested CAPI ClusterClass as an unstructured pointer.
func GetClusterClass(ctx context.Context, cli clipkg.Client, clusterClassNamespacedName types.NamespacedName) (*unstructured.Unstructured, error) {
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(GVKCAPIClusterClass)
	if err := cli.Get(context.TODO(), clusterClassNamespacedName, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// GetClusterClassFromCluster returns the ClusterClass associated with the provided CAPI Cluster.
func GetClusterClassFromCluster(ctx context.Context, cli clipkg.Client, cluster *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	clusterClassName, found, err := unstructured.NestedString(cluster.Object, "spec", "topology", "class")
	if !found {
		return nil, fmt.Errorf("spec.topology.class field not found in Cluster %s/%s with error: %v", cluster.GetNamespace(), cluster.GetName(), err)
	}
	if err != nil {
		return nil, err
	}
	clusterClassNamespacedName := types.NamespacedName{
		Name:      clusterClassName,
		Namespace: cluster.GetNamespace(),
	}
	return GetClusterClass(ctx, cli, clusterClassNamespacedName)
}
