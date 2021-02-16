// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	k8score "k8s.io/api/core/v1"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestCreateNamespace tests the synchronization method for the following use case.
// GIVEN a request to sync MultiClusterSecret objects
// WHEN the a new object exists
// THEN ensure that the MultiClusterSecret is created.
func TestCreateNamespace(t *testing.T) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	mcMocker := gomock.NewController(t)
	mcMock := mocks.NewMockClient(mcMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testProj, err := getSampleProject("testdata/verrazzanoproject.yaml")
	assert.NoError(err, "failed to get sample project")

	// Admin Cluster - expect call to list MultiClusterSecret objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
			list.Items = append(list.Items, testProj)
			return nil
		})

	// Managed Cluster - expect call to create a namespace
	mcMock.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, ns *k8score.Namespace, opts ...client.CreateOption) error {
			assert.Equal("ns1", ns.Namespace, "namespace name did not match")
			return nil
		})

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		MCClient:           mcMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncVerrazzanoProjects()

	// Validate the results
	adminMocker.Finish()
	mcMocker.Finish()
	assert.NoError(err)
}

// getSampleProject creates and returns a sample VerrazzanoProject used in tests
func getSampleProject(filePath string) (clustersv1alpha1.VerrazzanoProject, error) {
	proj := clustersv1alpha1.VerrazzanoProject{}
	file, err := filepath.Abs(filePath)
	if err != nil {
		return proj, err
	}

	raw, err := clusters.ReadYaml2Json(file)
	if err != nil {
		return proj, err
	}

	err = json.Unmarshal(raw, &proj)
	return proj, err
}
