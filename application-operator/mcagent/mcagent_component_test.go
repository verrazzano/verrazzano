// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testMCComponentName = "unit-mccomp"
const testMCComponentNamespace = "unit-mccomp-namespace"

var mcComponentTestExpectedLabels = map[string]string{"label1": "test1",
	vzconst.VerrazzanoManagedLabelKey: constants.LabelVerrazzanoManagedDefault}
var mcComponentTestExpectedLabelsOnUpdate = map[string]string{"label1": "test1updated",
	vzconst.VerrazzanoManagedLabelKey: constants.LabelVerrazzanoManagedDefault}

// TestCreateMCComponent tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterComponent objects
// WHEN the a new object exists
// THEN ensure that the MultiClusterComponent is created.
func TestCreateMCComponent(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCComponent, err := getSampleMCComponent("testdata/multicluster-component.yaml")
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterComponent")
	}

	// Admin Cluster - expect call to list MultiClusterComponent objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterComponentList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcComponentList *clustersv1alpha1.MultiClusterComponentList, listOptions *client.ListOptions) error {
			assert.Equal(testMCComponentNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcComponentList.Items = append(mcComponentList.Items, testMCComponent)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterComponent from the list returned by the admin cluster
	//                   Return the resource does not exist
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCComponentNamespace, Name: testMCComponentName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "MultiClusterComponent"}, testMCComponentName))

	// Managed Cluster - expect call to create a MultiClusterComponent
	mcMock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcComponent *clustersv1alpha1.MultiClusterComponent, opts ...client.CreateOption) error {
			assert.Equal(testMCComponentNamespace, mcComponent.Namespace, "mccomponent namespace did not match")
			assert.Equal(testMCComponentName, mcComponent.Name, "mccomponent name did not match")
			assert.Equal(mcComponentTestExpectedLabels, mcComponent.Labels, "mccomponent labels did not match")
			assert.Equal(testClusterName, mcComponent.Spec.Placement.Clusters[0].Name, "mccomponent does not contain expected placement")
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterComponent objects - return same list as admin
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterComponentList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcComponentList *clustersv1alpha1.MultiClusterComponentList, listOptions *client.ListOptions) error {
			assert.Equal(testMCComponentNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcComponentList.Items = append(mcComponentList.Items, testMCComponent)
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
	err = s.syncMCComponentObjects(testMCComponentNamespace)

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestUpdateMCComponent tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterComponent objects
// WHEN the a object exists
// THEN ensure that the MultiClusterComponent is updated.
func TestUpdateMCComponent(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCComponent, err := getSampleMCComponent("testdata/multicluster-component.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterComponent")

	testMCComponentUpdate, err := getSampleMCComponent("testdata/multicluster-component-update.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterComponent")

	// Admin Cluster - expect call to list MultiClusterComponent objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterComponentList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcComponentList *clustersv1alpha1.MultiClusterComponentList, listOptions *client.ListOptions) error {
			assert.Equal(testMCComponentNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcComponentList.Items = append(mcComponentList.Items, testMCComponentUpdate)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterComponent from the list returned by the admin cluster
	//                   Return the resource with some values different than what the admin cluster returned
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCComponentNamespace, Name: testMCComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcComponent *clustersv1alpha1.MultiClusterComponent) error {
			testMCComponent.DeepCopyInto(mcComponent)
			return nil
		})

	// Managed Cluster - expect call to update a MultiClusterComponent
	//                   Verify request had the updated values
	mcMock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcComponent *clustersv1alpha1.MultiClusterComponent, opts ...client.UpdateOption) error {
			assert.Equal(testMCComponentNamespace, mcComponent.Namespace, "mccomponent namespace did not match")
			assert.Equal(testMCComponentName, mcComponent.Name, "mccomponent name did not match")
			assert.Equal(mcComponentTestExpectedLabelsOnUpdate, mcComponent.Labels, "mccomponent labels did not match")
			workload := v1alpha2.ContainerizedWorkload{}
			err := json.Unmarshal(mcComponent.Spec.Template.Spec.Workload.Raw, &workload)
			assert.NoError(err, "failed to unmarshal the containerized workload")
			assert.Equal("hello2", workload.Spec.Containers[0].Name)
			assert.Equal("ghcr.io/oracle/oraclelinux:7-slim2", workload.Spec.Containers[0].Image)
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterComponent objects - return same list as admin
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterComponentList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcComponentList *clustersv1alpha1.MultiClusterComponentList, listOptions *client.ListOptions) error {
			assert.Equal(testMCComponentNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcComponentList.Items = append(mcComponentList.Items, testMCComponent)
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
	err = s.syncMCComponentObjects(testMCComponentNamespace)

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestDeleteMCComponent tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterComponent objects
// WHEN the object exists on the local cluster but not on the admin cluster
// THEN ensure that the MultiClusterComponent is deleted.
func TestDeleteMCComponent(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCComponent, err := getSampleMCComponent("testdata/multicluster-component.yaml")
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterComponent")
	}
	testMCComponentOrphan, err := getSampleMCComponent("testdata/multicluster-component.yaml")
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterComponent")
	}
	testMCComponentOrphan.Name = "orphaned-resource"

	// Admin Cluster - expect call to list MultiClusterComponent objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterComponentList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcComponentList *clustersv1alpha1.MultiClusterComponentList, listOptions *client.ListOptions) error {
			assert.Equal(testMCComponentNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcComponentList.Items = append(mcComponentList.Items, testMCComponent)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterComponent from the list returned by the admin cluster
	//                   Return the resource
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCComponentNamespace, Name: testMCComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcComponent *clustersv1alpha1.MultiClusterComponent) error {
			testMCComponent.DeepCopyInto(mcComponent)
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterComponent objects - return list including an orphaned object
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterComponentList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcComponentList *clustersv1alpha1.MultiClusterComponentList, listOptions *client.ListOptions) error {
			assert.Equal(testMCComponentNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcComponentList.Items = append(mcComponentList.Items, testMCComponent)
			mcComponentList.Items = append(mcComponentList.Items, testMCComponentOrphan)
			return nil
		})

	// Managed Cluster - expect a call to delete a MultiClusterComponent object
	mcMock.EXPECT().
		Delete(gomock.Any(), gomock.Eq(&testMCComponentOrphan), gomock.Any()).
		Return(nil)

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		LocalClient:        mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCComponentObjects(testMCComponentNamespace)

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestMCComponentPlacement tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterComponent objects
// WHEN the a object exists that is not targeted for the cluster
// THEN ensure that the MultiClusterComponent is not created or updated
func TestMCComponentPlacement(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	adminMCComponent, err := getSampleMCComponent("testdata/multicluster-component.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterComponent")
	adminMCComponent.Spec.Placement.Clusters[0].Name = "not-my-cluster"

	localMCComponent, err := getSampleMCComponent("testdata/multicluster-component.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterComponent")

	// Admin Cluster - expect call to list MultiClusterComponent objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterComponentList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcComponentList *clustersv1alpha1.MultiClusterComponentList, listOptions *client.ListOptions) error {
			assert.Equal(testMCComponentNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcComponentList.Items = append(mcComponentList.Items, adminMCComponent)
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterComponent objects - return same list as admin
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterComponentList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcComponentList *clustersv1alpha1.MultiClusterComponentList, listOptions *client.ListOptions) error {
			assert.Equal(testMCComponentNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcComponentList.Items = append(mcComponentList.Items, localMCComponent)
			return nil
		})

	// Managed Cluster - expect a call to delete a MultiClusterComponent object
	mcMock.EXPECT().
		Delete(gomock.Any(), gomock.Eq(&localMCComponent), gomock.Any()).
		Return(nil)

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		LocalClient:        mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCComponentObjects(testMCComponentNamespace)

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// getSampleMCComponent creates and returns a sample MultiClusterComponent used in tests
func getSampleMCComponent(filePath string) (clustersv1alpha1.MultiClusterComponent, error) {
	mcComp := clustersv1alpha1.MultiClusterComponent{}
	sampleComponentFile, err := filepath.Abs(filePath)
	if err != nil {
		return mcComp, err
	}

	rawMcComp, err := clusterstest.ReadYaml2Json(sampleComponentFile)
	if err != nil {
		return mcComp, err
	}

	err = json.Unmarshal(rawMcComp, &mcComp)
	return mcComp, err
}
