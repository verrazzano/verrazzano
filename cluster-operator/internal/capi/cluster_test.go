// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	clusterName      = "test-cluster"
	clusterClassName = "test-cluster-class"
	testNamespace    = "test-namespace"
)

func TestGetCluster(t *testing.T) {
	a := assert.New(t)
	ctx := context.TODO()
	fakeClient := fake.NewClientBuilder().WithObjects(newCAPICluster(clusterName, testNamespace)).Build()

	// get the CAPI cluster
	cluster, err := GetCluster(ctx, fakeClient, types.NamespacedName{Name: clusterName, Namespace: testNamespace})
	a.NoError(err)
	a.Equal(clusterName, cluster.GetName())
	a.Equal(testNamespace, cluster.GetNamespace())
	a.Equal(GVKCAPICluster, cluster.GetObjectKind().GroupVersionKind())

	// attempt to get a CAPI cluster that does not exist
	cluster, err = GetCluster(ctx, fakeClient, types.NamespacedName{Name: "nonexistent-cluster", Namespace: testNamespace})
	a.Error(err)
	a.Nil(cluster)
}

func TestGetClusterClass(t *testing.T) {
	a := assert.New(t)
	ctx := context.TODO()
	fakeClient := fake.NewClientBuilder().WithObjects(newCAPIClusterClass(clusterClassName, testNamespace)).Build()

	// get the CAPI ClusterClass
	clusterClass, err := GetClusterClass(ctx, fakeClient, types.NamespacedName{Name: clusterClassName, Namespace: testNamespace})
	a.NoError(err)
	a.Equal(clusterClassName, clusterClass.GetName())
	a.Equal(testNamespace, clusterClass.GetNamespace())
	a.Equal(GVKCAPIClusterClass, clusterClass.GetObjectKind().GroupVersionKind())

	// attempt to get a CAPI ClusterClass that does not exist
	clusterClass, err = GetClusterClass(ctx, fakeClient, types.NamespacedName{Name: "nonexistent-cluster-class", Namespace: testNamespace})
	a.Error(err)
	a.Nil(clusterClass)
}

func TestGetClusterClassFromCluster(t *testing.T) {
	a := assert.New(t)
	ctx := context.TODO()
	fakeClient := fake.NewClientBuilder().
		WithObjects(newCAPIClusterWithClassReference(clusterName, clusterClassName, testNamespace), newCAPIClusterClass(clusterClassName, testNamespace)).
		Build()

	cluster, err := GetCluster(ctx, fakeClient, types.NamespacedName{Name: clusterName, Namespace: testNamespace})
	a.NoError(err)

	// Get the ClusterClass associated with the Cluster
	clusterClass, err := GetClusterClassFromCluster(ctx, fakeClient, cluster)
	a.NoError(err)
	a.Equal(clusterClassName, clusterClass.GetName())
	a.Equal(testNamespace, clusterClass.GetNamespace())
	a.Equal(GVKCAPIClusterClass, clusterClass.GetObjectKind().GroupVersionKind())

	// Shouldn't work if the CAPI Cluster has no reference to a ClusterClass
	fakeClient = fake.NewClientBuilder().
		WithObjects(newCAPICluster(clusterName, testNamespace), newCAPIClusterClass(clusterClassName, testNamespace)).
		Build()
	cluster, err = GetCluster(ctx, fakeClient, types.NamespacedName{Name: clusterName, Namespace: testNamespace})
	a.NoError(err)
	clusterClass, err = GetClusterClassFromCluster(ctx, fakeClient, cluster)
	a.Error(err)
	a.Nil(clusterClass)
}

func newCAPICluster(name, namespace string) *unstructured.Unstructured {
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(GVKCAPICluster)
	cluster.SetName(name)
	cluster.SetNamespace(namespace)
	return cluster
}

func newCAPIClusterWithClassReference(name, className, namespace string) *unstructured.Unstructured {
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(GVKCAPICluster)
	cluster.SetName(name)
	cluster.SetNamespace(namespace)
	unstructured.SetNestedField(cluster.Object, className, "spec", "topology", "class")
	return cluster
}

func newCAPIClusterClass(name, namespace string) *unstructured.Unstructured {
	clusterClass := &unstructured.Unstructured{}
	clusterClass.SetGroupVersionKind(GVKCAPIClusterClass)
	clusterClass.SetName(name)
	clusterClass.SetNamespace(namespace)
	return clusterClass
}
