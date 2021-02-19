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

const testMCAppConfigName = "unit-mcappconfig"
const testMCAppConfigNamespace = "unit-mcappconfig-namespace"

var mcAppConfigTestLabels = map[string]string{"label1": "test1"}
var mcAppConfigTestUpdatedLabels = map[string]string{"label1": "test1updated"}

// TestCreateMCAppConfig tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterApplicationConfiguration objects
// WHEN the a new object exists
// THEN ensure that the MultiClusterApplicationConfiguration is created.
func TestCreateMCAppConfig(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")
	}

	// Admin Cluster - expect call to list MultiClusterApplicationConfiguration objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, opts ...*client.ListOptions) error {
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfig)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterApplicationConfiguration from the list returned by the admin cluster
	//                   Return the resource does not exist
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCAppConfigNamespace, Name: testMCAppConfigName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "MultiClusterApplicationConfiguration"}, testMCAppConfigName))

	// Managed Cluster - expect call to create a MultiClusterApplicationConfiguration
	mcMock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration, opts ...client.CreateOption) error {
			assert.Equal(testMCAppConfigNamespace, mcAppConfig.Namespace, "mcappconfig namespace did not match")
			assert.Equal(testMCAppConfigName, mcAppConfig.Name, "mcappconfig name did not match")
			assert.Equal(mcAppConfigTestLabels, mcAppConfig.Labels, "mcappconfig labels did not match")
			assert.Equal(testClusterName, mcAppConfig.Spec.Placement.Clusters[0].Name, "mcappconfig does not contain expected placement")
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterApplicationConfiguration objects - return same list as admin
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, opts ...*client.ListOptions) error {
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfig)
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
	err = s.syncMCApplicationConfigurationObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestUpdateMCAppConfig tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterApplicationConfiguration objects
// WHEN the object exists
// THEN ensure that the MultiClusterApplicationConfiguration is updated.
func TestUpdateMCAppConfig(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testMCAppConfigUpdate, err := getSampleMCAppConfig("testdata/multicluster-appconfig-update.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	// Admin Cluster - expect call to list MultiClusterApplicationConfiguration objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, opts ...*client.ListOptions) error {
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfigUpdate)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterApplicationConfiguration from the list returned by the admin cluster
	//                   Return the resource with some values different than what the admin cluster returned
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCAppConfigNamespace, Name: testMCAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration) error {
			testMCAppConfig.DeepCopyInto(mcAppConfig)
			return nil
		})

	// Managed Cluster - expect call to update a MultiClusterApplicationConfiguration
	//                   Verify request had the updated values
	mcMock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration, opts ...client.UpdateOption) error {
			assert.Equal(testMCAppConfigNamespace, mcAppConfig.Namespace, "mcappconfig namespace did not match")
			assert.Equal(testMCAppConfigName, mcAppConfig.Name, "mcappconfig name did not match")
			assert.Equal(mcAppConfigTestUpdatedLabels, mcAppConfig.Labels, "mcappconfig labels did not match")

			// assert app config metadata annotations updated
			assert.Equal("Hello application updated", mcAppConfig.Spec.Template.Metadata.Annotations["description"])

			// assert component information in the app config is updated
			assert.Equal(2, len(mcAppConfig.Spec.Template.Spec.Components))
			comp0 := mcAppConfig.Spec.Template.Spec.Components[0]
			comp1 := mcAppConfig.Spec.Template.Spec.Components[1]
			assert.Equal("hello-component-updated", comp0.ComponentName)
			assert.Equal("hello-component-extra", comp1.ComponentName)
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterApplicationConfiguration objects - return same list as admin
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, opts ...*client.ListOptions) error {
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfig)
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
	err = s.syncMCApplicationConfigurationObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestDeleteMCAppConfig tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterApplicationConfiguration objects
// WHEN the object exists on the local cluster but not on the admin cluster
// THEN ensure that the MultiClusterApplicationConfiguration is deleted.
func TestDeleteMCAppConfig(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")
	}
	testMCAppConfigOrphan, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")
	}
	testMCAppConfigOrphan.Name = "orphaned-resource"

	// Admin Cluster - expect call to list MultiClusterApplicationConfiguration objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, opts ...*client.ListOptions) error {
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfig)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterApplicationConfiguration from the list returned by the admin cluster
	//                   Return the resource
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCAppConfigNamespace, Name: testMCAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration) error {
			testMCAppConfig.DeepCopyInto(mcAppConfig)
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterApplicationConfiguration objects - return list including an orphaned object
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, opts ...*client.ListOptions) error {
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfig)
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfigOrphan)
			return nil
		})

	// Managed Cluster - expect a call to delete a MultiClusterApplicationConfiguration object
	mcMock.EXPECT().
		Delete(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcAppConfig *clustersv1alpha1.MultiClusterApplicationConfiguration, opts ...*client.ListOptions) error {
			assert.Equal(testMCAppConfigOrphan.Name, mcAppConfig.Name, "unexpected object being deleted")
			assert.Equal(testMCAppConfigOrphan.Namespace, mcAppConfig.Namespace, "unexpected object being deleted")
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
	err = s.syncMCApplicationConfigurationObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestMCAppConfigPlacement tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterApplicationConfiguration objects
// WHEN an object exists that is not targeted for the cluster
// THEN ensure that the MultiClusterApplicationConfiguration is not created or updated
func TestMCAppConfigPlacement(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")
	testMCAppConfig.Spec.Placement.Clusters[0].Name = "not-my-cluster"

	// Admin Cluster - expect call to list MultiClusterApplicationConfiguration objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, opts ...*client.ListOptions) error {
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfig)
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterApplicationConfiguration objects - return same list as admin
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, opts ...*client.ListOptions) error {
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfig)
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
	err = s.syncMCApplicationConfigurationObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// getSampleMCAppConfig creates and returns a sample MultiClusterApplicationConfiguration used in tests
func getSampleMCAppConfig(filePath string) (clustersv1alpha1.MultiClusterApplicationConfiguration, error) {
	mcAppConfig := clustersv1alpha1.MultiClusterApplicationConfiguration{}
	sampleAppConfigFile, err := filepath.Abs(filePath)
	if err != nil {
		return mcAppConfig, err
	}

	rawResource, err := clusters.ReadYaml2Json(sampleAppConfigFile)
	if err != nil {
		return mcAppConfig, err
	}

	err = json.Unmarshal(rawResource, &mcAppConfig)
	return mcAppConfig, err
}
