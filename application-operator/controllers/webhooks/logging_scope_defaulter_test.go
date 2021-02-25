// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/clusters"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"testing"
)

// NotFoundError indicates an error caused by a StatusNotFound status
type NotFoundError struct {
}

var emptyEsDetails = clusters.ElasticsearchDetails{}

func (s NotFoundError) Error() string {
	return "StatusNotFound"
}

func (s NotFoundError) Status() metav1.Status {
	return metav1.Status{
		Status: metav1.StatusFailure,
		Code:   http.StatusNotFound,
		Reason: metav1.StatusReasonNotFound,
	}
}

// TestLoggingScopeDefaulter tests adding a default LoggingScope to an appconfig
// GIVEN a AppConfigDefaulter and an appconfig
//  WHEN Default is called with an appconfig
//  THEN Default should add a default LoggingScope to the appconfig
func TestLoggingScopeDefaulter_Default(t *testing.T) {
	var cli *mocks.MockClient
	var mocker *gomock.Controller

	scopeName := "default-hello-app-logging-scope"
	namespacedName := types.NamespacedName{Name: scopeName, Namespace: "default"}

	// WHEN the appconfig has one component with no scopes and the default scope exists
	// THEN Default should add the default LoggingScope to the component of the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			scope.Name = "default-hello-app-logging-scope"
			return nil
		})
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf.yaml", map[string]string{"hello-component": scopeName}, false)
	mocker.Finish()

	// WHEN the appconfig has one component with no scopes and the default scope does not exist
	// THEN Default should create the default logging scope and add it to the component of the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// First expect it to check for a managed cluster secret
	doExpectGetManagedClusterSecretNotFound(cli)

	// Expect get existing logging scope (non-existent)
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			return NotFoundError{}
		})

	// Expect create logging scope
	cli.EXPECT().Create(gomock.Eq(context.TODO()), gomock.Eq(CreateDefaultLoggingScope(namespacedName, emptyEsDetails)), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return nil
		})
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf.yaml", map[string]string{"hello-component": scopeName}, false)
	mocker.Finish()

	// WHEN the appconfig has one component with a logging scope and the default scope does not exist
	// THEN Default should leave the existing logging scope on the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// Expect get default logging scope and since it exists, expect no other calls
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			return NotFoundError{}
		})
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_withScope.yaml", map[string]string{"hello-component": "logging-scope"}, false)
	mocker.Finish()

	// WHEN the appconfig has one component with a logging scope and the default scope exists
	// THEN Default should delete the default logging scope and leave the existing logging scope on the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// Expect get default logging scope
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			scope.Name = "default-hello-app-logging-scope"
			scope.Labels = map[string]string{"default.logging.scope": "true"}
			return nil
		})

	// Expect default logging scope delete
	cli.EXPECT().Delete(gomock.Eq(context.TODO()), gomock.Eq(CreateDefaultLoggingScope(namespacedName, emptyEsDetails)), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
			return nil
		})
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_withScope.yaml", map[string]string{"hello-component": "logging-scope"}, false)
	mocker.Finish()

	// WHEN the appconfig has multiple components (one with a logging scope) and the default scope exists
	// THEN Default should leave the existing logging scope for the one and set the default logging scope
	//   on the others in the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// Expect get default logging scope and since it exists, expect no other calls
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			scope.Name = "default-hello-app-logging-scope"
			scope.Labels = map[string]string{"default.logging.scope": "true"}
			return nil
		})
	scopeNames := map[string]string{"hello-component1": scopeName, "hello-component2": scopeName, "hello-component3": scopeName, "hello-component4": "logging-scope"}
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_multiComponents1.yaml", scopeNames, false)
	mocker.Finish()

	// WHEN the appconfig has multiple components (one with a logging scope) and the default scope does not exist
	// THEN Default should create the default logging scope instance, leave the existing logging scope for the one and
	//   set the default logging scope on the others in the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// Expect it to get managed cluster secret
	doExpectGetManagedClusterSecretNotFound(cli)

	// Expect get default logging scope
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			return NotFoundError{}
		})

	// Expect create default logging scope
	cli.EXPECT().Create(gomock.Eq(context.TODO()), gomock.Eq(CreateDefaultLoggingScope(namespacedName, emptyEsDetails)), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return nil
		})
	scopeNames = map[string]string{"hello-component1": scopeName, "hello-component2": scopeName, "hello-component3": scopeName, "hello-component4": "logging-scope"}
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_multiComponents1.yaml", scopeNames, false)
	mocker.Finish()

	// WHEN the appconfig has multiple components with no logging scopes and the default scope does not exist
	// THEN Default should create the default logging scope instance and set the default logging scope
	//   on all of the components in the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// Expect it to get managed cluster secret
	doExpectGetManagedClusterSecretNotFound(cli)

	// Expect get default logging scope
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			return NotFoundError{}
		})

	// Expect create default logging scope
	cli.EXPECT().Create(gomock.Eq(context.TODO()), gomock.Eq(CreateDefaultLoggingScope(namespacedName, emptyEsDetails)), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return nil
		})
	scopeNames = map[string]string{"hello-component1": scopeName, "hello-component2": scopeName, "hello-component3": scopeName, "hello-component4": scopeName}
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_multiComponents2.yaml", scopeNames, false)
	mocker.Finish()

	// WHEN the appconfig has multiple components with no logging scopes and the default scope does not exist
	//      and Default is called with dryRun true
	// THEN Default should set the default logging scope on all of the components in the appconfig but
	//      the default logging scope instance should not be created
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	scopeNames = map[string]string{"hello-component1": scopeName, "hello-component2": scopeName, "hello-component3": scopeName, "hello-component4": scopeName}
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_multiComponents2.yaml", scopeNames, true)
	mocker.Finish()
}

