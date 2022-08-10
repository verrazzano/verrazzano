// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
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
	"github.com/verrazzano/verrazzano/application-operator/constants"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testLabels = map[string]string{"label1": "test1", "label2": "test2"}
var testAnnotations = map[string]string{"annot1": "test1", "annot2": "test2"}

var testNamespace1 = clustersv1alpha1.NamespaceTemplate{
	Metadata: metav1.ObjectMeta{
		Name:        "newNS1",
		Labels:      testLabels,
		Annotations: testAnnotations,
	},
}

var testNamespace2 = clustersv1alpha1.NamespaceTemplate{
	Metadata: metav1.ObjectMeta{
		Name:        "newNS2",
		Labels:      testLabels,
		Annotations: testAnnotations,
	},
}

var testNamespace3 = clustersv1alpha1.NamespaceTemplate{
	Metadata: metav1.ObjectMeta{
		Name:        "newNS3",
		Labels:      testLabels,
		Annotations: testAnnotations,
	},
}

var testNamespace4 = clustersv1alpha1.NamespaceTemplate{
	Metadata: metav1.ObjectMeta{
		Name:   "newNS4",
		Labels: testLabels,
	},
}

// TestSyncer_syncVerrazzanoProjects tests the synchronization method for the following use case.
// GIVEN a request to sync VerrazzanoProject objects
// WHEN the a new object exists
// THEN ensure that the VerrazzanoProject is created.
func TestSyncer_syncVerrazzanoProjects(t *testing.T) {
	const existingVP = "existingVP"
	newNamespaces := []clustersv1alpha1.NamespaceTemplate{testNamespace1, testNamespace2}

	type fields struct {
		vpNamespace string
		vpName      string
		nsList      []clustersv1alpha1.NamespaceTemplate
		clusters    []clustersv1alpha1.Cluster
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
				[]clustersv1alpha1.Cluster{{Name: testClusterName}},
			},
			false,
		},
		{
			"Create VP",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				"newVP",
				newNamespaces,
				[]clustersv1alpha1.Cluster{{Name: testClusterName}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := asserts.New(t)
			log := zap.S().With("test")

			// Managed cluster mocks
			localMocker := gomock.NewController(t)
			localMock := mocks.NewMockClient(localMocker)

			// Admin cluster mocks
			adminMocker := gomock.NewController(t)
			adminMock := mocks.NewMockClient(adminMocker)

			// Test data
			testProj, err := getTestVerrazzanoProject(tt.fields.vpNamespace, tt.fields.vpName, tt.fields.nsList, tt.fields.clusters)
			assert.NoError(err, "failed to get sample project")

			// Admin Cluster - expect call to list VerrazzanoProject objects - return list with one object
			adminMock.EXPECT().
				List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
				DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, listOptions *client.ListOptions) error {
					assert.Equal(constants.VerrazzanoMultiClusterNamespace, listOptions.Namespace)
					list.Items = append(list.Items, testProj)
					return nil
				})

			if tt.fields.vpName == existingVP {
				// Managed Cluster - expect call to get VerrazzanoProject
				localMock.EXPECT().
					Get(gomock.Any(), types.NamespacedName{Namespace: tt.fields.vpNamespace, Name: tt.fields.vpName}, gomock.Not(gomock.Nil())).
					DoAndReturn(func(ctx context.Context, name types.NamespacedName, vp *clustersv1alpha1.VerrazzanoProject) error {
						vp.Namespace = tt.fields.vpNamespace
						vp.Name = tt.fields.vpName
						vp.Spec.Template.Namespaces = []clustersv1alpha1.NamespaceTemplate{testNamespace1, testNamespace2, testNamespace3}
						vp.Spec.Placement.Clusters = []clustersv1alpha1.Cluster{{Name: testClusterName}}
						return nil
					})

				// Managed Cluster - expect call to update a VerrazzanoProject
				localMock.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, vp *clustersv1alpha1.VerrazzanoProject, opts ...client.UpdateOption) error {
						assert.Equal(tt.fields.vpNamespace, vp.Namespace, "VerrazzanoProject namespace did not match")
						assert.Equal(tt.fields.vpName, vp.Name, "VerrazzanoProject name did not match")
						assert.ElementsMatch(tt.fields.nsList, vp.Spec.Template.Namespaces)
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
						assert.ElementsMatch(tt.fields.nsList, vp.Spec.Template.Namespaces)
						return nil
					})
			}

			// Managed Cluster - expect call to list VerrazzanoProject objects on the local cluster
			localMock.EXPECT().
				List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
				DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, listOptions *client.ListOptions) error {
					assert.Equal(constants.VerrazzanoMultiClusterNamespace, listOptions.Namespace)
					list.Items = append(list.Items, testProj)
					return nil
				})

			// Managed cluster - expect call to list Namespace objects - return list defined for this test run
			localMock.EXPECT().
				List(gomock.Any(), &corev1.NamespaceList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
				DoAndReturn(func(ctx context.Context, list *corev1.NamespaceList, listOptions *client.ListOptions) error {
					for _, namespace := range tt.fields.nsList {
						list.Items = append(list.Items, corev1.Namespace{
							ObjectMeta: namespace.Metadata,
							Spec:       namespace.Spec,
						})
					}
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
			assert.Equal(len(tt.fields.nsList), len(s.ProjectNamespaces), "number of expected namespaces did not match")
			for _, namespace := range tt.fields.nsList {
				assert.True(vzstring.SliceContainsString(s.ProjectNamespaces, namespace.Metadata.Name), "expected namespace not being watched")
			}
		})
	}
}

