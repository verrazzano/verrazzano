// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// newScheme creates a new scheme that includes this package's object to use for testing
func scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	return scheme
}

// TestCreateArgoCDResources tests the synchronization method for the following use case.
// GIVEN a request to create the k8s resources
//
//	containing argocd k8s resources
//
// WHEN the new object exists
// THEN ensure that the k8s resources (SA, secret, cluster role, role bindings are created)
func TestCreateArgoCDResources(t *testing.T) {
	log := zap.S().With("test")

	adminClient := fake.NewClientBuilder().WithScheme(scheme()).WithObjects().Build()
	localClient := fake.NewClientBuilder().WithScheme(scheme()).Build()
	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	// Verify the associated k8s resources got created on local cluster
	assert.NoError(t, s.createArgocdResources([]byte("foobar")))
	err := s.LocalClient.Get(s.Context, types.NamespacedName{Name: serviceAccountName, Namespace: kubeSystemNamespace}, &corev1.ServiceAccount{})
	assert.NoError(t, err)

	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: secName, Namespace: kubeSystemNamespace}, &corev1.Secret{})
	assert.NoError(t, err)

	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: clusterRoleName}, &rbacv1.ClusterRole{})
	assert.NoError(t, err)

	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: clusterRoleBindingName}, &rbacv1.ClusterRoleBinding{})
	assert.NoError(t, err)
}