func TestLoggingScopeDefaulter_Cleanup(t *testing.T) {
	var cli *mocks.MockClient
	var mocker *gomock.Controller

	scopeName := "default-hello-app-logging-scope"
	namespacedName := types.NamespacedName{Name: scopeName, Namespace: "default"}

	// WHEN the appconfig has one component with no logging scope and the default scope exists
	// THEN Cleanup should delete the default logging scope
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			scope.Name = "default-hello-app-logging-scope"
			scope.Labels = map[string]string{"default.logging.scope": "true"}
			return nil
		})
	cli.EXPECT().Delete(gomock.Eq(context.TODO()), gomock.Eq(CreateDefaultLoggingScope(namespacedName, emptyEsDetails)), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
			return nil
		})
	testLoggingScopeDefaulterCleanup(t, cli, "hello-conf.yaml", false)
	mocker.Finish()

	// WHEN the appconfig has one component with no logging scope and the default scope exists
	//      and Cleanup is called with dryRun true
	// THEN Cleanup should delete the default logging scope
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	testLoggingScopeDefaulterCleanup(t, cli, "hello-conf.yaml", true)
	mocker.Finish()
}

// TestCreateDefaultLoggingScope tests the behavior of CreateDefaultLoggingScope
// GIVEN a managed cluster with Elasticsearch details specified
// WHEN CreateDefaultLoggingScope is called with those ElasticsearchDetails
// THEN it should create a default LoggingScope with the provided details
func TestCreateDefaultLoggingScope(t *testing.T) {
	scopeName := "default-hello-app-logging-scope"
	namespacedName := types.NamespacedName{Name: scopeName, Namespace: "default"}

	// WHEN CreateDefaultLoggingScope is called with empty ES details
	// it should use the default values in the created logging scope
	scope := CreateDefaultLoggingScope(namespacedName, emptyEsDetails)
	asserts.Equal(t, defaultElasticSearchURL, scope.Spec.ElasticSearchURL)
	asserts.Equal(t, defaultSecretName, scope.Spec.SecretName)

	// WHEN CreateDefaultLoggingScope is called with non-empty ES details
	// it should use the values provided instead of the defaults
	esDetails := clusters.ElasticsearchDetails{URL: "http://some-other-es:9999", SecretName: "some-other-secret"}
	scope = CreateDefaultLoggingScope(namespacedName, esDetails)
	asserts.Equal(t, esDetails.URL, scope.Spec.ElasticSearchURL)
	asserts.Equal(t, esDetails.SecretName, scope.Spec.SecretName)
}

