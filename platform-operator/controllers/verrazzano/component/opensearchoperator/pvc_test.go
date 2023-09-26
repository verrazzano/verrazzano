// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"testing"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	testScheme = runtime.NewScheme()
	_          = k8scheme.AddToScheme(testScheme)
	_          = vmov1.AddToScheme(testScheme)
)

// TestGetOSPersistentVolumes tests the getOSPersistentVolumes function
// GIVEN a list of PVs created by VMO and OS nodes
// WHEN getOSPersistentVolumes is called
// THEN only PVs corresponding to OS nodes are returned
func TestGetOSPersistentVolumes(t *testing.T) {
	fakeCtx := newFakeContext()
	nodePools := getVZ().Spec.Components.Elasticsearch.Nodes
	pvList, err := getOSPersistentVolumes(fakeCtx, nodePools)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(pvList))
}

// TestSetPVsToRetain tests the setPVsToRetain function
// GIVEN a list of PVs created by VMO and OS nodes
// WHEN setPVsToRetain is called
// THEN only PVs corresponding to OS nodes are set to Retain
func TestSetPVsToRetain(t *testing.T) {
	fakeCtx := newFakeContext()
	nodePools := getVZ().Spec.Components.Elasticsearch.Nodes
	err := setPVsToRetain(fakeCtx, nodePools)
	assert.NoError(t, err)
	// When retaining the PVs, the old reclaim policy label is set
	pvList, err := common.GetPVsBasedOnLabel(fakeCtx, constants.OldReclaimPolicyLabel, "Delete")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(pvList))
	for _, pv := range pvList {
		assert.Equal(t, v1.PersistentVolumeReclaimRetain, pv.Spec.PersistentVolumeReclaimPolicy)
	}
}

// TestCreateNewPVCs tests the createNewPVCs function
// GIVEN a list of PVs created by VMO and OS nodes
// WHEN createNewPVCs is called
// THEN new PVCs corresponding to PVs are created
func TestCreateNewPVCs(t *testing.T) {
	fakeCtx := newFakeContext()
	nodePools := getVZ().Spec.Components.Elasticsearch.Nodes
	err := createNewPVCs(fakeCtx, nodePools)
	assert.NoError(t, err)

	pvcList, err := common.GetPVCsBasedOnLabel(fakeCtx, clusterLabel, clusterName)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(pvcList))
	for _, pvc := range pvcList {
		assert.Contains(t, pvc.Name, "data-opensearch")
	}
}

// newFakeContext returns a fake context with fake client built with list of PVs and VZ object
func newFakeContext() spi.ComponentContext {
	fakePVList := getFakePersistentVolumeList()
	fakeClient := fake.NewClientBuilder().WithScheme(testScheme).WithLists(fakePVList).Build()

	fakeCtx := spi.NewFakeContext(fakeClient, getVZ(), nil, false)
	return fakeCtx
}

// getFakePersistentVolumeList returns a PVList for testing
func getFakePersistentVolumeList() *v1.PersistentVolumeList {
	return &v1.PersistentVolumeList{Items: []v1.PersistentVolume{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pv-1",
				Labels: map[string]string{
					opensearchNodeLabel: esData,
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
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pv-2",
				Labels: map[string]string{
					opensearchNodeLabel: esData,
				},
			},
			Spec: v1.PersistentVolumeSpec{
				ClaimRef: &v1.ObjectReference{
					Namespace: constants.VerrazzanoSystemNamespace,
					Name:      "vmi-system-es-data-yewhf",
				},
				PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimDelete},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pv-3",
				Labels: map[string]string{
					opensearchNodeLabel: dataIngest,
				},
			},
			Spec: v1.PersistentVolumeSpec{
				ClaimRef: &v1.ObjectReference{
					Namespace: constants.VerrazzanoSystemNamespace,
					Name:      "vmi-system-data-ingest",
				},
				PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimDelete},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-pv-4",
				Labels: map[string]string{
					opensearchNodeLabel: esMaster,
				},
			},
			Spec: v1.PersistentVolumeSpec{
				ClaimRef: &v1.ObjectReference{
					Namespace: constants.VerrazzanoSystemNamespace,
					Name:      "elasticsearch-master-vmi-system-es-master-2",
				},
				PersistentVolumeReclaimPolicy: v1.PersistentVolumeReclaimDelete},
		},
	}}
}
