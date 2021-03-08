// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzanoproject

import (
	"context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var testLabels = map[string]string{"label1": "test1", "label2": "test2"}

var existingNS = clustersv1alpha1.NamespaceTemplate{
	Metadata: metav1.ObjectMeta{
		Name:   "existingNS",
		Labels: testLabels,
	},
}

// Omit labels on this namespace to handle test case of starting with an empty label
var newNS = clustersv1alpha1.NamespaceTemplate{
	Metadata: metav1.ObjectMeta{
		Name: "newNS",
	},
}

// TestReconcilerSetupWithManager test the creation of the Reconciler.
// GIVEN a controller implementation
// WHEN the controller is created
// THEN verify no error is returned
func TestReconcilerSetupWithManager(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var mgr *mocks.MockManager
	var cli *mocks.MockClient
	var scheme *runtime.Scheme
	var reconciler Reconciler
	var err error

	mocker = gomock.NewController(t)
	mgr = mocks.NewMockManager(mocker)
	cli = mocks.NewMockClient(mocker)
	scheme = runtime.NewScheme()
	clustersv1alpha1.AddToScheme(scheme)
	reconciler = Reconciler{Client: cli, Scheme: scheme}
	mgr.EXPECT().GetConfig().Return(&rest.Config{})
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(log.NullLogger{})
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

// TestReconcileVerrazzanoProject tests reconciling a VerrazzanoProject.
// GIVEN a VerrazzanoProject resource is created
// WHEN the controller Reconcile function is called
// THEN namespaces are created
func TestReconcileVerrazzanoProject(t *testing.T) {
	const existingVP = "existingVP"

	type fields struct {
		vpNamespace string
		vpName      string
		nsList      []clustersv1alpha1.NamespaceTemplate
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
				existingVP,
				[]clustersv1alpha1.NamespaceTemplate{existingNS},
			},
			false,
		},
		{
			"Create namespace",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				existingVP,
				[]clustersv1alpha1.NamespaceTemplate{newNS},
			},
			false,
		},
		{
			fmt.Sprintf("VP not in %s namespace", constants.VerrazzanoMultiClusterNamespace),
			fields{
				"random-namespace",
				existingVP,
				[]clustersv1alpha1.NamespaceTemplate{newNS},
			},
			false,
		},
		{
			"VP not found",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				"not-found-vp",
				[]clustersv1alpha1.NamespaceTemplate{existingNS},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := asserts.New(t)

			mocker := gomock.NewController(t)
			mockClient := mocks.NewMockClient(mocker)

			// expect call to get a verrazzanoproject
			if tt.fields.vpName == existingVP {
				mockClient.EXPECT().
					Get(gomock.Any(), types.NamespacedName{Namespace: tt.fields.vpNamespace, Name: tt.fields.vpName}, gomock.Not(gomock.Nil())).
					DoAndReturn(func(ctx context.Context, name types.NamespacedName, vp *clustersv1alpha1.VerrazzanoProject) error {
						vp.Namespace = tt.fields.vpNamespace
						vp.Name = tt.fields.vpName
						vp.Spec.Template.Namespaces = tt.fields.nsList
						return nil
					})

				if tt.fields.vpNamespace == constants.VerrazzanoMultiClusterNamespace {
					if tt.fields.nsList[0].Metadata.Name == existingNS.Metadata.Name {
						// expect call to get a namespace
						mockClient.EXPECT().
							Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: tt.fields.nsList[0].Metadata.Name}, gomock.Not(gomock.Nil())).
							DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
								return nil
							})

						// expect call to update the namespace
						mockClient.EXPECT().
							Update(gomock.Any(), gomock.Any()).
							DoAndReturn(func(ctx context.Context, namespace *corev1.Namespace, opts ...client.UpdateOption) error {
								assert.Equal(tt.fields.nsList[0].Metadata.Name, namespace.Name, "namespace name did not match")
								_, labelExists := namespace.Labels[constants.LabelVerrazzanoManaged]
								assert.True(labelExists, fmt.Sprintf("the label %s does not exist", constants.LabelVerrazzanoManaged))
								_, labelExists = namespace.Labels[constants.LabelIstioInjection]
								assert.True(labelExists, fmt.Sprintf("the label %s does not exist", constants.LabelIstioInjection))
								return nil
							})
					} else {
						// expect call to get a namespace that returns namespace not found
						mockClient.EXPECT().
							Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: tt.fields.nsList[0].Metadata.Name}, gomock.Not(gomock.Nil())).
							Return(errors.NewNotFound(schema.GroupResource{Group: "", Resource: "Namespace"}, tt.fields.nsList[0].Metadata.Name))

						// expect call to create a namespace
						mockClient.EXPECT().
							Create(gomock.Any(), gomock.Any()).
							DoAndReturn(func(ctx context.Context, ns *corev1.Namespace) error {
								assert.Equal(tt.fields.nsList[0].Metadata.Name, ns.Name, "namespace name did not match")
								return nil
							})
					}
				}
			} else {
				mockClient.EXPECT().
					Get(gomock.Any(), types.NamespacedName{Namespace: tt.fields.vpNamespace, Name: tt.fields.vpName}, gomock.Not(gomock.Nil())).
					Return(errors.NewNotFound(schema.GroupResource{Group: "clusters.verrazzano.io", Resource: "VerrazzanoProject"}, tt.fields.vpName))
			}

			// Make the request
			request := clusterstest.NewRequest(tt.fields.vpNamespace, tt.fields.vpName)
			reconciler := newVerrazzanoProjectReconciler(mockClient)
			_, err := reconciler.Reconcile(request)

			mocker.Finish()

			if (err != nil) != tt.wantErr {
				t.Errorf("syncVerrazzanoProjects() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// newVerrazzanoProjectReconciler creates a new reconciler for testing
// c - The K8s client to inject into the reconciler
func newVerrazzanoProjectReconciler(c client.Client) Reconciler {
	return Reconciler{
		Client: c,
		Log:    ctrl.Log.WithName("test"),
		Scheme: clusters.NewScheme(),
	}
}
