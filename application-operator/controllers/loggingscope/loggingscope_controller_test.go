// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package loggingscope

import (
	"context"
	"fmt"
	"testing"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
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

// TestFromWorkloadLabels tests the FromWorkloadLabels function.
func TestFromWorkloadLabels(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()

	// GIVEN workload labels
	// WHEN an attempt is made to get the logging scopes from the app component but there are no scopes
	// THEN expect no error and a nil logging scope is returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: "unit-test-app-config"}

	// expect a call to fetch the oam application configuration
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "unit-test-namespace", Name: "unit-test-app-config"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})

	loggingScope, err := FromWorkloadLabels(ctx, cli, "unit-test-namespace", labels)

	mocker.Finish()
	assert.NoError(err)
	assert.Nil(loggingScope)

	// GIVEN workload labels
	// WHEN an attempt is made to get the logging scopes from the app component and there is a logging scope
	// THEN expect no error and a logging scope is returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	loggingScopeName := "unit-test-logging-scope"
	fluentdImage := "unit-test-image:latest"
	esURL := "localhost"
	esSecretName := "unit-test-secret"

	// expect a call to fetch the oam application configuration
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "unit-test-namespace", Name: "unit-test-app-config"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			loggingScope := oamcore.ComponentScope{ScopeReference: oamrt.TypedReference{Kind: vzapi.LoggingScopeKind, Name: loggingScopeName}}
			component.Scopes = []oamcore.ComponentScope{loggingScope}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the logging scope
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "unit-test-namespace", Name: loggingScopeName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, loggingScope *vzapi.LoggingScope) error {
			loggingScope.Spec.FluentdImage = fluentdImage
			loggingScope.Spec.ElasticSearchURL = esURL
			loggingScope.Spec.SecretName = esSecretName
			return nil
		})

	loggingScope, err = FromWorkloadLabels(ctx, cli, "unit-test-namespace", labels)

	mocker.Finish()
	assert.NoError(err)
	assert.NotNil(loggingScope)
	assert.Equal(fluentdImage, loggingScope.Spec.FluentdImage)
	assert.Equal(esURL, loggingScope.Spec.ElasticSearchURL)
	assert.Equal(esSecretName, loggingScope.Spec.SecretName)

	// GIVEN workload labels
	// WHEN an attempt is made to get the logging scopes from the app component and we cannot fetch the logging scope details
	// THEN expect a NotFound error is returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the oam application configuration
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "unit-test-namespace", Name: "unit-test-app-config"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			loggingScope := oamcore.ComponentScope{ScopeReference: oamrt.TypedReference{Kind: vzapi.LoggingScopeKind, Name: loggingScopeName}}
			component.Scopes = []oamcore.ComponentScope{loggingScope}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the logging scope
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "unit-test-namespace", Name: loggingScopeName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, loggingScope *vzapi.LoggingScope) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})

	loggingScope, err = FromWorkloadLabels(ctx, cli, "unit-test-namespace", labels)

	mocker.Finish()
	assert.True(k8serrors.IsNotFound(err))
	assert.Nil(loggingScope)
}

// TestFetchLoggingScopeWithDefaults tests that defaults are correctly applied when
// fetching a logging scope
func TestFetchLoggingScopeWithDefaults(t *testing.T) {
	assert := asserts.New(t)

	// set the logging scope default FLUENTD image for this test and then put it back when we're done
	oldDefaultFluentdImage := DefaultFluentdImage
	defer func() {
		DefaultFluentdImage = oldDefaultFluentdImage
	}()

	DefaultFluentdImage = "default-unit-test-image:latest"

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()

	loggingScopeName := "unit-test-logging-scope"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: "unit-test-app-config"}

	// GIVEN workload labels
	// WHEN an attempt is made to get the logging scopes from the app component and there is a logging scope
	// AND the logging scope has no spec fields populated
	// THEN expect no error and a logging scope with populated defaults is returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the oam application configuration
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "unit-test-namespace", Name: "unit-test-app-config"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			loggingScope := oamcore.ComponentScope{ScopeReference: oamrt.TypedReference{Kind: vzapi.LoggingScopeKind, Name: loggingScopeName}}
			component.Scopes = []oamcore.ComponentScope{loggingScope}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the logging scope
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "unit-test-namespace", Name: loggingScopeName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, loggingScope *vzapi.LoggingScope) error {
			return nil
		})
	// logging scope URL and secret are empty so expect a call to get the cluster secret, return NotFound
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(clusters.MCRegistrationSecretFullName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *v1.Secret) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})

	loggingScope, err := FromWorkloadLabels(ctx, cli, "unit-test-namespace", labels)

	mocker.Finish()
	assert.NoError(err)
	assert.NotNil(loggingScope)
	assert.Equal(DefaultFluentdImage, loggingScope.Spec.FluentdImage)
	assert.Equal(DefaultElasticSearchURL, loggingScope.Spec.ElasticSearchURL)
	assert.Equal(DefaultSecretName, loggingScope.Spec.SecretName)
}

