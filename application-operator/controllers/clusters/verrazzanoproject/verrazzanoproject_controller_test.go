// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzanoproject

import (
	"context"
	"fmt"
	netv1 "k8s.io/api/networking/v1"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	clusterstest "github.com/verrazzano/verrazzano/application-operator/controllers/clusters/test"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
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

const finalizer = "project.verrazzano.io"

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

var ns1 = clustersv1alpha1.NamespaceTemplate{
	Metadata: metav1.ObjectMeta{
		Name: "ns1",
	},
}
var ns2 = clustersv1alpha1.NamespaceTemplate{
	Metadata: metav1.ObjectMeta{
		Name: "ns2",
	},
}

var ns1Netpol = clustersv1alpha1.NetworkPolicyTemplate{
	Metadata: metav1.ObjectMeta{
		Name:      "ns1Netpol",
		Namespace: "ns1",
	},
	Spec: netv1.NetworkPolicySpec{
	},
}

var ns2Netpol = clustersv1alpha1.NetworkPolicyTemplate{
	Metadata: metav1.ObjectMeta{
		Name:      "ns2Netpol",
		Namespace: "ns2",
	},
	Spec: netv1.NetworkPolicySpec{
	},
}

var ns2Netpol2 = clustersv1alpha1.NetworkPolicyTemplate{
	Metadata: metav1.ObjectMeta{
		Name:      "ns2Netpol2",
		Namespace: "ns2",
	},
	Spec: netv1.NetworkPolicySpec{
	},
}

// roleBindingMatcher is a gomock Matcher that matches a rbacv1.RoleBinding based on roleref name
type roleBindingMatcher struct{ roleRefName string }

func RoleBindingMatcher(roleName string) gomock.Matcher {
	return &roleBindingMatcher{roleName}
}

func (r *roleBindingMatcher) Matches(x interface{}) bool {
	if rb, ok := x.(*rbacv1.RoleBinding); ok {
		if r.roleRefName == rb.RoleRef.Name {
			return true
		}
	}
	return false
}

func (r *roleBindingMatcher) String() string {
	return "rolebinding roleref name does not match expected name: " + r.roleRefName
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

	adminSubjects := []rbacv1.Subject{
		{Kind: "Group", Name: "project-admin-test-group"},
		{Kind: "User", Name: "project-admin-test-user"},
	}

	monitorSubjects := []rbacv1.Subject{
		{Kind: "Group", Name: "project-monitor-test-group"},
		{Kind: "User", Name: "project-monitor-test-user"},
	}

	clusterList := []clustersv1alpha1.Cluster{
		{Name: clusterstest.UnitTestClusterName},
	}

	type fields struct {
		vpNamespace     string
		vpName          string
		nsList          []clustersv1alpha1.NamespaceTemplate
		adminSubjects   []rbacv1.Subject
		monitorSubjects []rbacv1.Subject
		placementList   []clustersv1alpha1.Cluster
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
				adminSubjects,
				monitorSubjects,
				clusterList,
			},
			false,
		},
		{
			"Create namespace",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				existingVP,
				[]clustersv1alpha1.NamespaceTemplate{newNS},
				nil,
				nil,
				clusterList,
			},
			false,
		},
		{
			"Create project admin rolebindings",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				existingVP,
				[]clustersv1alpha1.NamespaceTemplate{newNS},
				adminSubjects,
				nil,
				clusterList,
			},
			false,
		},
		{
			"Create project monitor rolebindings",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				existingVP,
				[]clustersv1alpha1.NamespaceTemplate{newNS},
				nil,
				monitorSubjects,
				clusterList,
			},
			false,
		},
		{
			fmt.Sprintf("VP not in %s namespace", constants.VerrazzanoMultiClusterNamespace),
			fields{
				"random-namespace",
				existingVP,
				[]clustersv1alpha1.NamespaceTemplate{newNS},
				nil,
				nil,
				clusterList,
			},
			false,
		},
		{
			"VP not found",
			fields{
				constants.VerrazzanoMultiClusterNamespace,
				"not-found-vp",
				[]clustersv1alpha1.NamespaceTemplate{existingNS},
				nil,
				nil,
				clusterList,
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := asserts.New(t)

			mocker := gomock.NewController(t)
			mockClient := mocks.NewMockClient(mocker)
			mockStatusWriter := mocks.NewMockStatusWriter(mocker)

			// expect call to get a verrazzanoproject
			if tt.fields.vpName == existingVP {
				mockClient.EXPECT().
					Get(gomock.Any(), types.NamespacedName{Namespace: tt.fields.vpNamespace, Name: tt.fields.vpName}, gomock.Not(gomock.Nil())).
					DoAndReturn(func(ctx context.Context, name types.NamespacedName, vp *clustersv1alpha1.VerrazzanoProject) error {
						vp.Namespace = tt.fields.vpNamespace
						vp.Name = tt.fields.vpName
						vp.ObjectMeta.Finalizers = []string{finalizer}
						vp.Spec.Template.Namespaces = tt.fields.nsList
						vp.Spec.Template.Security.ProjectAdminSubjects = tt.fields.adminSubjects
						vp.Spec.Template.Security.ProjectMonitorSubjects = tt.fields.monitorSubjects
						vp.Spec.Placement.Clusters = tt.fields.placementList
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

						if len(tt.fields.adminSubjects) > 0 {
							mockUpdatedRoleBindingExpectations(assert, mockClient, tt.fields.nsList[0].Metadata.Name, projectAdminRole, projectAdminK8sRole, tt.fields.adminSubjects)
						}
						if len(tt.fields.monitorSubjects) > 0 {
							mockUpdatedRoleBindingExpectations(assert, mockClient, tt.fields.nsList[0].Metadata.Name, projectMonitorRole, projectMonitorK8sRole, tt.fields.monitorSubjects)
						}

					} else { // not an existing namespace
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

						if len(tt.fields.adminSubjects) > 0 {
							mockNewRoleBindingExpectations(assert, mockClient, tt.fields.nsList[0].Metadata.Name, projectAdminRole, projectAdminK8sRole, tt.fields.adminSubjects)
						}
						if len(tt.fields.monitorSubjects) > 0 {
							mockNewRoleBindingExpectations(assert, mockClient, tt.fields.nsList[0].Metadata.Name, projectMonitorRole, projectMonitorK8sRole, tt.fields.monitorSubjects)
						}
					}
				} // END VerrazzanoProject is in the expected Multi cluster namespace

				// expect call to list network policies
				mockClient.EXPECT().
					List(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
						// return no resources
						return nil
					})

				// status update should be to "succeeded" in both existing and new namespace
				doExpectStatusUpdateSucceeded(mockClient, mockStatusWriter, assert)

			} else { // The VerrazzanoProject is not an existing one i.e. not existingVP
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

// TestNetworkPolicies tests the creation of network policies
// GIVEN a VerrazzanoProject resource is create with specified network policies
// WHEN the controller Reconcile function is called
// THEN the network policies are created
func TestNetworkPolicies(t *testing.T) {
	const vpName = "testNetwokPolicies"

	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	mockStatusWriter := mocks.NewMockStatusWriter(mocker)

	// Expect call to get the project
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: vpName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vp *clustersv1alpha1.VerrazzanoProject) error {
			vp.Namespace = constants.VerrazzanoMultiClusterNamespace
			vp.Name = vpName
			vp.ObjectMeta.Finalizers = []string{finalizer}
			vp.Spec.Template.Namespaces = []clustersv1alpha1.NamespaceTemplate{ns1}
			vp.Spec.Template.NetworkPolicies = []clustersv1alpha1.NetworkPolicyTemplate{ns1Netpol}
			vp.Spec.Placement.Clusters = []clustersv1alpha1.Cluster{{Name: clusterstest.UnitTestClusterName}}
			return nil
		})

	// expect call to get a namespace
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "", Name: ns1.Metadata.Name}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, ns *corev1.Namespace) error {
			return nil
		})

	// expect call to update the namespace
	mockClient.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, namespace *corev1.Namespace, opts ...client.UpdateOption) error {
			return nil
		})

	// expect call to get a network policy
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: ns1Netpol.Metadata.Namespace, Name: ns1Netpol.Metadata.Name}, gomock.Not(gomock.Nil())).
		Return(errors.NewNotFound(schema.GroupResource{Group: ns1.Metadata.Namespace, Resource: "NetworkPolicy"}, ns1.Metadata.Name))

	// Expect call to create the network policies in the namespace
	mockClient.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, policy *netv1.NetworkPolicy, opts ...client.CreateOption) error {
			return nil
		})

	// Expect call to get the network policies in the namespace
	mockClient.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *netv1.NetworkPolicyList, opts ...client.ListOption) error {
			list.Items = []netv1.NetworkPolicy{{
				ObjectMeta: metav1.ObjectMeta{Namespace: "ns1", Name: "ns1Netpol"},
				Spec:       netv1.NetworkPolicySpec{},
			},
				{
					ObjectMeta: metav1.ObjectMeta{Namespace: "ns2", Name: "ns2Netpol"},
					Spec:       netv1.NetworkPolicySpec{},
				}}
			return nil
		})

	// Expect call to delete network policy ns2 since it is not defined in the project
	mockClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, policy *netv1.NetworkPolicy, opts ...client.DeleteOption) error {
			assert.Equal("ns2Netpol", policy.Name, "Incorrect NetworkPolicy being deleted")
			return nil
		})

	// the status update should be to success status/conditions on the VerrazzanoProject
	// status update should be to "succeeded" in both existing and new namespace
	doExpectStatusUpdateSucceeded(mockClient, mockStatusWriter, assert)

	// Make the request
	request := clusterstest.NewRequest(constants.VerrazzanoMultiClusterNamespace, vpName)
	reconciler := newVerrazzanoProjectReconciler(mockClient)
	_, err := reconciler.Reconcile(request)
	assert.NoError(err)

	mocker.Finish()
}

