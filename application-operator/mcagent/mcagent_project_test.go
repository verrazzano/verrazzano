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

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
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
				[]string{},
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
							vp.Namespace = tt.fields.vpNamespace
							vp.Name = tt.fields.vpName
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
						Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "VerrazzanoProject"}, tt.fields.vpName))

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

			// Managed Cluster - expect call to list VerrazzanoProject objects on the local cluster
			localMock.EXPECT().
				List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
					list.Items = append(list.Items, testProj)
					return nil
				})

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

			// Validate the namespace list that resulted from processing the VerrazzanoProject objects
			assert.Equal(len(tt.fields.nsNames), len(s.ProjectNamespaces), "number of expected namespaces did not match")
			for _, namespace := range tt.fields.nsNames {
				assert.True(controllers.StringSliceContainsString(s.ProjectNamespaces, namespace), "expected namespace not being watched")
			}
		})
	}
}

// TestVerrazzanoProjectMulti tests the synchronization method for the following use case.
// GIVEN a request to sync multiple VerrazzanoProject objects
// WHEN the a new object exists
// THEN ensure that the list of namespaces to watch is correct
func TestVerrazzanoProjectMulti(t *testing.T) {
	type fields struct {
		vpNamespace string
		vpName      string
		nsNames     []string
	}
	tests := []struct {
		name               string
		vp1Fields          fields
		vp2Fields          fields
		expectedNamespaces int
		wantErr            bool
	}{
		{
			"TwoVP",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				"newVP",
				[]string{"ns1", "ns2"},
			},
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				"newVP",
				[]string{"ns3", "ns4"},
			},
			4,
			false,
		},
		{
			"DuplicateNamespace",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				"newVP",
				[]string{"ns1", "ns2"},
			},
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				"newVP",
				[]string{"ns3", "ns1"},
			},
			3,
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
			testProj1, err := getTestVerrazzanoProject(tt.vp1Fields.vpNamespace, tt.vp1Fields.vpName, tt.vp1Fields.nsNames)
			assert.NoError(err, "failed to get sample project")
			testProj2, err := getTestVerrazzanoProject(tt.vp2Fields.vpNamespace, tt.vp2Fields.vpName, tt.vp2Fields.nsNames)
			assert.NoError(err, "failed to get sample project")

			// Admin Cluster - expect call to list VerrazzanoProject objects - return list with two objects
			adminMock.EXPECT().
				List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
					list.Items = append(list.Items, testProj1)
					list.Items = append(list.Items, testProj2)
					return nil
				})

			if tt.vp1Fields.vpNamespace == constants.VerrazzanoMultiClusterNamespace {
				// Managed Cluster - expect call to get VerrazzanoProject
				localMock.EXPECT().
					Get(gomock.Any(), types.NamespacedName{Namespace: tt.vp1Fields.vpNamespace, Name: tt.vp1Fields.vpName}, gomock.Not(gomock.Nil())).
					Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "VerrazzanoProject"}, tt.vp1Fields.vpName))

				// Managed Cluster - expect call to create a VerrazzanoProject
				localMock.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, vp *clustersv1alpha1.VerrazzanoProject, opts ...client.CreateOption) error {
						assert.Equal(tt.vp1Fields.vpNamespace, vp.Namespace, "VerrazzanoProject namespace did not match")
						assert.Equal(tt.vp1Fields.vpName, vp.Name, "VerrazzanoProject name did not match")
						assert.ElementsMatch(tt.vp1Fields.nsNames, vp.Spec.Namespaces)
						return nil
					})

				// Managed Cluster - expect call to get VerrazzanoProject
				localMock.EXPECT().
					Get(gomock.Any(), types.NamespacedName{Namespace: tt.vp2Fields.vpNamespace, Name: tt.vp2Fields.vpName}, gomock.Not(gomock.Nil())).
					Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "VerrazzanoProject"}, tt.vp2Fields.vpName))

				// Managed Cluster - expect call to create a VerrazzanoProject
				localMock.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, vp *clustersv1alpha1.VerrazzanoProject, opts ...client.CreateOption) error {
						assert.Equal(tt.vp2Fields.vpNamespace, vp.Namespace, "VerrazzanoProject namespace did not match")
						assert.Equal(tt.vp2Fields.vpName, vp.Name, "VerrazzanoProject name did not match")
						assert.ElementsMatch(tt.vp2Fields.nsNames, vp.Spec.Namespaces)
						return nil
					})

				// Managed Cluster - expect call to list VerrazzanoProject objects on the local cluster
				localMock.EXPECT().
					List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.Not(gomock.Nil())).
					DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
						list.Items = append(list.Items, testProj1)
						list.Items = append(list.Items, testProj2)
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

			if (err != nil) != tt.wantErr {
				t.Errorf("syncVerrazzanoProjects() error = %v, wantErr %v", err, tt.wantErr)
			}

			// Validate the namespace list that resulted from processing the VerrazzanoProject objects
			assert.Equal(tt.expectedNamespaces, len(s.ProjectNamespaces), "number of expected namespaces did not match")
			for _, namespace := range tt.vp1Fields.nsNames {
				assert.True(controllers.StringSliceContainsString(s.ProjectNamespaces, namespace), "expected namespace not being watched")
			}
			for _, namespace := range tt.vp2Fields.nsNames {
				assert.True(controllers.StringSliceContainsString(s.ProjectNamespaces, namespace), "expected namespace not being watched")
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
