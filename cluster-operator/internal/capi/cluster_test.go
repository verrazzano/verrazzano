// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	clusterName      = "test-cluster"
	clusterClassName = "test-cluster-class"
	testNamespace    = "test-namespace"
)
func TestGetClusterClassFromCluster(t *testing.T) {
	a := assert.New(t)
	ctx := context.TODO()
	scheme := runtime.NewScheme()
	_ = v1beta1.AddToScheme(scheme)
	cluster := newCAPIClusterWithClassReference(clusterName, clusterClassName, testNamespace)
	clusterClass := newCAPIClusterClass(clusterClassName, testNamespace)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster, clusterClass).Build()

	retrievedCluster := &v1beta1.Cluster{}
	err := fakeClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: testNamespace}, retrievedCluster)
	a.NoError(err)

	// Get the ClusterClass associated with the Cluster
	retrievedClusterClass, err := GetClusterClassFromCluster(ctx, fakeClient, retrievedCluster)
	a.NoError(err)
	a.Equal(clusterClass, retrievedClusterClass)

	// Shouldn't work if the CAPI Cluster has no reference to a ClusterClass
	fakeClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(newCAPICluster(clusterName, testNamespace)).Build()
	err = fakeClient.Get(ctx, types.NamespacedName{Name: clusterName, Namespace: testNamespace}, retrievedCluster)
	a.NoError(err)
	retrievedClusterClass, err = GetClusterClassFromCluster(ctx, fakeClient, retrievedCluster)
	a.Error(err)
	a.Nil(retrievedClusterClass)
}

func newCAPICluster(name, namespace string) *v1beta1.Cluster {
	cluster := v1beta1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind: "Cluster",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return &cluster
}

func newCAPIClusterWithClassReference(name, className, namespace string) *v1beta1.Cluster {
	cluster := v1beta1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind: "Cluster",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.ClusterSpec{
			Topology: &v1beta1.Topology{
				Class: className,
			},
		},
	}
	return &cluster
}

func newCAPIClusterClass(name, namespace string) *v1beta1.ClusterClass {
	clusterClass := v1beta1.ClusterClass{
		TypeMeta: metav1.TypeMeta{
			Kind: "ClusterClass",
			APIVersion: "cluster.x-k8s.io/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	return &clusterClass
}