// TestLoggingScopeDefaulter_DefaultManagedCluster tests adding a default LoggingScope to an
// appconfig in a managed cluster (in this case the default logging scope should refer to the admin
// cluster's Elasticsearch endpoint and secret)
// GIVEN a AppConfigDefaulter and an appconfig in a managed cluster,
// WHEN Default is called with an appconfig
// THEN it should add a default LoggingScope to the appconfig which refers to the admin cluster ES
func TestLoggingScopeDefaulter_DefaultOnManagedCluster(t *testing.T) {
	var cli *mocks.MockClient
	var mocker *gomock.Controller

	scopeName := "default-hello-app-logging-scope"
	namespacedName := types.NamespacedName{Name: scopeName, Namespace: "default"}

	// The appconfig has one component with no scopes and the default scope does not exist
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// Expect it to get managed cluster secret and return a managed cluster secret (all other tests
	// assumed no managed cluster secret found)
	esDetails := clusters.ElasticsearchDetails{URL: "http://some-es-host:9999", SecretName: constants.ElasticsearchSecretName}
	mcSecret := v1.Secret{Data: map[string][]byte{
		constants.ClusterNameData:         []byte("managed-cluster1"),
		constants.ElasticsearchURLData:    []byte(esDetails.URL)}}
	doExpectGetManagedClusterSecretFound(cli, mcSecret)

	// Expect get default logging scope
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			return NotFoundError{}
		})

	// Expect create logging scope with the esDetails specified in the managedClusterSecret (instead of the defaults)
	cli.EXPECT().Create(gomock.Eq(context.TODO()), gomock.Eq(CreateDefaultLoggingScope(namespacedName, esDetails)), gomock.Not(gomock.Nil())).
		Return(nil)

	testLoggingScopeDefaulterDefault(t, cli, "hello-conf.yaml", map[string]string{"hello-component": scopeName}, false)
	mocker.Finish()
}

func testLoggingScopeDefaulterDefault(t *testing.T, cli client.Client, configPath string, scopeNames map[string]string, dryRun bool) {
	assert := asserts.New(t)
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, configPath)}
	decoder := decoder()
	appConfig := &oamv1.ApplicationConfiguration{}
	err := decoder.Decode(req, appConfig)
	if err != nil {
		t.Fatalf("Error in decoder.Decode %v", err)
	}
	defaulter := &LoggingScopeDefaulter{Client: cli}
	err = defaulter.Default(appConfig, dryRun)
	if err != nil {
		t.Fatalf("Error in defaulter.Default %v", err)
	}
	for _, component := range appConfig.Spec.Components {
		foundLoggingScope := false
		for _, scope := range component.Scopes {
			if scope.ScopeReference.APIVersion == apiVersion && scope.ScopeReference.Kind == v1alpha1.LoggingScopeKind && scope.ScopeReference.Name == scopeNames[component.ComponentName] {
				foundLoggingScope = true
			}
		}
		assert.True(foundLoggingScope)
	}
}

func testLoggingScopeDefaulterCleanup(t *testing.T, cli client.Client, configPath string, dryRun bool) {
	req := admission.Request{}
	req.Object = runtime.RawExtension{Raw: readYaml2Json(t, configPath)}
	decoder := decoder()
	appConfig := &oamv1.ApplicationConfiguration{}
	err := decoder.Decode(req, appConfig)
	if err != nil {
		t.Fatalf("Error in decoder.Decode %v", err)
	}
	defaulter := &LoggingScopeDefaulter{Client: cli}
	err = defaulter.Cleanup(appConfig, dryRun)
	if err != nil {
		t.Fatalf("Error in defaulter.Default %v", err)
	}
}

func doExpectGetManagedClusterSecretNotFound(cli *mocks.MockClient) {
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(clusters.MCRegistrationSecretFullName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *v1.Secret) error {
			return NotFoundError{}
		})
}

func doExpectGetManagedClusterSecretFound(cli *mocks.MockClient, managedClusterSecret v1.Secret) {
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(clusters.MCRegistrationSecretFullName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, secret *v1.Secret) error {
			secret.Data = managedClusterSecret.Data
			return nil
		})
}
