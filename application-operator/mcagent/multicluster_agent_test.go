// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"testing"

	"github.com/verrazzano/verrazzano/application-operator/constants"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testClusterName = "cluster1"
const testNamespace = constants.VerrazzanoSystemNamespace
const testMCSecretName = "test-mcsecret"
const testPlacement = "cluster1"

var testLabels = map[string]string{"label1": "test1"}

// TestCreateMCSecret tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterSecret objects
// WHEN the a new object exists
// THEN ensure that the MultiClusterSecret is created.
func TestCreateMCSecret(t *testing.T) {
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
						Name: testPlacement,
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
			assert.Equal(testPlacement, mcSecret.Spec.Placement.Clusters[0].Name, "mcsecret does not contain expected placement")
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

// TestUpdateMCSecret tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterSecret objects
// WHEN the a object exists
// THEN ensure that the MultiClusterSecret is updated.
func TestUpdateMCSecret(t *testing.T) {
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
						Name: testPlacement,
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
	//                   Return the resource with some values different than what the admin cluster returned
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testMCSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcSecret *clustersv1alpha1.MultiClusterSecret) error {
			mcSecret = &testMCSecret
			mcSecret.Spec.Placement.Clusters[0].Name = "new-name"
			mcSecret.Spec.Template.Data["username"] = []byte("test-username-new")
			mcSecret.Spec.Template.StringData["test"] = "test-stringdata-new"
			return nil
		})

	// Managed Cluster - expect call to update a MultiClusterSecret
	//                   Verify request had the updated values
	mcMock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret, opts ...client.UpdateOption) error {
			assert.Equal(testNamespace, mcSecret.Namespace, "mcsecret namespace did not match")
			assert.Equal(testMCSecretName, mcSecret.Name, "mcsecret name did not match")
			assert.Equal(testLabels, mcSecret.Labels, "mcsecret labels did not match")
			assert.Equal("new-name", mcSecret.Spec.Placement.Clusters[0].Name, "mcsecret does not contain expected placement")
			assert.Equal([]byte("test-username-new"), mcSecret.Spec.Template.Data["username"], "mcsecret does not contain expected template data")
			assert.Equal("test-stringdata-new", mcSecret.Spec.Template.StringData["test"], "mcsecret does not contain expected string data")
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
