// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"fmt"
	"testing"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	testKind       = "test-type"
	workloadUID    = "test-workload-uid"
	workloadName   = "test-workload-name"
	testNamespace  = "test-namespace"
	testAPIVersion = "testv1"
	testScopeName  = "test-scope-name"
)

// TestReconcilerSetupWithManager test the creation of the logging scope reconciler.
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
	vzapi.AddToScheme(scheme)
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

// TestLoggingScopeReconcileApply tests reconcile apply functionality
// GIVEN a logging scope which contains a workload which isn't tet associated to the scope
// WHEN reconcile is called with the scope information
// THEN ensure that all of the FLUENTD artifacts are properly created in the system
func TestLoggingScopeReconcileApply(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	mockHandler := mocks.NewMockHandler(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	scheme := runtime.NewScheme()
	reconciler := newTestReconciler(mockClient, scheme, mockHandler)
	testWorkload := createTestWorkload()
	testScope := createTestLoggingScope(true)

	testStatus := vzapi.LoggingScopeStatus{Resources: []vzapi.QualifiedResourceRelation{{
		APIVersion: testAPIVersion, Kind: testKind, Name: workloadName, Namespace: testNamespace,
	}}}
	updatedScope := createTestLoggingScope(true)
	updatedScope.Status = testStatus

	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testScopeName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, scope *vzapi.LoggingScope) error {
			*scope = *testScope
			return nil
		})
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			*workload = *testWorkload
			return nil
		})
	mockClient.EXPECT().Status().Return(mockStatus)

	mockHandler.EXPECT().Apply(context.Background(), toResource(testWorkload), testScope)
	mockStatus.EXPECT().Update(context.Background(), updatedScope).Return(nil)

	result, err := reconciler.Reconcile(createTestLoggingScopeRequest())
	asserts.Equal(t, ctrl.Result{}, result)
	asserts.Nil(t, err)

	mocker.Finish()
}

// TestLoggingScopeReconcileApply tests reconcile remove functionality
// GIVEN a logging scope which doesn't contain a previously associated component
// WHEN reconcile is called with the scope information
// THEN ensure that all of the FLUENTD artifacts are properly cleaned up in the system
func TestLoggingScopeReconcileRemove(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	mockHandler := mocks.NewMockHandler(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	scheme := runtime.NewScheme()
	reconciler := newTestReconciler(mockClient, scheme, mockHandler)
	testScope := createTestLoggingScope(false)

	testStatus := vzapi.LoggingScopeStatus{Resources: []vzapi.QualifiedResourceRelation{{
		APIVersion: testAPIVersion, Kind: testKind, Name: workloadName, Namespace: testNamespace,
	}}}
	testScope.Status = testStatus

	// lookup scope
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testScopeName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, scope *vzapi.LoggingScope) error {
			*scope = *testScope
			return nil
		})

	mockClient.EXPECT().Status().Return(mockStatus)
	mockHandler.EXPECT().Remove(context.Background(), testStatus.Resources[0], testScope)
	mockStatus.EXPECT().Update(context.Background(), testScope)

	result, err := reconciler.Reconcile(createTestLoggingScopeRequest())
	asserts.Equal(t, ctrl.Result{}, result)
	asserts.Nil(t, err)

	mocker.Finish()
}

// TestLoggingScopeReconcileBothApplyAndRemove tests reconcile apply and remove functionality
// GIVEN a logging scope which contains a workload which isn't tet associated to the scope and also doesn't
//       contain a previously associated component
// WHEN reconcile is called with the scope information
// THEN ensure that all of the FLUENTD artifacts are properly created for the associated component and removed
//      for the resource which is no longer associated
func TestLoggingScopeReconcileBothApplyAndRemove(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	mockHandler := mocks.NewMockHandler(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	scheme := runtime.NewScheme()
	reconciler := newTestReconciler(mockClient, scheme, mockHandler)
	testWorkload := createTestWorkload()
	testScope := createTestLoggingScope(true)

	// This status has a resource not associated with a scope workload which should therefore be deleted
	testStatus := vzapi.LoggingScopeStatus{Resources: []vzapi.QualifiedResourceRelation{{
		APIVersion: testAPIVersion, Kind: testKind, Name: "someOtherName", Namespace: testNamespace,
	}}}
	testScope.Status = testStatus
	// this is the status that should be persisted as a result of the call to Reconcile()
	updatedStatus := vzapi.LoggingScopeStatus{Resources: []vzapi.QualifiedResourceRelation{{
		APIVersion: testAPIVersion, Kind: testKind, Name: workloadName, Namespace: testNamespace,
	}}}
	updatedScope := createTestLoggingScope(true)
	updatedScope.Status = updatedStatus

	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testScopeName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, scope *vzapi.LoggingScope) error {
			*scope = *testScope
			return nil
		})
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *unstructured.Unstructured) error {
			*workload = *testWorkload
			return nil
		})
	mockClient.EXPECT().Status().Return(mockStatus)
	mockHandler.EXPECT().Apply(context.Background(), toResource(testWorkload), testScope)
	mockHandler.EXPECT().
		Remove(context.Background(), testStatus.Resources[0], testScope).Return(true, nil)
	// the scope status should be updated without the workload resource
	mockStatus.EXPECT().Update(context.Background(), updatedScope)

	result, err := reconciler.Reconcile(createTestLoggingScopeRequest())
	asserts.Equal(t, ctrl.Result{}, result)
	asserts.Nil(t, err)

	mocker.Finish()
}