// TestDeleteVerrazzanoProject tests the synchronization method for the following use case.
// GIVEN a request to sync VerrazzanoProject objects
// WHEN the object exists on the local cluster but not on the admin cluster
// THEN ensure that the VerrazzanoProject is deleted.
func TestDeleteVerrazzanoProject(t *testing.T) {
	type fields struct {
		vpNamespace string
		vpName      string
		nsList      []clustersv1alpha1.NamespaceTemplate
		clusters    []clustersv1alpha1.Cluster
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			"Orphaned VP",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				"TestVP",
				[]clustersv1alpha1.NamespaceTemplate{testNamespace1},
				[]clustersv1alpha1.Cluster{{Name: testClusterName}},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := asserts.New(t)
			log := zap.S().With("test")

			// Managed cluster mocks
			localMocker := gomock.NewController(t)
			localMock := mocks.NewMockClient(localMocker)

			// Admin cluster mocks
			adminMocker := gomock.NewController(t)
			adminMock := mocks.NewMockClient(adminMocker)

			// Test data
			testProj, err := getTestVerrazzanoProject(tt.fields.vpNamespace, tt.fields.vpName, tt.fields.nsList, tt.fields.clusters)
			assert.NoError(err, "failed to get sample project")
			testProjOrphan, err := getTestVerrazzanoProject(tt.fields.vpNamespace, tt.fields.vpName+"-orphan", tt.fields.nsList, tt.fields.clusters)
			assert.NoError(err, "failed to get sample project")

			// Admin Cluster - expect call to list VerrazzanoProject objects - return list with one object
			adminMock.EXPECT().
				List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
				DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, listOptions *client.ListOptions) error {
					assert.Equal(constants.VerrazzanoMultiClusterNamespace, listOptions.Namespace)
					list.Items = append(list.Items, testProj)
					return nil
				})

			// Managed Cluster - expect call to get VerrazzanoProject
			localMock.EXPECT().
				Get(gomock.Any(), types.NamespacedName{Namespace: tt.fields.vpNamespace, Name: tt.fields.vpName}, gomock.Not(gomock.Nil())).
				DoAndReturn(func(ctx context.Context, name types.NamespacedName, vp *clustersv1alpha1.VerrazzanoProject) error {
					vp.Namespace = tt.fields.vpNamespace
					vp.Name = tt.fields.vpName
					vp.Spec.Template.Namespaces = tt.fields.nsList
					vp.Spec.Placement.Clusters = tt.fields.clusters
					vp.Labels = testProj.Labels
					vp.Annotations = testProj.Annotations
					return nil
				})

			// Managed Cluster - expect call to list VerrazzanoProject objects on the local cluster, return an object that
			// does not exist on the admin cluster
			localMock.EXPECT().
				List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
				DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
					list.Items = append(list.Items, testProj)
					list.Items = append(list.Items, testProjOrphan)
					return nil
				})

			// Managed cluster - expect call to list Namespace objects - return list defined for this test run
			localMock.EXPECT().
				List(gomock.Any(), &corev1.NamespaceList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
				DoAndReturn(func(ctx context.Context, list *corev1.NamespaceList, listOptions *client.ListOptions) error {
					for _, namespace := range tt.fields.nsList {
						list.Items = append(list.Items, corev1.Namespace{
							ObjectMeta: namespace.Metadata,
							Spec:       namespace.Spec,
						})
					}
					return nil
				})

			// Managed Cluster - expect a call to delete a VerrazzanoProject object
			localMock.EXPECT().
				Delete(gomock.Any(), gomock.Eq(&testProjOrphan), gomock.Any()).
				Return(nil)

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
			assert.Equal(len(tt.fields.nsList), len(s.ProjectNamespaces), "number of expected namespaces did not match")
			for _, namespace := range tt.fields.nsList {
				assert.True(vzstring.SliceContainsString(s.ProjectNamespaces, namespace.Metadata.Name), "expected namespace not being watched")
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
		nsList      []clustersv1alpha1.NamespaceTemplate
		clusters    []clustersv1alpha1.Cluster
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
				[]clustersv1alpha1.NamespaceTemplate{testNamespace1, testNamespace2},
				[]clustersv1alpha1.Cluster{{Name: testClusterName}},
			},
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				"newVP",
				[]clustersv1alpha1.NamespaceTemplate{testNamespace3, testNamespace4},
				[]clustersv1alpha1.Cluster{{Name: testClusterName}},
			},
			4,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := asserts.New(t)
			log := zap.S().With("test")

			// Managed cluster mocks
			localMocker := gomock.NewController(t)
			localMock := mocks.NewMockClient(localMocker)

			// Admin cluster mocks
			adminMocker := gomock.NewController(t)
			adminMock := mocks.NewMockClient(adminMocker)

			// Test data
			testProj1, err := getTestVerrazzanoProject(tt.vp1Fields.vpNamespace, tt.vp1Fields.vpName, tt.vp1Fields.nsList, tt.vp1Fields.clusters)
			assert.NoError(err, "failed to get sample project")
			testProj2, err := getTestVerrazzanoProject(tt.vp2Fields.vpNamespace, tt.vp2Fields.vpName, tt.vp2Fields.nsList, tt.vp2Fields.clusters)
			assert.NoError(err, "failed to get sample project")

			// Admin Cluster - expect call to list VerrazzanoProject objects - return list with two objects
			adminMock.EXPECT().
				List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
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
						assert.ElementsMatch(tt.vp1Fields.nsList, vp.Spec.Template.Namespaces)
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
						assert.ElementsMatch(tt.vp2Fields.nsList, vp.Spec.Template.Namespaces)
						return nil
					})

				// Managed Cluster - expect call to list VerrazzanoProject objects on the local cluster
				localMock.EXPECT().
					List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
					DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
						list.Items = append(list.Items, testProj1)
						list.Items = append(list.Items, testProj2)
						return nil
					})

				// Managed cluster - expect call to list Namespace objects - return list defined for this test run
				localMock.EXPECT().
					List(gomock.Any(), &corev1.NamespaceList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
					DoAndReturn(func(ctx context.Context, list *corev1.NamespaceList, listOptions *client.ListOptions) error {
						for _, namespace := range tt.vp1Fields.nsList {
							list.Items = append(list.Items, corev1.Namespace{
								ObjectMeta: namespace.Metadata,
								Spec:       namespace.Spec,
							})
						}
						for _, namespace := range tt.vp2Fields.nsList {
							list.Items = append(list.Items, corev1.Namespace{
								ObjectMeta: namespace.Metadata,
								Spec:       namespace.Spec,
							})
						}
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
			for _, namespace := range tt.vp1Fields.nsList {
				assert.True(vzstring.SliceContainsString(s.ProjectNamespaces, namespace.Metadata.Name), "expected namespace not being watched")
			}
			for _, namespace := range tt.vp2Fields.nsList {
				assert.True(vzstring.SliceContainsString(s.ProjectNamespaces, namespace.Metadata.Name), "expected namespace not being watched")
			}
		})
	}
}

