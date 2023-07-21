// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"testing"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestIsSingleDataNodeCluster tests the IsSingleDataNodeCluster function
// GIVEN a VZ CR
// WHEN IsSingleDataNodeCluster is called
// THEN expected boolean is returned
func TestIsSingleDataNodeCluster(t *testing.T) {
	defer func() {
		GetControllerRuntimeClient = GetClient
	}()

	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	GetControllerRuntimeClient = func() (client.Client, error) {
		return fakeClient, nil
	}

	fakeCtx := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{Spec: v1alpha1.VerrazzanoSpec{Profile: "dev"}}, nil, false, profilesRelativePath)
	assert.True(t, IsSingleDataNodeCluster(fakeCtx))

	fakeCtx = spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false, profilesRelativePath)
	assert.False(t, IsSingleDataNodeCluster(fakeCtx))
}

// TestIsUpgrade tests the IsUpgrade function
// GIVEN a call to IsUpgrade
// WHEN there are older PVs is called
// THEN expected boolean is returned
func TestIsUpgrade(t *testing.T) {
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).Build()
	nodePools := []NodePool{
		{
			Component: "es-data",
			Roles:     []string{"data"},
			Replicas:  1,
		},
	}
	fakeCtx := spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false, profilesRelativePath)
	assert.False(t, IsUpgrade(fakeCtx, nodePools))

	fakeClient = fake.NewClientBuilder().WithScheme(testScheme).WithLists(
		&v1.PersistentVolumeList{Items: []v1.PersistentVolume{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-pv-1",
					Labels: map[string]string{
						opensearchNodeLabel: "es-data",
						clusterLabel:        clusterName,
					},
				},
				Spec: v1.PersistentVolumeSpec{
					ClaimRef: &v1.ObjectReference{
						Namespace: constants.VerrazzanoSystemNamespace,
						Name:      "vmi-system-es-data-1",
					},
					PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimDelete,
				},
			},
		}}).Build()
	fakeCtx = spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{}, nil, false, profilesRelativePath)
	assert.True(t, IsUpgrade(fakeCtx, nodePools))
}

// TestIsSingleMasterNodeCluster tests the IsSingleMasterNodeCluster function
// GIVEN a VZ CR
// WHEN IsSingleMasterNodeCluster is called
// THEN expected boolean is returned
func TestIsSingleMasterNodeCluster(t *testing.T) {
	nodePools := []NodePool{
		{
			Component: "es-master",
			Roles:     []string{"master"},
			Replicas:  1,
		},
	}
	assert.True(t, IsSingleMasterNodeCluster(nodePools))

	nodePools = []NodePool{
		{
			Component: "es-master",
			Roles:     []string{"master"},
			Replicas:  3,
		},
	}
	assert.False(t, IsSingleMasterNodeCluster(nodePools))

	nodePools = []NodePool{
		{
			Component: "es-master",
			Roles:     []string{"cluster_manager"},
			Replicas:  1,
		},
	}
	assert.True(t, IsSingleMasterNodeCluster(nodePools))
}
