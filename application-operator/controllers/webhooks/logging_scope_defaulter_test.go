// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"context"
	"net/http"
	"testing"

	v1 "k8s.io/api/core/v1"

	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// NotFoundError indicates an error caused by a StatusNotFound status
type NotFoundError struct {
}

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
	namespaceName := types.NamespacedName{Name: "default", Namespace: ""}

	// WHEN the appconfig is being deleted
	// THEN Default should not add the default LoggingScope to the component of the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect get namespace
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespaceName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, ns *v1.Namespace) error {
			return nil
		})
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf-deleting.yaml", map[string]string{"hello-component": scopeName}, false, false)
	mocker.Finish()

	// WHEN the appconfig has one component with no scopes and the default scope exists
	// THEN Default should add the default LoggingScope to the component of the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect get namespace
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespaceName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, ns *v1.Namespace) error {
			return nil
		})
	// Expect get existing logging scope
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			scope.Name = "default-hello-app-logging-scope"
			return nil
		})
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf.yaml", map[string]string{"hello-component": scopeName}, false, true)
	mocker.Finish()

	// WHEN the appconfig has one component with no scopes and the default scope does not exist
	// THEN Default should create the default logging scope and add it to the component of the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect get namespace
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespaceName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, ns *v1.Namespace) error {
			return nil
		})
	// Expect get existing logging scope (non-existent)
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			return NotFoundError{}
		})
	// Expect create logging scope
	cli.EXPECT().Create(gomock.Eq(context.TODO()), gomock.Eq(createDefaultLoggingScope(namespacedName)), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return nil
		})
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf.yaml", map[string]string{"hello-component": scopeName}, false, true)
	mocker.Finish()

	// WHEN the appconfig has one component with a logging scope and the default scope does not exist
	// THEN Default should leave the existing logging scope on the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect get namespace
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespaceName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, ns *v1.Namespace) error {
			return nil
		})
	// Expect get default logging scope and since it exists, expect no other calls
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			return NotFoundError{}
		})
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_withScope.yaml", map[string]string{"hello-component": "logging-scope"}, false, true)
	mocker.Finish()

	// WHEN the appconfig has one component with a logging scope and the default scope exists
	// THEN Default should delete the default logging scope and leave the existing logging scope on the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect get namespace
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespaceName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, ns *v1.Namespace) error {
			return nil
		})
	// Expect get default logging scope
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			scope.Name = "default-hello-app-logging-scope"
			scope.Labels = map[string]string{"default.logging.scope": "true"}
			return nil
		})

	// Expect default logging scope delete
	cli.EXPECT().Delete(gomock.Eq(context.TODO()), gomock.Eq(createDefaultLoggingScope(namespacedName)), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, obj runtime.Object, opts ...client.DeleteOption) error {
			return nil
		})
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_withScope.yaml", map[string]string{"hello-component": "logging-scope"}, false, true)
	mocker.Finish()

	// WHEN the appconfig has multiple components (one with a logging scope) and the default scope exists
	// THEN Default should leave the existing logging scope for the one and set the default logging scope
	//   on the others in the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect get namespace
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespaceName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, ns *v1.Namespace) error {
			return nil
		})
	// Expect get default logging scope and since it exists, expect no other calls
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			scope.Name = "default-hello-app-logging-scope"
			scope.Labels = map[string]string{"default.logging.scope": "true"}
			return nil
		})
	scopeNames := map[string]string{"hello-component1": scopeName, "hello-component2": scopeName, "hello-component3": scopeName, "hello-component4": "logging-scope"}
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_multiComponents1.yaml", scopeNames, false, true)
	mocker.Finish()

	// WHEN the appconfig has multiple components (one with a logging scope) and the default scope does not exist
	// THEN Default should create the default logging scope instance, leave the existing logging scope for the one and
	//   set the default logging scope on the others in the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect get namespace
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespaceName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, ns *v1.Namespace) error {
			return nil
		})
	// Expect get default logging scope
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			return NotFoundError{}
		})

	// Expect create default logging scope
	cli.EXPECT().Create(gomock.Eq(context.TODO()), gomock.Eq(createDefaultLoggingScope(namespacedName)), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return nil
		})
	scopeNames = map[string]string{"hello-component1": scopeName, "hello-component2": scopeName, "hello-component3": scopeName, "hello-component4": "logging-scope"}
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_multiComponents1.yaml", scopeNames, false, true)
	mocker.Finish()

	// WHEN the appconfig has multiple components with no logging scopes and the default scope does not exist
	// THEN Default should create the default logging scope instance and set the default logging scope
	//   on all of the components in the appconfig
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect get namespace
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespaceName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, ns *v1.Namespace) error {
			return nil
		})
	// Expect get default logging scope
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespacedName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, scope *vzapi.LoggingScope) error {
			return NotFoundError{}
		})

	// Expect create default logging scope
	cli.EXPECT().Create(gomock.Eq(context.TODO()), gomock.Eq(createDefaultLoggingScope(namespacedName)), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, obj runtime.Object, opts ...client.CreateOption) error {
			return nil
		})
	scopeNames = map[string]string{"hello-component1": scopeName, "hello-component2": scopeName, "hello-component3": scopeName, "hello-component4": scopeName}
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_multiComponents2.yaml", scopeNames, false, true)
	mocker.Finish()

	// WHEN the appconfig has multiple components with no logging scopes and the default scope does not exist
	//      and Default is called with dryRun true
	// THEN Default should set the default logging scope on all of the components in the appconfig but
	//      the default logging scope instance should not be created
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect get namespace
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespaceName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, ns *v1.Namespace) error {
			return nil
		})
	scopeNames = map[string]string{"hello-component1": scopeName, "hello-component2": scopeName, "hello-component3": scopeName, "hello-component4": scopeName}
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_multiComponents2.yaml", scopeNames, true, true)
	mocker.Finish()

	// WHEN the appconfig has multiple components with no logging scopes and the default scope does not exist
	//      and Default is called while the namespace is terminating
	// THEN Default should NOT set the default logging scope on all of the components in the appconfig and
	//      the default logging scope instance should NOT be created
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect get namespace
	cli.EXPECT().Get(gomock.Eq(context.TODO()), gomock.Eq(namespaceName), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, ns *v1.Namespace) error {
			ns.Status.Phase = v1.NamespaceTerminating
			return nil
		})
	testLoggingScopeDefaulterDefault(t, cli, "hello-conf_multiComponents2.yaml", map[string]string{}, false, false)
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
	cli.EXPECT().Delete(gomock.Eq(context.TODO()), gomock.Eq(createDefaultLoggingScope(namespacedName)), gomock.Not(gomock.Nil())).
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

func testLoggingScopeDefaulterDefault(t *testing.T, cli client.Client, configPath string, scopeNames map[string]string, dryRun bool, expectFoundLoggingScope bool) {
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
			if scope.ScopeReference.APIVersion == apiVersion && scope.ScopeReference.Kind == vzapi.LoggingScopeKind && scope.ScopeReference.Name == scopeNames[component.ComponentName] {
				foundLoggingScope = true
			}
		}
		assert.Equal(expectFoundLoggingScope, foundLoggingScope)
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