// TestRemovePlacementVerrazzanoProject tests the synchronization method for the following use case.
// GIVEN a request to sync VerrazzanoProject objects
// WHEN the object exists on the local cluster but is no longer placed on the local cluster
// THEN ensure that the VerrazzanoProject is deleted.
func TestRemovePlacementVerrazzanoProject(t *testing.T) {
	assert := asserts.New(t)
	log := zap.S().With("test")
	vpName := "test"
	vpNamespace := constants.VerrazzanoMultiClusterNamespace
	nsList := []clustersv1alpha1.NamespaceTemplate{testNamespace1, testNamespace2}
	clusters := []clustersv1alpha1.Cluster{{Name: testClusterName}}
	clustersUpdated := []clustersv1alpha1.Cluster{{Name: constants.DefaultClusterName}}

	// Managed cluster mocks
	localMocker := gomock.NewController(t)
	localMock := mocks.NewMockClient(localMocker)

	// Admin cluster mocks
	adminMocker := gomock.NewController(t)
	adminMock := mocks.NewMockClient(adminMocker)

	// Test data
	testProj, err := getTestVerrazzanoProject(vpNamespace, vpName, nsList, clusters)
	assert.NoError(err, "failed to get sample project")
	testProjUpdated, err := getTestVerrazzanoProject(vpNamespace, vpName, nsList, clustersUpdated)
	assert.NoError(err, "failed to get sample project")

	// Admin Cluster - expect call to list VerrazzanoProject objects - return list with one object
	adminMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
		DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, listOptions *client.ListOptions) error {
			assert.Equal(constants.VerrazzanoMultiClusterNamespace, listOptions.Namespace)
			list.Items = append(list.Items, testProjUpdated)
			return nil
		})

	// Managed Cluster - expect call to get VerrazzanoProject
	localMock.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: vpNamespace, Name: vpName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vp *clustersv1alpha1.VerrazzanoProject) error {
			testProj.DeepCopyInto(vp)
			return nil
		})

	// Managed cluster - expect call to list Namespace objects - return list defined for this test run
	localMock.EXPECT().
		List(gomock.Any(), &corev1.NamespaceList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
		DoAndReturn(func(ctx context.Context, list *corev1.NamespaceList, listOptions *client.ListOptions) error {
			for _, namespace := range nsList {
				list.Items = append(list.Items, corev1.Namespace{
					ObjectMeta: namespace.Metadata,
					Spec:       namespace.Spec,
				})
			}
			return nil
		})

	// Managed Cluster - expect a call to delete a VerrazzanoProject object
	localMock.EXPECT().
		Delete(gomock.Any(), gomock.Eq(&testProj), gomock.Any()).
		Return(nil)

	// Managed Cluster - expect call to list VerrazzanoProject objects on the local cluster, return an empty list
	localMock.EXPECT().
		List(gomock.Any(), &clustersv1alpha1.VerrazzanoProjectList{}, gomock.AssignableToTypeOf(&client.ListOptions{})).
		DoAndReturn(func(ctx context.Context, list *clustersv1alpha1.VerrazzanoProjectList, opts ...*client.ListOptions) error {
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
	assert.NoError(err)
}

// getTestVerrazzanoProject creates and returns VerrazzanoProject used in tests
func getTestVerrazzanoProject(vpNamespace string, vpName string, nsNames []clustersv1alpha1.NamespaceTemplate, clusters []clustersv1alpha1.Cluster) (clustersv1alpha1.VerrazzanoProject, error) {
	proj := clustersv1alpha1.VerrazzanoProject{}
	templateFile, err := filepath.Abs("testdata/verrazzanoproject.yaml")
	if err != nil {
		return proj, err
	}

	// Convert template file to VerrazzanoProject struct
	rawMcComp, err := clusterstest.ReadYaml2Json(templateFile)
	if err != nil {
		return proj, err
	}
	err = json.Unmarshal(rawMcComp, &proj)

	// Populate the content
	proj.Namespace = vpNamespace
	proj.Name = vpName
	proj.Spec.Template.Namespaces = nsNames
	proj.Spec.Placement.Clusters = clusters
	return proj, err
}
