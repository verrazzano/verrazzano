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
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testClusterName = "managed1"
const testMCSecretNamespace = "unit-mcsecret-namespace"
const testMCSecretName = "unit-mcsecret"

var mcSecretTestLabels = map[string]string{"label1": "test1"}
var mcSecretTestUpdatedLabels = map[string]string{"label1": "test1updated"}

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
	testMCSecret, err := getSampleMCSecret("testdata/multicluster-secret.yaml")
	assert.NoError(err, "failed to get sample secret data")

	// Admin Cluster - expect call to list MultiClusterSecret objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterSecretList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcSecretList *clustersv1alpha1.MultiClusterSecretList, listOptions *client.ListOptions) error {
			assert.Equal(testMCSecretNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcSecretList.Items = append(mcSecretList.Items, testMCSecret)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterSecret secret from the list returned by the admin cluster
	//                   Return the resource does not exist
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCSecretNamespace, Name: testMCSecretName}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "MultiClusterSecret"}, testMCSecretName))

	// Managed Cluster - expect call to create a MultiClusterSecret
	mcMock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret, opts ...client.CreateOption) error {
			assert.Equal(testMCSecretNamespace, mcSecret.Namespace, "mcsecret namespace did not match")
			assert.Equal(testMCSecretName, mcSecret.Name, "mcsecret name did not match")
			assert.Equal(mcSecretTestLabels, mcSecret.Labels, "mcsecret labels did not match")
			assert.Equal(testClusterName, mcSecret.Spec.Placement.Clusters[0].Name, "mcsecret does not contain expected placement")
			assert.Equal([]byte("verrazzano"), mcSecret.Spec.Template.Data["username"], "mcsecret does not contain expected template data")
			assert.Equal("test-stringdata", mcSecret.Spec.Template.StringData["test"], "mcsecret does not contain expected string data")
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterSecret objects - return same list as admin cluster
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterSecretList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcSecretList *clustersv1alpha1.MultiClusterSecretList, listOptions *client.ListOptions) error {
			assert.Equal(testMCSecretNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcSecretList.Items = append(mcSecretList.Items, testMCSecret)
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
	err = s.syncMCSecretObjects(testMCSecretNamespace)

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
	testMCSecret, err := getSampleMCSecret("testdata/multicluster-secret.yaml")
	assert.NoError(err, "failed to get sample secret data")
	testMCSecretUpdate, err := getSampleMCSecret("testdata/multicluster-secret-update.yaml")
	assert.NoError(err, "failed to get sample secret data")

	// Admin Cluster - expect call to list MultiClusterSecret objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterSecretList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcSecretList *clustersv1alpha1.MultiClusterSecretList, listOptions *client.ListOptions) error {
			assert.Equal(testMCSecretNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcSecretList.Items = append(mcSecretList.Items, testMCSecretUpdate)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterSecret secret from the list returned by the admin cluster
	//                   Return the resource with some values different than what the admin cluster returned
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCSecretNamespace, Name: testMCSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcSecret *clustersv1alpha1.MultiClusterSecret) error {
			testMCSecret.DeepCopyInto(mcSecret)
			return nil
		})

	// Managed Cluster - expect call to update a MultiClusterSecret
	//                   Verify request had the updated values
	mcMock.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, mcSecret *clustersv1alpha1.MultiClusterSecret, opts ...client.UpdateOption) error {
			assert.Equal(testMCSecretNamespace, mcSecret.Namespace, "mcsecret namespace did not match")
			assert.Equal(testMCSecretName, mcSecret.Name, "mcsecret name did not match")
			assert.Equal(mcSecretTestUpdatedLabels, mcSecret.Labels, "mcsecret labels did not match")
			assert.Equal("test-stringdata2", mcSecret.Spec.Template.StringData["test"], "mcsecret does not contain expected string data")
			assert.Equal([]byte("test"), mcSecret.Spec.Template.Data["username"], "mcsecret does not contain expected data")
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterSecret objects - return same list as admin cluster
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterSecretList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcSecretList *clustersv1alpha1.MultiClusterSecretList, listOptions *client.ListOptions) error {
			assert.Equal(testMCSecretNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcSecretList.Items = append(mcSecretList.Items, testMCSecret)
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
	err = s.syncMCSecretObjects(testMCSecretNamespace)

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestMCSecretPlacement tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterSecret objects
// WHEN the a object exists that is not targeted for the cluster
// THEN ensure that the MultiClusterSecret is not created or updated
func TestMCSecretPlacement(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCSecret, err := getSampleMCSecret("testdata/multicluster-secret.yaml")
	assert.NoError(err, "failed to get sample secret data")
	testMCSecret.Spec.Placement.Clusters[0].Name = "not-my-cluster"

	// Admin Cluster - expect call to list MultiClusterSecret objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterSecretList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcSecretList *clustersv1alpha1.MultiClusterSecretList, listOptions *client.ListOptions) error {
			assert.Equal(testMCSecretNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcSecretList.Items = append(mcSecretList.Items, testMCSecret)
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterSecret objects - return same list as admin cluster
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterSecretList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcSecretList *clustersv1alpha1.MultiClusterSecretList, listOptions *client.ListOptions) error {
			assert.Equal(testMCSecretNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcSecretList.Items = append(mcSecretList.Items, testMCSecret)
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
	err = s.syncMCSecretObjects(testMCSecretNamespace)

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// TestDeleteMCSecret tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterSecret objects
// WHEN the object exists on the local cluster but not on the admin cluster
// THEN ensure that the MultiClusterSecret is deleted.
func TestDeleteMCSecret(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testMCSecret, err := getSampleMCSecret("testdata/multicluster-secret.yaml")
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterSecret")
	}
	testMCSecretOrphan, err := getSampleMCSecret("testdata/multicluster-secret.yaml")
	if err != nil {
		assert.NoError(err, "failed to read sample data for MultiClusterSecret")
	}
	testMCSecretOrphan.Name = "orphaned-resource"

	// Admin Cluster - expect call to list MultiClusterSecret objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterSecretList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcSecretList *clustersv1alpha1.MultiClusterSecretList, listOptions *client.ListOptions) error {
			assert.Equal(testMCSecretNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcSecretList.Items = append(mcSecretList.Items, testMCSecret)
			return nil
		})

	// Managed Cluster - expect call to get a MultiClusterSecret from the list returned by the admin cluster
	//                   Return the resource
	mcMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testMCSecretNamespace, Name: testMCSecretName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, mcSecret *clustersv1alpha1.MultiClusterSecret) error {
			testMCSecret.DeepCopyInto(mcSecret)
			return nil
		})

	// Managed Cluster - expect call to list MultiClusterSecret objects - return list including an orphaned object
	mcMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.MultiClusterSecretList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, mcSecretList *clustersv1alpha1.MultiClusterSecretList, listOptions *client.ListOptions) error {
			assert.Equal(testMCSecretNamespace, listOptions.Namespace, "list request did not have expected namespace")
			mcSecretList.Items = append(mcSecretList.Items, testMCSecret)
			mcSecretList.Items = append(mcSecretList.Items, testMCSecretOrphan)
			return nil
		})

	// Managed Cluster - expect a call to delete a MultiClusterSecret object
	mcMock.EXPECT().
		Delete(gomock.Any(), gomock.Eq(&testMCSecretOrphan), gomock.Any()).
		Return(nil)

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		LocalClient:        mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncMCSecretObjects(testMCSecretNamespace)

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// getSampleMCSecret creates and returns a sample MultiClusterSecret used in tests
func getSampleMCSecret(filePath string) (clustersv1alpha1.MultiClusterSecret, error) {
	mcSecret := clustersv1alpha1.MultiClusterSecret{}
	sampleSecretFile, err := filepath.Abs(filePath)
	if err != nil {
		return mcSecret, err
	}

	rawMcSecret, err := clusterstest.ReadYaml2Json(sampleSecretFile)
	if err != nil {
		return mcSecret, err
	}

	err = json.Unmarshal(rawMcSecret, &mcSecret)
	return mcSecret, err
}
