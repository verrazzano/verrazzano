// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"context"
	"encoding/json"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	k8score "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const testNamespace1 = "ns1"

func TestCreateNamespace(t *testing.T) {
	doTestCreateNamespace(t, false)
}

func TestUpdateNamespace(t *testing.T) {
	doTestCreateNamespace(t, true)
}

func doTestCreateNamespace(t *testing.T, existingNS bool) {
	assert := asserts.New(t)
	log := ctrl.Log.WithName("test")

	// Managed cluster mocks
	localMocker := gomock.NewController(t)
	localMock := mocks.NewMockClient(localMocker)

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

	// Managed Cluster - expect call to get a namespace
	if existingNS {
		localMock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: testNamespace1}, gomock.Not(gomock.Nil())).
			DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *k8score.Namespace) error {
				ns.Name = testNamespace1
				return nil
			})
	} else {
		localMock.EXPECT().
			Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: testNamespace1}, gomock.Not(gomock.Nil())).
			Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "VerrazzanoProject"}, testNamespace1))
	}

	// Managed Cluster - expect call to create a namespace if non-existing namespace
	if !existingNS {
		localMock.EXPECT().
			Create(gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, ns *k8score.Namespace, opts ...client.CreateOption) error {
				assert.Equal(testNamespace1, ns.Name, "namespace name did not match")
				return nil
			})
	}

	// Make the request
	s := &Syncer{
		AdminClient:        adminMock,
		LocalClient:        localMock,
		Log:                log,
		ManagedClusterName: testClusterName,
		Context:            context.TODO(),
	}
	err = s.syncVerrazzanoProjects()

	// Validate the results
	adminMocker.Finish()
	localMocker.Finish()
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
