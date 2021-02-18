// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testMCConfigMapName = "unit-mccm"
const testMCConfigMapNamespace = "unit-mccm-namespace"

var mcConfigMapTestLabels = map[string]string{"label1": "test1"}
var mcConfigMapTestUpdatedLabels = map[string]string{"label1": "test1updated"}

// TestCreateMCConfigMap tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterConfigMap objects
// WHEN the a new object exists
// THEN ensure that the MultiClusterConfigMap is created.
func TestCreateMCConfigMap(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCConfigMap, err := getSampleMCConfigMap("testdata/multicluster-configmap.yaml")
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterConfigMap")
	}

	// Admin Cluster - expect call to list MultiClusterConfigMap objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterConfigMapList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcConfigMapList *clustersv1alpha1.MultiClusterConfigMapList, opts ...*client.ListOptions) error {
			mcConfigMapList.Items = append(mcConfigMapList.Items, testMCConfigMap)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterConfigMap from the list returned by the admin cluster
	//                   Return the resource does not exist
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCConfigMapNamespace, Name: testMCConfigMapName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "MultiClusterConfigMap"}, testMCConfigMapName))

	// Managed Cluster - expect call to create a MultiClusterConfigMap
	mcMock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap, opts ...client.CreateOption) error {
			assert.Equal(testMCConfigMapNamespace, mcConfigMap.Namespace, "mcConfigMap namespace did not match")
			assert.Equal(testMCConfigMapName, mcConfigMap.Name, "mcConfigMap name did not match")
			assert.Equal(mcConfigMapTestLabels, mcConfigMap.Labels, "mcConfigMap labels did not match")
			assert.Equal(testClusterName, mcConfigMap.Spec.Placement.Clusters[0].Name, "mcConfigMap does not contain expected placement")
			assert.Equal("simplevalue", mcConfigMap.Spec.Template.Data["simple.key"])
			return nil
		})

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		LocalClient:        mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCConfigMapObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestUpdateMCConfigMap tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterConfigMap objects
// WHEN the a object exists
// THEN ensure that the MultiClusterConfigMap is updated.
func TestUpdateMCConfigMap(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)
	//mcMockStatusWriter := mocks.NewMockStatusWriter(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCConfigMap, err := getSampleMCConfigMap("testdata/multicluster-configmap.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterConfigMap")

	testMCConfigMapUpdate, err := getSampleMCConfigMap("testdata/multicluster-configmap-update.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterConfigMap")

	// Admin Cluster - expect call to list MultiClusterConfigMap objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterConfigMapList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcConfigMapList *clustersv1alpha1.MultiClusterConfigMapList, opts ...*client.ListOptions) error {
			mcConfigMapList.Items = append(mcConfigMapList.Items, testMCConfigMapUpdate)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterConfigMap from the list returned by the admin cluster
	//                   Return the resource with some values different than what the admin cluster returned
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCConfigMapNamespace, Name: testMCConfigMapName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap) error {
			testMCConfigMap.DeepCopyInto(mcConfigMap)
			return nil
		})

	// Managed Cluster - expect call to update a MultiClusterConfigMap
	//                   Verify request had the updated values
	mcMock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcConfigMap *clustersv1alpha1.MultiClusterConfigMap, opts ...client.UpdateOption) error {
			assert.Equal(testMCConfigMapNamespace, mcConfigMap.Namespace, "mcConfigMap namespace did not match")
			assert.Equal(testMCConfigMapName, mcConfigMap.Name, "mcConfigMap name did not match")
			assert.Equal(mcConfigMapTestUpdatedLabels, mcConfigMap.Labels, "mcConfigMap labels did not match")
			assert.Equal("simplevalue2", mcConfigMap.Spec.Template.Data["simple.key"])
			return nil
		})

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		LocalClient:        mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCConfigMapObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestMCConfigMapPlacement tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterConfigMap objects
// WHEN the a object exists that is not targeted for the cluster
// THEN ensure that the MultiClusterConfigMap is not created or updated
func TestMCConfigMapPlacement(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCConfigMap, err := getSampleMCConfigMap("testdata/multicluster-configmap.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterConfigMap")
	testMCConfigMap.Spec.Placement.Clusters[0].Name = "not-my-cluster"

	// Admin Cluster - expect call to list MultiClusterConfigMap objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterConfigMapList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mCConfigMapList *clustersv1alpha1.MultiClusterConfigMapList, opts ...*client.ListOptions) error {
			mCConfigMapList.Items = append(mCConfigMapList.Items, testMCConfigMap)
			return nil
		})

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		LocalClient:        mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCConfigMapObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// getSampleMCConfigMap creates and returns a sample MultiClusterConfigMap used in tests
func getSampleMCConfigMap(filePath string) (clustersv1alpha1.MultiClusterConfigMap, error) {
	mcComp := clustersv1alpha1.MultiClusterConfigMap{}
	sampleConfigMapFile, err := filepath.Abs(filePath)
	if err != nil {
		return mcComp, err
	}

	rawMcComp, err := clusters.ReadYaml2Json(sampleConfigMapFile)
	if err != nil {
		return mcComp, err
	}

	err = json.Unmarshal(rawMcComp, &mcComp)
	return mcComp, err
}
