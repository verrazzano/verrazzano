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
	clusterName = "capi-test-cluster"
	testNamespace = "test-namespace"
)

func TestGetCluster(t *testing.T) {
	a := assert.New(t)
	ctx := context.TODO()
	fakeClient := fake.NewClientBuilder().WithObjects(newCAPICluster(clusterName, testNamespace)).Build()
	cluster, err := GetCluster(ctx, fakeClient, types.NamespacedName{Name: clusterName, Namespace: testNamespace})

	a.NoError(err)
	a.Equal(clusterName, cluster.GetName())
	a.Equal(testNamespace, cluster.GetNamespace())
	a.Equal(GVKCAPICluster, cluster.GetObjectKind().GroupVersionKind())

	cluster, err = GetCluster(ctx, fakeClient, types.NamespacedName{Name: "nonexistent-cluster", Namespace: testNamespace})
	a.Error(err)
	a.Nil(cluster)
}

func newCAPICluster(name, namespace string) *unstructured.Unstructured {
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(GVKCAPICluster)
	cluster.SetName(name)
	cluster.SetNamespace(namespace)
	return cluster
}