// TestApplyDefaults tests various combinations of applied defaults
func TestApplyDefaults(t *testing.T) {

	// set the logging scope default FLUENTD image for this test and then put it back when we're done
	oldDefaultFluentdImage := DefaultFluentdImage
	defer func() {
		DefaultFluentdImage = oldDefaultFluentdImage
	}()

	DefaultFluentdImage = "default-unit-test-image:latest"

	var mocker *gomock.Controller
	var cli *mocks.MockClient

	// GIVEN a logging scope with no spec fields populated
	// WHEN we apply defaults
	// THEN the logging scope spec fields are populated with all of the default values
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	loggingScope := &vzapi.LoggingScope{}
	expected := &vzapi.LoggingScope{
		Spec: vzapi.LoggingScopeSpec{
			FluentdImage:     DefaultFluentdImage,
			ElasticSearchURL: DefaultElasticSearchURL,
			SecretName:       DefaultSecretName,
		},
	}

	// logging scope URL and secret are empty so expect a call to get the cluster secret, return NotFound
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(clusters.MCRegistrationSecretFullName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *v1.Secret) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})

	assertApplyDefaults(cli, expected, loggingScope, t)
	mocker.Finish()

	// GIVEN a logging scope with just the fluentd image in the spec
	// WHEN we apply defaults
	// THEN the remaining logging scope spec fields are populated with default values
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	fluentdImage := "unit-test-image:1.0"
	loggingScope = &vzapi.LoggingScope{
		Spec: vzapi.LoggingScopeSpec{
			FluentdImage: fluentdImage,
		},
	}
	expected = &vzapi.LoggingScope{
		Spec: vzapi.LoggingScopeSpec{
			FluentdImage:     fluentdImage,
			ElasticSearchURL: DefaultElasticSearchURL,
			SecretName:       DefaultSecretName,
		},
	}

	// logging scope URL and secret are empty so expect a call to get the cluster secret, return NotFound
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(clusters.MCRegistrationSecretFullName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *v1.Secret) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})

	assertApplyDefaults(cli, expected, loggingScope, t)
	mocker.Finish()

	// GIVEN a logging scope with the fluentd image and ES URL in the spec
	// WHEN we apply defaults
	// THEN the remaining logging scope spec fields are populated with default values
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	esURL := "localhost:9200"
	loggingScope = &vzapi.LoggingScope{
		Spec: vzapi.LoggingScopeSpec{
			FluentdImage:     fluentdImage,
			ElasticSearchURL: esURL,
		},
	}
	expected = &vzapi.LoggingScope{
		Spec: vzapi.LoggingScopeSpec{
			FluentdImage:     fluentdImage,
			ElasticSearchURL: esURL,
			SecretName:       DefaultSecretName,
		},
	}
	assertApplyDefaults(cli, expected, loggingScope, t)
	mocker.Finish()

	// GIVEN a logging scope with all spec fields populated
	// WHEN we apply defaults
	// THEN none of the spec fields
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	esSecretName := "sssshhhhhhh"
	loggingScope = &vzapi.LoggingScope{
		Spec: vzapi.LoggingScopeSpec{
			FluentdImage:     fluentdImage,
			ElasticSearchURL: esURL,
			SecretName:       esSecretName,
		},
	}
	expected = &vzapi.LoggingScope{
		Spec: vzapi.LoggingScopeSpec{
			FluentdImage:     fluentdImage,
			ElasticSearchURL: esURL,
			SecretName:       esSecretName,
		},
	}
	assertApplyDefaults(cli, expected, loggingScope, t)
	mocker.Finish()
}

// TestApplyDefaultsForManagedCluster tests applying defaults when the
// logging scope is in a multi-cluster managed cluster
func TestApplyDefaultsForManagedCluster(t *testing.T) {

	// set the logging scope default FLUENTD image for this test and then put it back when we're done
	oldDefaultFluentdImage := DefaultFluentdImage
	defer func() {
		DefaultFluentdImage = oldDefaultFluentdImage
	}()

	DefaultFluentdImage = "default-unit-test-image:latest"

	var mocker *gomock.Controller
	var cli *mocks.MockClient

	// GIVEN a logging scope with no spec fields populated
	// WHEN we apply defaults
	// THEN the logging scope spec fields are populated with all of the default values
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	adminClusterESURL := "http://some-es-host:9999"

	loggingScope := &vzapi.LoggingScope{}
	expected := &vzapi.LoggingScope{
		Spec: vzapi.LoggingScopeSpec{
			FluentdImage:     DefaultFluentdImage,
			ElasticSearchURL: adminClusterESURL,
			SecretName:       constants.ElasticsearchSecretName,
		},
	}

	mcSecret := v1.Secret{Data: map[string][]byte{
		constants.ClusterNameData:      []byte("managed-cluster1"),
		constants.ElasticsearchURLData: []byte(adminClusterESURL)}}

	// logging scope URL and secret are empty so expect a call to get the cluster secret
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(clusters.MCRegistrationSecretFullName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *v1.Secret) error {
			secret.Data = mcSecret.Data
			return nil
		})

	assertApplyDefaults(cli, expected, loggingScope, t)
	mocker.Finish()
}

// assertApplyDefaults applies defaults to the passed in actual logging scope and
// asserts that it is equal to the expected logging scope
func assertApplyDefaults(cli client.Reader, expected, actual *vzapi.LoggingScope, t *testing.T) {
	assert := asserts.New(t)
	applyDefaults(cli, actual)
	assert.Equal(expected, actual)
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