// TestDeleteVerrazzanoProject tests deleting a VerrazzanoProject
// GIVEN a VerrazzanoProject resource is deleted
// WHEN the controller Reconcile function is called
// THEN the resource is successfully cleaned up
func TestDeleteVerrazzanoProject(t *testing.T) {
	vpName := "testDelete"
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: vpName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vp *clustersv1alpha1.VerrazzanoProject) error {
			vp.Namespace = constants.VerrazzanoMultiClusterNamespace
			vp.Name = vpName
			vp.Spec.Template.Namespaces = []clustersv1alpha1.NamespaceTemplate{existingNS}
			vp.Spec.Placement.Clusters = []clustersv1alpha1.Cluster{{Name: clusterstest.UnitTestClusterName}}
			vp.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			return nil
		})

	// Make the request
	request := clusterstest.NewRequest(constants.VerrazzanoMultiClusterNamespace, vpName)
	reconciler := newVerrazzanoProjectReconciler(mockClient)
	_, err := reconciler.Reconcile(request)
	assert.NoError(err)

	mocker.Finish()
}

// TestDeleteVerrazzanoProjectFinalizer tests deleting a VerrazzanoProject with finalizer
// GIVEN a VerrazzanoProject resource is deleted
// WHEN the controller Reconcile function is called
// THEN the resource is successfully cleaned up, including network policies
func TestDeleteVerrazzanoProjectFinalizer(t *testing.T) {
	vpName := "testDelete"
	assert := asserts.New(t)

	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)

	// Expect call to get the project
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: constants.VerrazzanoMultiClusterNamespace, Name: vpName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, vp *clustersv1alpha1.VerrazzanoProject) error {
			vp.Namespace = constants.VerrazzanoMultiClusterNamespace
			vp.Name = vpName
			vp.ObjectMeta.Finalizers = []string{finalizer}
			vp.Spec.Template.Namespaces = []clustersv1alpha1.NamespaceTemplate{existingNS}
			vp.Spec.Placement.Clusters = []clustersv1alpha1.Cluster{{Name: clusterstest.UnitTestClusterName}}
			vp.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			return nil
		})

	// Expect call to get the network policies in the namespace
	mockClient.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *netv1.NetworkPolicyList, opts ...client.ListOption) error {
			list.Items = []netv1.NetworkPolicy{{
				ObjectMeta: metav1.ObjectMeta{Namespace: "existingNS", Name: "ns1"},
				Spec:       netv1.NetworkPolicySpec{},
			}}
			return nil
		})

	// Expect call to delete network policies in the namespace
	mockClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)

	// the status update should be to success status/conditions on the VerrazzanoProject
	mockClient.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.VerrazzanoProject{})).
		DoAndReturn(func(ctx context.Context, vp *clustersv1alpha1.VerrazzanoProject) error {
			assert.NotContainsf(vp.Finalizers, finalizer, "finalizer should be cleared")
			return nil
		})

	// Make the request
	request := clusterstest.NewRequest(constants.VerrazzanoMultiClusterNamespace, vpName)
	reconciler := newVerrazzanoProjectReconciler(mockClient)
	_, err := reconciler.Reconcile(request)
	assert.NoError(err)

	mocker.Finish()
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

