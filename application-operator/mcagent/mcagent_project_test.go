// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcagent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/verrazzano/verrazzano/application-operator/constants"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	k8score "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestSyncer_syncVerrazzanoProjects tests the synchronization method for the following use case.
// GIVEN a request to sync VerrazzanoProject objects
// WHEN the a new object exists
// THEN create namespaces specified in the Project resources in the local cluster
func TestSyncer_syncVerrazzanoProjects(t *testing.T) {
	const existingNS = "existingNS"
	type fields struct {
		vpNamespace string
		nsNames     []string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"Update namespace",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				[]string{existingNS},
			},
			false,
		},
		{
			"Create namespace",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				[]string{"newNS"},
			},
			false,
		},
		{
			"VP not in verrazzano-mc namespace",
			fields{
				"random-namespace",
				[]string{"newNS"},
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
			testProj, err := getTestVerrazzanoProject(tt.fields.vpNamespace, tt.fields.nsNames)
			assert.NoError(err, "failed to get sample project")

			// Admin Cluster - expect call to list MultiClusterSecret objects - return list with one object
			adminMock.EXPECT().
				List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
					list.Items = append(list.Items, testProj)
					return nil
				})

			if tt.fields.vpNamespace == constants.VerrazzanoMultiClusterNamespace {
				// Managed Cluster - expect call to get a namespace
				if tt.fields.nsNames[0] == existingNS {
					localMock.EXPECT().
						Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: tt.fields.nsNames[0]}, gomock.Not(gomock.Nil())).
						DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *k8score.Namespace) error {
							ns.Name = tt.fields.nsNames[0]
							return nil
						})
				} else {
					localMock.EXPECT().
						Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: tt.fields.nsNames[0]}, gomock.Not(gomock.Nil())).
						Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "VerrazzanoProject"}, tt.fields.nsNames[0]))
				}

				// Managed Cluster - expect call to create a namespace if non-existing namespace
				if tt.fields.nsNames[0] != existingNS {
					localMock.EXPECT().
						Create(gomock.Any(), gomock.Any()).
						DoAndReturn(func(ctx context.Context, ns *k8score.Namespace, opts ...client.CreateOption) error {
							assert.Equal(tt.fields.nsNames[0], ns.Name, "namespace name did not match")
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
func getTestVerrazzanoProject(vpNamespace string, nsNames []string) (clustersv1alpha1.VerrazzanoProject, error) {
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
	for _, nsName := range nsNames {
		nsNameBytes := []byte("\n    - " + nsName)
		yamlBytes = append(yamlBytes, nsNameBytes...)
	}
	jsonBytes, err := yaml.YAMLToJSON(yamlBytes)
	if err != nil {
		return proj, err
	}
	fmt.Println(string(jsonBytes))
	err = json.Unmarshal(jsonBytes, &proj)
	return proj, err
}
