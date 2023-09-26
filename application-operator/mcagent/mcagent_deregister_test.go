// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"testing"
	"time"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/cluster-operator/apis/clusters/v1alpha1"
	apierrors "github.com/verrazzano/verrazzano/pkg/k8s/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestSyncDeregistration tests the synchronization method for the following use case.
// GIVEN objects on the managed cluster used for managed cluster synchronization
// WHEN tthe VMC is deleted
// THEN ensure that the managed cluster resources are cleaned up
func TestSyncDeregistration(t *testing.T) {
	a := asserts.New(t)

	vmcDeleted := clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         constants.VerrazzanoMultiClusterNamespace,
			Name:              testClusterName,
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{"testFinalizer"},
		},
	}
	vmcNotDeleted := clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoMultiClusterNamespace,
			Name:      testClusterName,
		},
	}

	mcAgentSec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.MCAgentSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}
	mcRegSec := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      constants.MCRegistrationSecret,
			Namespace: constants.VerrazzanoSystemNamespace,
		},
	}

	tests := []struct {
		name    string
		vmc     *clustersv1alpha1.VerrazzanoManagedCluster
		objects []client.Object
	}{
		{
			name:    "test not deleted",
			vmc:     &vmcNotDeleted,
			objects: []client.Object{},
		},
		{
			name: "test not deleted with secrets",
			vmc:  &vmcNotDeleted,
			objects: []client.Object{
				&mcRegSec,
				&mcAgentSec,
			},
		},
		{
			name:    "test deleted no objects",
			vmc:     &vmcDeleted,
			objects: []client.Object{},
		},
		{
			name:    "test deleted registration secret",
			vmc:     &vmcDeleted,
			objects: []client.Object{&mcRegSec},
		},
		{
			name:    "test deleted agent secret",
			vmc:     &vmcDeleted,
			objects: []client.Object{&mcAgentSec},
		},
		{
			name: "test deleted both secrets",
			vmc:  &vmcDeleted,
			objects: []client.Object{
				&mcRegSec,
				&mcAgentSec,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := k8scheme.Scheme
			err := clustersv1alpha1.AddToScheme(scheme)
			a.NoError(err)
			adminFake := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.vmc).Build()
			managedFake := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()
			s := &Syncer{
				AdminClient:        adminFake,
				LocalClient:        managedFake,
				Log:                zap.S(),
				ManagedClusterName: testClusterName,
				Context:            context.TODO(),
			}
			err = s.syncDeregistration()
			a.NoError(err)

			// Verify that the objects have been deleted
			if !tt.vmc.DeletionTimestamp.IsZero() {
				for _, obj := range tt.objects {
					err := managedFake.Get(context.TODO(), types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
					a.True(apierrors.IsNotFound(err))
				}
				return
			}
			// If not deleted, make sure the objects persist
			for _, obj := range tt.objects {
				err := managedFake.Get(context.TODO(), types.NamespacedName{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
				a.NoError(err)
			}
		})
	}
}

// TestVerifyDeregister tests function to decide if the cluster should be deregistered
// GIVEN objects on the managed cluster used for managed cluster synchronization
// WHEN deregistration is expected
// THEN the function returns true
func TestVerifyDeregister(t *testing.T) {
	assert := asserts.New(t)

	scheme := k8scheme.Scheme
	err := clustersv1alpha1.AddToScheme(scheme)
	assert.NoError(err)

	vmcDeleted := clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         constants.VerrazzanoMultiClusterNamespace,
			Name:              testClusterName,
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
			Finalizers:        []string{"testFinalizer"},
		},
	}
	adminDeleted := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&vmcDeleted).Build()

	vmc := clustersv1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: constants.VerrazzanoMultiClusterNamespace,
			Name:      testClusterName,
		},
	}
	admin := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&vmc).Build()

	tests := []struct {
		name        string
		adminClient client.Client
		expectFunc  func(bool, ...interface{}) bool
	}{
		{
			name:        "test nil Admin client",
			adminClient: nil,
			expectFunc:  assert.True,
		},
		{
			name:        "test non zero timestamp",
			adminClient: adminDeleted,
			expectFunc:  assert.True,
		},
		{
			name:        "test zero deletion timestamp",
			adminClient: admin,
			expectFunc:  assert.False,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := Syncer{
				AdminClient:        tt.adminClient,
				Log:                zap.S(),
				ManagedClusterName: testClusterName,
				Context:            context.TODO(),
			}
			dereg, err := s.verifyDeregister()
			assert.NoError(err)
			tt.expectFunc(dereg)
		})
	}

}