// mockNewRoleBindingExpectations mocks the expections for project rolebindings when the rolebindings do not already exist
func mockNewRoleBindingExpectations(assert *asserts.Assertions, mockClient *mocks.MockClient, namespace, role, k8sRole string, subjects []rbacv1.Subject) {
	// expect a call to fetch a rolebinding for the specified role and return NotFound
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: role}, gomock.AssignableToTypeOf(&rbacv1.RoleBinding{})).
		Return(errors.NewNotFound(schema.GroupResource{}, ""))

	// expect a call to create a rolebinding for the subjects to the specified role
	mockClient.EXPECT().
		Create(gomock.Any(), RoleBindingMatcher(role)).
		DoAndReturn(func(ctx context.Context, rb *rbacv1.RoleBinding) error {
			assert.Equal(subjects, rb.Subjects)
			return nil
		})

	// expect a call to fetch a rolebinding for the specified k8s role and return NotFound
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: k8sRole}, gomock.AssignableToTypeOf(&rbacv1.RoleBinding{})).
		Return(errors.NewNotFound(schema.GroupResource{}, ""))

	// expect a call to create a rolebinding for the subjects to the specified k8s role
	mockClient.EXPECT().
		Create(gomock.Any(), RoleBindingMatcher(k8sRole)).
		DoAndReturn(func(ctx context.Context, rb *rbacv1.RoleBinding) error {
			assert.Equal(subjects, rb.Subjects)
			return nil
		})
}

