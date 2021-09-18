// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	oamv1alpha2 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testMCAppConfigNamespace = "unit-mcappconfig-namespace"

var mcAppConfigTestLabels = map[string]string{"label1": "test1"}
var mcAppConfigTestUpdatedLabels = map[string]string{"label1": "test1updated"}

// TestCreateMCAppConfig tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterApplicationConfiguration objects
// WHEN the new object exists
// THEN ensure that the MultiClusterApplicationConfiguration and its associated OAM Component are created.
func TestCreateMCAppConfig(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Test data
	testMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testComponent, err := getSampleOamComponent("testdata/hello-component.yaml")
	assert.NoError(err, "failed to read sample data for OAM Component")

	adminClient := fake.NewFakeClientWithScheme(newScheme(), &testMCAppConfig, &testComponent)

	localClient := fake.NewFakeClientWithScheme(newScheme())

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCApplicationConfigurationObjects(testMCAppConfigNamespace)

	// Validate the results
	assert.NoError(err)

	// Verify the associated OAM component got created on local cluster
	component := &oamv1alpha2.Component{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testComponent.Name, Namespace: testComponent.Namespace}, component)
	assert.NoError(err)

	// Verify MultiClusterApplicationConfiguration got created on local cluster
	mcAppConfig := &clustersv1alpha1.MultiClusterApplicationConfiguration{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testMCAppConfig.Name, Namespace: testMCAppConfig.Namespace}, mcAppConfig)
	assert.NoError(err)
	assert.Equal(mcAppConfigTestLabels, mcAppConfig.Labels, "mcappconfig labels did not match")
	assert.Equal(testClusterName, mcAppConfig.Spec.Placement.Clusters[0].Name, "mcappconfig does not contain expected placement")
}

// TestUpdateMCAppConfig tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterApplicationConfiguration objects
// WHEN the object exists
// THEN ensure that the MultiClusterApplicationConfiguration is updated.
func TestUpdateMCAppConfig(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Test data
	testMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testMCAppConfigUpdate, err := getSampleMCAppConfig("testdata/multicluster-appconfig-update.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testComponent1, err := getSampleOamComponent("testdata/hello-component.yaml")
	assert.NoError(err, "failed to read sample data for OAM Component")

	testComponent2, err := getSampleOamComponent("testdata/goodbye-component.yaml")
	assert.NoError(err, "failed to read sample data for OAM Component")

	adminClient := fake.NewFakeClientWithScheme(newScheme(), &testMCAppConfigUpdate, &testComponent1, &testComponent2)

	localClient := fake.NewFakeClientWithScheme(newScheme(), &testMCAppConfig, &testComponent1, &testComponent2)

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCApplicationConfigurationObjects(testMCAppConfigNamespace)

	// Validate the results
	assert.NoError(err)

	// Verify the MultiClusterApplicationConfiguration on the managed cluster is equal to the one on the admin cluster
	mcAppConfig := &clustersv1alpha1.MultiClusterApplicationConfiguration{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testMCAppConfig.Name, Namespace: testMCAppConfig.Namespace}, mcAppConfig)
	assert.NoError(err)
	assert.Equal(mcAppConfigTestUpdatedLabels, mcAppConfig.Labels, "mcappconfig labels did not match")
	assert.Equal("Hello application updated", mcAppConfig.Spec.Template.Metadata.Annotations["description"])
	assert.Equal(2, len(mcAppConfig.Spec.Template.Spec.Components))
	comp0 := mcAppConfig.Spec.Template.Spec.Components[0]
	comp1 := mcAppConfig.Spec.Template.Spec.Components[1]
	assert.Equal("hello-component", comp0.ComponentName)
	assert.Equal("goodbye-component", comp1.ComponentName)
}

// TestDeleteMCAppConfig tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterApplicationConfiguration objects
// WHEN the object exists on the local cluster but not on the admin cluster
// THEN ensure that the MultiClusterApplicationConfiguration is deleted.
func TestDeleteMCAppConfig(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Test data
	testMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testMCAppConfigOrphan, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	testComponent, err := getSampleOamComponent("testdata/hello-component.yaml")
	assert.NoError(err, "failed to read sample data for OAM Component")

	testMCAppConfigOrphan.Name = "orphaned-resource"

	adminClient := fake.NewFakeClientWithScheme(newScheme(), &testMCAppConfig, &testComponent)

	localClient := fake.NewFakeClientWithScheme(newScheme(), &testComponent, &testMCAppConfig, &testMCAppConfigOrphan)

	// Make the request
	s := &Syncer{
		AdminClient:        adminClient,
		LocalClient:        localClient,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCApplicationConfigurationObjects(testMCAppConfigNamespace)

	// Validate the results
	assert.NoError(err)

	// Expect the orphaned MultiClusterApplicationConfiguration object to be deleted from the local cluster
	appConfig := &clustersv1alpha1.MultiClusterApplicationConfiguration{}
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: testMCAppConfigOrphan.Name, Namespace: testMCAppConfigOrphan.Namespace}, appConfig)
	assert.True(errors.IsNotFound(err))
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

	localMCAppConfig, err := getSampleMCAppConfig("testdata/multicluster-appconfig.yaml")
	assert.NoError(err, "failed to read sample data for MultiClusterApplicationConfiguration")

	adminClient := fake.NewFakeClientWithScheme(newScheme(), &adminMCAppConfig)

	localClient := fake.NewFakeClientWithScheme(newScheme(), &localMCAppConfig)

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
	err = s.LocalClient.Get(s.Context, types.NamespacedName{Name: localMCAppConfig.Name, Namespace: localMCAppConfig.Namespace}, delAppConfig)
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
