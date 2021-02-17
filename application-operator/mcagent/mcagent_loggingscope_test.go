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

const testMCLoggingScopeName = "unit-mclogscope"
const testMCLoggingScopeNamespace = "unit-mclogscope-namespace"

var mcLoggingScopeTestLabels = map[string]string{"label1": "test1"}
var mcLoggingScopeTestUpdatedLabels = map[string]string{"label1": "test1updated"}

// TestCreateMCLoggingScope tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterLoggingScope objects
// WHEN the a new object exists
// THEN ensure that the MultiClusterLoggingScope is created.
func TestCreateMCLoggingScope(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCLoggingScope, err := getSampleMCLoggingScope("testdata/multicluster-loggingscope.yaml")
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterLoggingScope")
	}

	// Admin Cluster - expect call to list MultiClusterLoggingScope objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterLoggingScopeList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcLoggingScopeList *clustersv1alpha1.MultiClusterLoggingScopeList, opts ...*client.ListOptions) error {
			mcLoggingScopeList.Items = append(mcLoggingScopeList.Items, testMCLoggingScope)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterLoggingScope from the list returned by the admin cluster
	//                   Return the resource does not exist
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCLoggingScopeNamespace, Name: testMCLoggingScopeName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: testMCLoggingScopeNamespace, Resource: "MultiClusterLoggingScope"}, testMCLoggingScopeName))

	// Managed Cluster - expect call to create a MultiClusterLoggingScope
	mcMock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcLoggingScope *clustersv1alpha1.MultiClusterLoggingScope, opts ...client.CreateOption) error {
			assert.Equal(testMCLoggingScopeNamespace, mcLoggingScope.Namespace, "mcloggingscope namespace did not match")
			assert.Equal(testMCLoggingScopeName, mcLoggingScope.Name, "mcloggingscope name did not match")
			assert.Equal(mcLoggingScopeTestLabels, mcLoggingScope.Labels, "mcloggingscope labels did not match")
			assert.Equal(testClusterName, mcLoggingScope.Spec.Placement.Clusters[0].Name, "mcloggingscope does not contain expected placement")
			assert.Equal("logScopeSecret", mcLoggingScope.Spec.Template.Spec.SecretName, "mcloggingscope does not contain expected secret")
			assert.Equal("myLocalEsHost", mcLoggingScope.Spec.Template.Spec.ElasticSearchHost, "mcloggingscope does not contain expected elasticSearchHost")
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
	err = s.syncMCLoggingScopeObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestUpdateMCLoggingScope tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterLoggingScope objects
// WHEN the a object exists
// THEN ensure that the MultiClusterLoggingScope is updated.
func TestUpdateMCLoggingScope(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCLoggingScope, err := getSampleMCLoggingScope("testdata/multicluster-loggingscope.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterLoggingScope")

	testMCLoggingScopeUpdate, err := getSampleMCLoggingScope("testdata/multicluster-loggingscope-update.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterLoggingScope")

	// Admin Cluster - expect call to list MultiClusterLoggingScope objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterLoggingScopeList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcLoggingScopeList *clustersv1alpha1.MultiClusterLoggingScopeList, opts ...*client.ListOptions) error {
			mcLoggingScopeList.Items = append(mcLoggingScopeList.Items, testMCLoggingScopeUpdate)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterLoggingScope from the list returned by the admin cluster
	//                   Return the resource with some values different than what the admin cluster returned
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCLoggingScopeNamespace, Name: testMCLoggingScopeName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcLoggingScope *clustersv1alpha1.MultiClusterLoggingScope) error {
			testMCLoggingScope.DeepCopyInto(mcLoggingScope)
			return nil
		})

	// Managed Cluster - expect call to update a MultiClusterLoggingScope
	//                   Verify request had the updated values
	mcMock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcLoggingScope *clustersv1alpha1.MultiClusterLoggingScope, opts ...client.UpdateOption) error {
			assert.Equal(testMCLoggingScopeNamespace, mcLoggingScope.Namespace, "mcloggingscope namespace did not match")
			assert.Equal(testMCLoggingScopeName, mcLoggingScope.Name, "mcloggingscope name did not match")
			assert.Equal(mcLoggingScopeTestUpdatedLabels, mcLoggingScope.Labels, "mcloggingscope labels did not match")
			assert.Equal("logScopeSecret2", mcLoggingScope.Spec.Template.Spec.SecretName, "mcloggingscope does not contain expected secret")
			assert.Equal("myLocalEsHost2", mcLoggingScope.Spec.Template.Spec.ElasticSearchHost, "mcloggingscope does not contain expected elasticSearchHost")
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
	err = s.syncMCLoggingScopeObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestMCLoggingScopePlacement tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterLoggingScope objects
// WHEN the a object exists that is not targeted for the cluster
// THEN ensure that the MultiClusterLoggingScope is not created or updated
func TestMCLoggingScopePlacement(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCLoggingScope, err := getSampleMCLoggingScope("testdata/multicluster-loggingscope.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterLoggingScope")
	testMCLoggingScope.Spec.Placement.Clusters[0].Name = "not-my-cluster"

	// Admin Cluster - expect call to list MultiClusterLoggingScope objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterLoggingScopeList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcLoggingScopeList *clustersv1alpha1.MultiClusterLoggingScopeList, opts ...*client.ListOptions) error {
			mcLoggingScopeList.Items = append(mcLoggingScopeList.Items, testMCLoggingScope)
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
	err = s.syncMCLoggingScopeObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// getSampleMCLoggingScope creates and returns a sample MultiClusterLoggingScope used in tests
func getSampleMCLoggingScope(filePath string) (clustersv1alpha1.MultiClusterLoggingScope, error) {
	mcLoggingScope := clustersv1alpha1.MultiClusterLoggingScope{}
	sampleLoggingScopeFile, err := filepath.Abs(filePath)
	if err != nil {
		return mcLoggingScope, err
	}

	rawMcComp, err := clusters.ReadYaml2Json(sampleLoggingScopeFile)
	if err != nil {
		return mcLoggingScope, err
	}

	err = json.Unmarshal(rawMcComp, &mcLoggingScope)
	return mcLoggingScope, err
}