// mockUpdatedRoleBindingExpectations mocks the expections for project rolebindings when the rolebindings already exist
func mockUpdatedRoleBindingExpectations(assert *asserts.Assertions, mockClient *mocks.MockClient, namespace, role, k8sRole string, subjects []rbacv1.Subject) {
	// expect a call to fetch a rolebinding for the specified role and return an existing rolebinding
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: role}, gomock.AssignableToTypeOf(&rbacv1.RoleBinding{})).
		DoAndReturn(func(ctx context.Context, ns types.NamespacedName, roleBinding *rbacv1.RoleBinding) error {
			// simulate subjects and roleref being set our of band, we should reset them
			roleBinding.RoleRef.Name = "changed"
			roleBinding.Subjects = []rbacv1.Subject{
				{Kind: "Group", Name: "I-manually-set-this"},
			}
			return nil
		})

	// expect a call to update the rolebinding
	mockClient.EXPECT().
		Update(gomock.Any(), RoleBindingMatcher(role)).
		DoAndReturn(func(ctx context.Context, rb *rbacv1.RoleBinding) error {
			assert.Equal(role, rb.RoleRef.Name)
			assert.Equal(subjects, rb.Subjects)
			return nil
		})

	// expect a call to fetch a rolebinding for the specified k8s role and return an existing rolebinding
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: k8sRole}, gomock.AssignableToTypeOf(&rbacv1.RoleBinding{})).
		DoAndReturn(func(ctx context.Context, ns types.NamespacedName, roleBinding *rbacv1.RoleBinding) error {
			// simulate subjects and roleref being set our of band, we should reset them
			roleBinding.RoleRef.Name = "changed"
			roleBinding.Subjects = []rbacv1.Subject{
				{Kind: "Group", Name: "I-manually-set-this"},
			}
			return nil
		})

	// expect a call to update the rolebinding
	mockClient.EXPECT().
		Update(gomock.Any(), RoleBindingMatcher(k8sRole)).
		DoAndReturn(func(ctx context.Context, rb *rbacv1.RoleBinding) error {
			assert.Equal(k8sRole, rb.RoleRef.Name)
			assert.Equal(subjects, rb.Subjects)
			return nil
		})
}

// doExpectStatusUpdateFailed expects a call to update status of
// VerrazzanoProject to failure
func doExpectStatusUpdateFailed(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to fetch the MCRegistration secret to get the cluster name for status update
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to update the status of the VerrazzanoProject
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to failure status/conditions on the VerrazzanoProject
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.VerrazzanoProject{})).
		DoAndReturn(func(ctx context.Context, vp *clustersv1alpha1.VerrazzanoProject) error {
			clusterstest.AssertMultiClusterResourceStatus(assert, vp.Status, clustersv1alpha1.Failed, clustersv1alpha1.DeployFailed, corev1.ConditionTrue)
			return nil
		})
}

// doExpectStatusUpdateSucceeded expects a call to update status of
// VerrazzanoProject to success
func doExpectStatusUpdateSucceeded(cli *mocks.MockClient, mockStatusWriter *mocks.MockStatusWriter, assert *asserts.Assertions) {
	// expect a call to fetch the MCRegistration secret to get the cluster name for status update
	clusterstest.DoExpectGetMCRegistrationSecret(cli)

	// expect a call to update the status of the VerrazzanoProject
	cli.EXPECT().Status().Return(mockStatusWriter)

	// the status update should be to success status/conditions on the VerrazzanoProject
	mockStatusWriter.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&clustersv1alpha1.VerrazzanoProject{})).
		DoAndReturn(func(ctx context.Context, vp *clustersv1alpha1.VerrazzanoProject) error {
			clusterstest.AssertMultiClusterResourceStatus(assert, vp.Status, clustersv1alpha1.Succeeded, clustersv1alpha1.DeployComplete, corev1.ConditionTrue)
			return nil
		})
}
