// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/verrazzano/verrazzano/application-operator/constants"

	"sigs.k8s.io/yaml"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestSyncer_syncVerrazzanoProjects tests the synchronization method for the following use case.
// GIVEN a request to sync VerrazzanoProject objects
// WHEN the a new object exists
// THEN ensure that the VerrazzanoProject is created.
func TestSyncer_syncVerrazzanoProjects(t *testing.T) {
	const existingVP = "existingVP"
	newNamespaces := []string{"newNS1", "newNS2"}
	type fields struct {
		vpNamespace string
		vpName      string
		nsNames     []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"Update VP",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				existingVP,
				newNamespaces,
			},
			false,
		},
		{
			"Create VP",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				"newVP",
				newNamespaces,
			},
			false,
		},
		{
			"VP not in verrazzano-mc namespace",
			fields{
				"random-namespace",
				"vpInRandomNamespace",
				newNamespaces,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := asserts.New(t)
			log := ctrl.Log.WithName("test")

			// Managed cluster mocks
			localMocker := gomock.NewController(t)
			localMock := mocks.NewMockClient(localMocker)

			// Admin cluster mocks
			adminMocker := gomock.NewController(t)
			adminMock := mocks.NewMockClient(adminMocker)

			// Test data
			testProj, err := getTestVerrazzanoProject(tt.fields.vpNamespace, tt.fields.vpName, tt.fields.nsNames)
			assert.NoError(err, "failed to get sample project")

			// Admin Cluster - expect call to list VerrazzanoProject objects - return list with one object
			adminMock.EXPECT().
				List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
					list.Items = append(list.Items, testProj)
					return nil
				})

			if tt.fields.vpNamespace == constants.VerrazzanoMultiClusterNamespace {
				if tt.fields.vpName == existingVP {
					// Managed Cluster - expect call to get VerrazzanoProject
					localMock.EXPECT().
						Get(gomock.Any(), types.NamespacedName{Namespace: tt.fields.vpNamespace, Name: tt.fields.vpName}, gomock.Not(gomock.Nil())).
						DoAndReturn(func(ctx context.Context, name types.NamespacedName, vp *clustersv1alpha1.VerrazzanoProject) error {
							vp.Spec.Namespaces = []string{"existingNS1", "existingNS2", "existingNS3"}
							return nil
						})

					// Managed Cluster - expect call to update a VerrazzanoProject
					localMock.EXPECT().
						Update(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, vp *clustersv1alpha1.VerrazzanoProject, opts ...client.UpdateOption) error {
							assert.Equal(tt.fields.vpNamespace, vp.Namespace, "VerrazzanoProject namespace did not match")
							assert.Equal(tt.fields.vpName, vp.Name, "VerrazzanoProject name did not match")
							assert.ElementsMatch(tt.fields.nsNames, vp.Spec.Namespaces)
							return nil
						})
				} else {
					// Managed Cluster - expect call to get VerrazzanoProject
					localMock.EXPECT().
						Get(gomock.Any(), types.NamespacedName{Namespace: tt.fields.vpNamespace, Name: tt.fields.vpName}, gomock.Not(gomock.Nil())).
						Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "VerrazzanoProject"}, tt.fields.vpName))

					// Managed Cluster - expect call to create a VerrazzanoProject
					localMock.EXPECT().
						Create(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, vp *clustersv1alpha1.VerrazzanoProject, opts ...client.CreateOption) error {
							assert.Equal(tt.fields.vpNamespace, vp.Namespace, "VerrazzanoProject namespace did not match")
							assert.Equal(tt.fields.vpName, vp.Name, "VerrazzanoProject name did not match")
							assert.ElementsMatch(tt.fields.nsNames, vp.Spec.Namespaces)
							return nil
						})
				}
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

			if (err != nil) != tt.wantErr {
				t.Errorf("syncVerrazzanoProjects() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// getTestVerrazzanoProject creates and returns VerrazzanoProject used in tests
func getTestVerrazzanoProject(vpNamespace string, vpName string, nsNames []string) (clustersv1alpha1.VerrazzanoProject, error) {
	proj := clustersv1alpha1.VerrazzanoProject{}
	templateFile, err := filepath.Abs("testdata/verrazzanoproject.yaml")
	if err != nil {
		return proj, err
	}
	templateBytes, err := ioutil.ReadFile(templateFile)
	if err != nil {
		return proj, err
	}
	yamlBytes := bytes.Replace(templateBytes, []byte("VP_NS"), []byte(vpNamespace), -1)
	yamlBytes = bytes.Replace(yamlBytes, []byte("VP_NAME"), []byte(vpName), -1)
	for _, nsName := range nsNames {
		nsNameBytes := []byte("\n    - " + nsName)
		yamlBytes = append(yamlBytes, nsNameBytes...)
	}
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return proj, err
	}
	err = json.Unmarshal(jsonBytes, &proj)
	return proj, err
}
