// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testComponent, err := getSampleOamComponent("testdata/hello-component.yaml")
	assert.NoError(err, "failed to read sample data for OAM Component")

	// Admin Cluster - expect call to list MultiClusterApplicationConfiguration objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, listOptions *client.ListOptions) error {
			assert.Equal(testMCAppConfigNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfig)
			return nil
		})

	// Admin Cluster - expect a call to get the OAM Component of the application
	adminMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testComponent.Namespace, Name: testComponent.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1alpha2.Component) error {
			component.ObjectMeta = testComponent.ObjectMeta
			component.Spec = testComponent.Spec
			return nil
		})

	// Managed Cluster - expect a call to get the OAM Component of the application - return that it does not exist
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testComponent.Namespace, Name: testComponent.Name}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: testComponent.GroupVersionKind().Group, Resource: "Component"}, testComponent.Name))

	// Managed Cluster - expect call to create a OAM Component
	mcMock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, component *oamv1alpha2.Component, opts ...client.CreateOption) error {
			assert.Equal(testComponent.Namespace, component.Namespace, "OAM component namespace did not match")
			assert.Equal(testComponent.Name, component.Name, "OAM component name did not match")
			assert.Equal(mcAppConfigTestLabels, component.Labels, "OAM component labels did not match")
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
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, listOptions *client.ListOptions) error {
			assert.Equal(testMCAppConfigNamespace, listOptions.Namespace, "list request did not have expected namespace")
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
	err = s.syncMCApplicationConfigurationObjects(testMCAppConfigNamespace)

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

	testComponent1, err := getSampleOamComponent("testdata/hello-component.yaml")
	assert.NoError(err, "failed to read sample data for OAM Component")

	testComponent2, err := getSampleOamComponent("testdata/goodbye-component.yaml")
	assert.NoError(err, "failed to read sample data for OAM Component")

	// Admin Cluster - expect call to list MultiClusterApplicationConfiguration objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, listOptions *client.ListOptions) error {
			assert.Equal(testMCAppConfigNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfigUpdate)
			return nil
		})

	// Admin Cluster - expect a call to get the OAM Component of the application
	adminMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testComponent1.Namespace, Name: testComponent1.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1alpha2.Component) error {
			component.ObjectMeta = testComponent1.ObjectMeta
			component.Spec = testComponent1.Spec
			return nil
		})

	// Managed Cluster - expect a call to get the OAM Component of the application - return that it exists
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testComponent1.Namespace, Name: testComponent1.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1alpha2.Component) error {
			component.ObjectMeta = testComponent1.ObjectMeta
			component.Spec = testComponent1.Spec
			return nil
		})

	// Admin Cluster - expect a call to get the second OAM Component of the application
	adminMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testComponent2.Namespace, Name: testComponent2.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1alpha2.Component) error {
			component.ObjectMeta = testComponent2.ObjectMeta
			component.Spec = testComponent2.Spec
			return nil
		})

	// Managed Cluster - expect a call to get the second OAM Component of the application - return that it exists
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testComponent2.Namespace, Name: testComponent2.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1alpha2.Component) error {
			component.ObjectMeta = testComponent2.ObjectMeta
			component.Spec = testComponent2.Spec
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
			assert.Equal("hello-component", comp0.ComponentName)
			assert.Equal("goodbye-component", comp1.ComponentName)
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterApplicationConfiguration objects - return same list as admin
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, listOptions *client.ListOptions) error {
			assert.Equal(testMCAppConfigNamespace, listOptions.Namespace, "list request did not have expected namespace")
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
	err = s.syncMCApplicationConfigurationObjects(testMCAppConfigNamespace)

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
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testMCAppConfigOrphan, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testComponent, err := getSampleOamComponent("testdata/hello-component.yaml")
	assert.NoError(err, "failed to read sample data for OAM Component")

	testMCAppConfigOrphan.Name = "orphaned-resource"

	// Admin Cluster - expect call to list MultiClusterApplicationConfiguration objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterApplicationConfigurationList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, listOptions *client.ListOptions) error {
			assert.Equal(testMCAppConfigNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfig)
			return nil
		})

	// Admin Cluster - expect a call to get the OAM Component of the application
	adminMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testComponent.Namespace, Name: testComponent.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1alpha2.Component) error {
			component.ObjectMeta = testComponent.ObjectMeta
			component.Spec = testComponent.Spec
			return nil
		})

	// Managed Cluster - expect a call to get the OAM Component of the application - return that it exists
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testComponent.Namespace, Name: testComponent.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1alpha2.Component) error {
			component.ObjectMeta = testComponent.ObjectMeta
			component.Spec = testComponent.Spec
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
		DoAndReturn(func(ctx context.Context, mcAppConfigList *clustersv1alpha1.MultiClusterApplicationConfigurationList, listOptions *client.ListOptions) error {
			assert.Equal(testMCAppConfigNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfig)
			mcAppConfigList.Items = append(mcAppConfigList.Items, testMCAppConfigOrphan)
			return nil
		})

	// Managed Cluster - expect a call to delete a MultiClusterApplicationConfiguration object
	mcMock.EXPECT().
		Delete(gomock.Any(), gomock.Eq(&testMCAppConfigOrphan), gomock.Any()).
		Return(nil)

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		LocalClient:        mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCApplicationConfigurationObjects(testMCAppConfigNamespace)

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

	// Test data
	adminMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")
	adminMCAppConfig.Spec.Placement.Clusters[0].Name = "not-my-cluster"

	loclaMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	adminClient := fake.NewFakeClientWithScheme(newScheme(),
		&clustersv1alpha1.MultiClusterApplicationConfigurationList{
			Items: []clustersv1alpha1.MultiClusterApplicationConfiguration{adminMCAppConfig}})

	localClient := fake.NewFakeClientWithScheme(newScheme(),
		&clustersv1alpha1.MultiClusterApplicationConfigurationList{
			Items: []clustersv1alpha1.MultiClusterApplicationConfiguration{loclaMCAppConfig}})

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCApplicationConfigurationObjects(testMCAppConfigNamespace)

	// Verify the local MultiClusterApplicationConiguration was deleted
	assert.NoError(err)
	delAppConfig := &clustersv1alpha1.MultiClusterApplicationConfiguration{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: loclaMCAppConfig.Name, Namespace: loclaMCAppConfig.Namespace}, delAppConfig)
	assert.True(errors.IsNotFound(err))
}

