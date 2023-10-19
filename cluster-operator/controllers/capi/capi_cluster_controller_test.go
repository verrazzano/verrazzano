// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package capi

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/cluster-operator/internal/capi"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	clusterName = "capi1"
)

// GIVEN a CAPI cluster resource is created
// WHEN  the reconciler runs
// THEN  a VMC is created
func TestClusterCreation(t *testing.T) {
	asserts := assert.New(t)

	ep := &v1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubernetes",
			Namespace: "default",
		},
		Subsets: []v1.EndpointSubset{{Addresses: []v1.EndpointAddress{{IP: "1.2.3.4"}}}},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(newCAPICluster(clusterName), ep).Build()
	reconciler := newCAPIClusterReconciler(fakeClient)
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	vmc := &v1alpha1.VerrazzanoManagedCluster{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: clusterName, Namespace: constants.VerrazzanoMultiClusterNamespace}, vmc)
	asserts.NoError(err)
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(capi.GVKCAPICluster)
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: clusterName}, cluster)
	asserts.NoError(err)
	asserts.Equal(finalizerName, cluster.GetFinalizers()[0])
}

// GIVEN a CAPI cluster resource is deleted
// WHEN  the reconciler runs
// THEN  the VMC is removed
func TestClusterDeletion(t *testing.T) {
	asserts := assert.New(t)

	cluster := newCAPICluster(clusterName)
	now := metav1.Now()
	cluster.SetDeletionTimestamp(&now)
	cluster.SetFinalizers([]string{finalizerName})

	vmc := &v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: constants.VerrazzanoMultiClusterNamespace,
		},
	}
	vmc.Spec = v1alpha1.VerrazzanoManagedClusterSpec{
		Description: fmt.Sprintf("%s VerrazzanoManagedCluster Resource", cluster.GetName()),
	}

	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(cluster, vmc).Build()
	reconciler := newCAPIClusterReconciler(fakeClient)
	request := newRequest(clusterName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	asserts.NoError(err)

	remainingVmc := &v1alpha1.VerrazzanoManagedCluster{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: clusterName}, remainingVmc)
	asserts.Error(err)
	deletedCluster := &unstructured.Unstructured{}
	deletedCluster.SetGroupVersionKind(capi.GVKCAPICluster)
	asserts.Error(fakeClient.Get(context.TODO(), types.NamespacedName{Name: clusterName}, deletedCluster))
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clientgoscheme.AddToScheme(scheme)
	v1alpha1.AddToScheme(scheme)

	return scheme
}

func newCAPICluster(name string) *unstructured.Unstructured {
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(capi.GVKCAPICluster)
	cluster.SetName(name)
	return cluster
}

func newCAPIClusterReconciler(c client.Client) CAPIClusterReconciler {
	return CAPIClusterReconciler{
		Client:         c,
		Scheme:         newScheme(),
		Log:            zap.S(),
		RancherEnabled: true,
	}
}

func newRequest(name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name: name,
		},
	}
}
