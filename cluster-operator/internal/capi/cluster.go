// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OCNEControlPlaneProvider = "OCNEControlPlane"
	OCNEInfrastructureProvider = "OCICluster"
	OKEControlPlaneProvider = "OCIManagedControlPlane"
	OKEInfrastructureProvider = "OCIManagedCluster"
)

var GVKCAPICluster = schema.GroupVersionKind{
	Group:   "cluster.x-k8s.io",
	Version: "v1beta1",
	Kind:    "Cluster",
}

func GetCluster(ctx context.Context, cli clipkg.Client, clusterNamespacedName types.NamespacedName) (*unstructured.Unstructured, error) {
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(GVKCAPICluster)
	if err := cli.Get(context.TODO(), clusterNamespacedName, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}
