// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	json "github.com/json-iterator/go"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testMCComponentName = "unit-mccomp"
const testMCComponentNamespace = "unit-mccomp-namespace"

// TestCreateMCComponent tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterSecret objects
// WHEN the a new object exists
// THEN ensure that the MultiClusterSecret is created.
func TestCreateMCComponent(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCSecret := clustersv1alpha1.MultiClusterSecret{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      testMCSecretName,
			Labels:    testLabels,
		},
		Spec: clustersv1alpha1.MultiClusterSecretSpec{
			Placement: clustersv1alpha1.Placement{
				Clusters: []clustersv1alpha1.Cluster{
					{
						Name: testClusterName,
					},
				},
			},
			Template: clustersv1alpha1.SecretTemplate{
				Data:       map[string][]byte{"username": []byte("test-username")},
				StringData: map[string]string{"test": "test-stringdata"},
			},
		},
	}

	// Admin Cluster - expect call to list MultiClusterSecret objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterSecretList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcSecretList *clustersv1alpha1.MultiClusterSecretList, opts ...*client.ListOptions) error {
			mcSecretList.Items = append(mcSecretList.Items, testMCSecret)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterSecret secret from the list returned by the admin cluster
	//                   Return the resource does not exist
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testMCSecretName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: testNamespace, Resource: "MultiClusterSecret"}, testMCSecretName))

	// Managed Cluster - expect call to create a MultiClusterSecret
	mcMock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret, opts ...client.CreateOption) error {
			assert.Equal(testNamespace, mcSecret.Namespace, "mcsecret namespace did not match")
			assert.Equal(testMCSecretName, mcSecret.Name, "mcsecret name did not match")
			assert.Equal(testLabels, mcSecret.Labels, "mcsecret labels did not match")
			assert.Equal(testClusterName, mcSecret.Spec.Placement.Clusters[0].Name, "mcsecret does not contain expected placement")
			assert.Equal([]byte("test-username"), mcSecret.Spec.Template.Data["username"], "mcsecret does not contain expected template data")
			assert.Equal("test-stringdata", mcSecret.Spec.Template.StringData["test"], "mcsecret does not contain expected string data")
			return nil
		})

	// Make the request
	s := &Syncer{
		AdminClient: adminMock,
		MCClient:    mcMock,
		Log:         log,
		ClusterName: testClusterName,
		Context:     context.TODO(),
	}
	err := s.syncMCSecretObjects()

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
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCComponent, err := getSampleMCComponent()
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterComponent")
	}

	// Admin Cluster - expect call to list MultiClusterComponent objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterComponentList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcComponentList *clustersv1alpha1.MultiClusterComponentList, opts ...*client.ListOptions) error {
			mcComponentList.Items = append(mcComponentList.Items, testMCComponent)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterComponent secret from the list returned by the admin cluster
	//                   Return the resource with some values different than what the admin cluster returned
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCComponentNamespace, Name: testMCComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcComponent *clustersv1alpha1.MultiClusterComponent) error {
			mcComponent = &testMCComponent
			mcComponent.Spec.Placement.Clusters[0].Name = "new-name"
			return nil
		})

	// Managed Cluster - expect call to update a MultiClusterComponent
	//                   Verify request had the updated values
	mcMock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcComponent *clustersv1alpha1.MultiClusterComponent, opts ...client.UpdateOption) error {
			assert.Equal(testMCComponentNamespace, mcComponent.Namespace, "mcsecret namespace did not match")
			assert.Equal(testMCComponentName, mcComponent.Name, "mcsecret name did not match")
			assert.Equal(testLabels, mcComponent.Labels, "mcsecret labels did not match")
			assert.Equal("new-name", mcComponent.Spec.Placement.Clusters[0].Name, "mcsecret does not contain expected placement")
			return nil
		})

	// Make the request
	s := &Syncer{
		AdminClient: adminMock,
		MCClient:    mcMock,
		Log:         log,
		ClusterName: testClusterName,
		Context:     context.TODO(),
	}
	err = s.syncMCComponentObjects()

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
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCComponent := clustersv1alpha1.MultiClusterComponent{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testMCComponentNamespace,
			Name:      testMCComponentName,
			Labels:    testLabels,
		},
		Spec: clustersv1alpha1.MultiClusterComponentSpec{
			Placement: clustersv1alpha1.Placement{
				Clusters: []clustersv1alpha1.Cluster{
					{
						Name: "not-my-cluster",
					},
				},
			},
		},
	}

	// Admin Cluster - expect call to list MultiClusterComponent objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterComponentList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcComponentList *clustersv1alpha1.MultiClusterComponentList, opts ...*client.ListOptions) error {
			mcComponentList.Items = append(mcComponentList.Items, testMCComponent)
			return nil
		})

	// Make the request
	s := &Syncer{
		AdminClient: adminMock,
		MCClient:    mcMock,
		Log:         log,
		ClusterName: testClusterName,
		Context:     context.TODO(),
	}
	err := s.syncMCComponentObjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// getSampleMCComponent creates and returns a sample MultiClusterComponent used in tests
func getSampleMCComponent() (clustersv1alpha1.MultiClusterComponent, error) {
	mcComp := clustersv1alpha1.MultiClusterComponent{}
	sampleComponentFile, err := filepath.Abs("testdata/hello-multiclustercomponent.yaml")
	if err != nil {
		return mcComp, err
	}

	rawMcComp, err := clusters.ReadYaml2Json(sampleComponentFile)
	if err != nil {
		return mcComp, err
	}

	err = json.Unmarshal(rawMcComp, &mcComp)
	return mcComp, err
}
