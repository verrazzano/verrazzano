// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package appoper

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/clusters/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"go.uber.org/zap"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestAppOperatorPostUpgradeNoDeleteClusterRoleBinding tests the PostUpgrade function
// GIVEN a call to PostUpgrade
//  WHEN a VMC exists but no associated ClusterRoleBinding
//  THEN no delete of a ClusterRoleBinding
func TestAppOperatorPostUpgradeNoDeleteClusterRoleBinding(t *testing.T) {
	clusterName := "managed1"
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	fakeClient := fake.NewFakeClientWithScheme(newScheme(),
		&v1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterName,
			},
		})
	err := NewComponent().PostUpgrade(spi.NewContext(zap.S(), fakeClient, vz, false))
	assert.NoError(t, err)
}

// TestAppOperatorPostUpgradeDeleteClusterRoleBinding tests the PostUpgrade function
// GIVEN a call to PostUpgrade
//  WHEN a VMC exists with an associated ClusterRoleBinding
//  THEN successful delete of the ClusterRoleBinding
func TestAppOperatorPostUpgradeDeleteClusterRoleBinding(t *testing.T) {
	clusterName := "managed1"
	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				DNS: &vzapi.DNSComponent{
					OCI: &vzapi.OCI{
						DNSZoneName: "mydomain.com",
					},
				},
			},
		},
	}
	fakeClient := fake.NewFakeClientWithScheme(newScheme(),
		&v1alpha1.VerrazzanoManagedCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: clusterName,
			},
		},
		&rbacv1.ClusterRoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("verrazzano-cluster-%s", clusterName),
			},
			Subjects: nil,
			RoleRef:  rbacv1.RoleRef{},
		})
	err := NewComponent().PostUpgrade(spi.NewContext(zap.S(), fakeClient, vz, false))
	assert.Nil(t, err)

	// Verify the ClusterRolebinding was deleted
	clusterRoleBinding := rbacv1.ClusterRoleBinding{}
	err = fakeClient.Get(context.TODO(), types.NamespacedName{Name: fmt.Sprintf("verrazzano-cluster-%s", clusterName)}, &clusterRoleBinding)
	assert.Error(t, err)
	assert.True(t, errors.IsNotFound(err))
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	clustersv1alpha1.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	rbacv1.AddToScheme(scheme)
	return scheme
}