// TestSyncComponentList tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterApplicationConfiguration objects
// WHEN it contains a list of OAM Components
// THEN ensure that the embedded OAM Components are created or updated
func TestSyncComponentList(t *testing.T) {
	appName := "test"
	appNamespace := "test-ns"
	compName1 := "test-comp-1"
	compName2 := "test-comp-2"
	param1 := "parameter-1"
	param2 := "parameter-2"
	testLabel := "test-label"
	testAnnot := "test-annotation"

	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Create a fake client for the admin cluster
	adminClient := fake.NewFakeClientWithScheme(newScheme(),
		&oamv1alpha2.Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:        compName1,
				Namespace:   appNamespace,
				Labels:      map[string]string{"test": testLabel},
				Annotations: map[string]string{"test": testAnnot}},
			Spec: oamv1alpha2.ComponentSpec{
				Parameters: []oamv1alpha2.ComponentParameter{
					{
						Name: param1,
					},
				},
			},
		},
		&oamv1alpha2.Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:        compName2,
				Namespace:   appNamespace,
				Labels:      map[string]string{"test": testLabel},
				Annotations: map[string]string{"test": testAnnot}},
			Spec: oamv1alpha2.ComponentSpec{
				Parameters: []oamv1alpha2.ComponentParameter{
					{
						Name: param2,
					},
				},
			},
		},
	)

	// Create a fake client for the local cluster
	localClient := fake.NewFakeClientWithScheme(newScheme())

	// MultiClusterApplicationConfiguration test data
	mcAppConfig := clustersv1alpha1.MultiClusterApplicationConfiguration{
		ObjectMeta: metav1.ObjectMeta{Name: appName, Namespace: appNamespace},
		Spec: clustersv1alpha1.MultiClusterApplicationConfigurationSpec{
			Template: clustersv1alpha1.ApplicationConfigurationTemplate{
				Spec: oamv1alpha2.ApplicationConfigurationSpec{
					Components: []oamv1alpha2.ApplicationConfigurationComponent{
						{
							ComponentName: compName1,
						},
						{
							ComponentName: compName2,
						},
					},
				},
			},
		},
	}

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err := s.syncComponentList(mcAppConfig)
	assert.NoError(err)

	// Verify the components were created locally
	component1 := &oamv1alpha2.Component{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: compName1, Namespace: appNamespace}, component1)
	assert.NoError(err)
	assert.Equal(param1, component1.Spec.Parameters[0].Name)
	assert.Equal(testLabel, component1.ObjectMeta.Labels["test"])
	assert.Equal(testAnnot, component1.ObjectMeta.Annotations["test"])

	component2 := &oamv1alpha2.Component{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: compName2, Namespace: appNamespace}, component2)
	assert.NoError(err)
	assert.Equal(param2, component2.Spec.Parameters[0].Name)
	assert.Equal(testLabel, component2.ObjectMeta.Labels["test"])
	assert.Equal(testAnnot, component2.ObjectMeta.Annotations["test"])
}

// getSampleMCAppConfig creates and returns a sample MultiClusterApplicationConfiguration used in tests
func getSampleMCAppConfig(filePath string) (clustersv1alpha1.MultiClusterApplicationConfiguration, error) {
	mcAppConfig := clustersv1alpha1.MultiClusterApplicationConfiguration{}
	sampleAppConfigFile, err := filepath.Abs(filePath)
	if err != nil {
		return mcAppConfig, err
	}

	rawResource, err := clusterstest.ReadYaml2Json(sampleAppConfigFile)
	if err != nil {
		return mcAppConfig, err
	}

	err = json.Unmarshal(rawResource, &mcAppConfig)
	return mcAppConfig, err
}

// getSampleOamComponent creates and returns a sample OAM Component
func getSampleOamComponent(filePath string) (oamv1alpha2.Component, error) {
	component := oamv1alpha2.Component{}
	sampleComponentFile, err := filepath.Abs(filePath)
	if err != nil {
		return component, err
	}

	rawResource, err := clusterstest.ReadYaml2Json(sampleComponentFile)
	if err != nil {
		return component, err
	}

	err = json.Unmarshal(rawResource, &component)
	return component, err
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	oamv1alpha2.SchemeBuilder.AddToScheme(scheme)
	clustersv1alpha1.AddToScheme(scheme)
	return scheme
}