// TestLoggingScopeReconcileApply tests reconcile remove functionality
// GIVEN a logging scope which doesn't contain a previously associated component
// WHEN reconcile is called with the scope information
// THEN ensure that all of the FLUENTD artifacts are properly cleaned up in the system and that the component
//      association is removed from the scope status
func TestLoggingScopeReconcileRemoveAndForget(t *testing.T) {
	mocker := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mocker)
	mockHandler := mocks.NewMockHandler(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)
	scheme := runtime.NewScheme()
	reconciler := newTestReconciler(mockClient, scheme, mockHandler)
	testScope := createTestLoggingScope(false)
	testStatus := vzapi.LoggingScopeStatus{Resources: []vzapi.QualifiedResourceRelation{{
		APIVersion: testAPIVersion, Kind: testKind, Name: workloadName, Namespace: testNamespace,
	}}}
	testScope.Status = testStatus
	updatedScope := createTestLoggingScope(false)

	// lookup scope
	mockClient.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testScopeName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, scope *vzapi.LoggingScope) error {
			*scope = *testScope
			return nil
		})

	mockClient.EXPECT().Status().Return(mockStatus)
	// in this case Remove() returns true to indicate that we should forget about the workload
	mockHandler.EXPECT().
		Remove(context.Background(), testStatus.Resources[0], testScope).Return(true, nil)
	// the scope status should be updated without the workload resource
	mockStatus.EXPECT().Update(context.Background(), updatedScope)

	result, err := reconciler.Reconcile(createTestLoggingScopeRequest())
	asserts.Equal(t, ctrl.Result{}, result)
	asserts.Nil(t, err)

	mocker.Finish()
}

// createTestLoggingScopeRequest creates a test logging scope reconcile request
func createTestLoggingScopeRequest() ctrl.Request {
	return ctrl.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: testScopeName}}
}

// newTestReconciler creates a new test reconciler
func newTestReconciler(client client.Client, scheme *runtime.Scheme, handler Handler) *Reconciler {
	reconciler := NewReconciler(client, ctrl.Log.WithName("controllers").WithName("LoggingScope"), scheme)
	reconciler.Handlers[fmt.Sprintf("%s/%s", testAPIVersion, testKind)] = handler

	return reconciler
}

// createTestWorkload creates a test workload
func createTestWorkload() *unstructured.Unstructured {
	workload := unstructured.Unstructured{}
	workload.SetKind(testKind)
	workload.SetNamespace(testNamespace)
	workload.SetName(workloadName)
	workload.SetUID(workloadUID)
	workload.SetAPIVersion(testAPIVersion)

	return &workload
}

// createTestLoggingScope creates a test logging scope
func createTestLoggingScope(includeWorkload bool) *vzapi.LoggingScope {
	scope := vzapi.LoggingScope{}
	scope.TypeMeta = k8smeta.TypeMeta{
		APIVersion: vzapi.GroupVersion.Identifier(),
		Kind:       vzapi.LoggingScopeKind}
	scope.ObjectMeta = k8smeta.ObjectMeta{
		Namespace: testNamespace,
		Name:      testScopeName}
	scope.Spec.ElasticSearchURL = testESURL
	scope.Spec.SecretName = testESSecret
	scope.Spec.FluentdImage = "fluentd/image/location"
	if includeWorkload {
		scope.Spec.WorkloadReferences = []oamrt.TypedReference{{
			APIVersion: oamcore.SchemeGroupVersion.Identifier(),
			Kind:       testKind,
			Name:       workloadName}}
	}

	return &scope
}

func updateLoggingScope(scope *vzapi.LoggingScope) {
	scope.Spec.ElasticSearchURL = testESURLUpdate
	scope.Spec.SecretName = testESSecretUpdate
}
