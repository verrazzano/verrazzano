// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"testing"
	"time"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	vmcDeleted := v1alpha1.VerrazzanoManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         constants.VerrazzanoMultiClusterNamespace,
			Name:              testClusterName,
			DeletionTimestamp: &metav1.Time{Time: time.Now()},
		},
	}
	vmcNotDeleted := v1alpha1.VerrazzanoManagedCluster{
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
		vmc     *v1alpha1.VerrazzanoManagedCluster
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
			err := v1alpha1.AddToScheme(scheme)
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
