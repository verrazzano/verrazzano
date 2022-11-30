// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

// newScheme creates a new scheme that includes this package's object to use for testing
func scheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	_ = rbacv1.AddToScheme(scheme)
	return scheme
}

func TestCreateArgoCDResources(t *testing.T) {
	scheme := scheme()

	vmc := &v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: rancherNamespace,
			Name:      "cluster",
		},
		Status: v1alpha1.VerrazzanoManagedClusterStatus{
			RancherRegistration: v1alpha1.RancherRegistration{
				ClusterID: "cluster-id",
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects().Build()
	rc := &VerrazzanoManagedClusterReconciler{
		Client: fakeClient,
		log:    vzlog.DefaultLogger(),
	}
	assert.NoError(t, rc.createArgoCDServiceAccount(vmc, vzlog.DefaultLogger()))
	assert.NoError(t, rc.createArgoCDSecret(vzlog.DefaultLogger(), context.TODO(), []byte("foobar")))
	assert.NoError(t, rc.createArgoCDRole(vzlog.DefaultLogger()))
	assert.NoError(t, rc.createArgoCDRoleBinding(vzlog.DefaultLogger()))
}
