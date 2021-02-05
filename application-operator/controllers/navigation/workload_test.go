// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package navigation

import (
	"context"
	"fmt"
	"strings"
	"testing"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	k8sapps "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TestFetchWorkloadDefinition tests the FetchWorkloadDefinition function
func TestFetchWorkloadDefinition(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()
	var err error
	var workload unstructured.Unstructured
	var definition *oamcore.WorkloadDefinition

	// GIVEN a nil workload reference
	// WHEN an attempt is made to fetch the workload definition
	// THEN expect an error
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	definition, err = FetchWorkloadDefinition(ctx, cli, ctrl.Log, nil)
	mocker.Finish()
	assert.Error(err)
	assert.Nil(definition)

	// GIVEN a valid workload reference
	// WHEN an attempt is made to fetch the workload definition
	// THEN the workload definition to be returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, wlDef *oamcore.WorkloadDefinition) error {
			wlDef.SetNamespace(key.Namespace)
			wlDef.SetName(key.Name)
			return nil
		})
	workload = unstructured.Unstructured{}
	workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
	definition, err = FetchWorkloadDefinition(ctx, cli, ctrl.Log, &workload)
	mocker.Finish()
	assert.NoError(err)
	assert.NotNil(definition)
	assert.Equal("containerizedworkloads.core.oam.dev", definition.Name)

	// GIVEN a valid workload reference
	// WHEN an underlying error occurs with the k8s api
	// THEN expect the error will be propagated
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, wlDef *oamcore.WorkloadDefinition) error {
			return fmt.Errorf("test-error")
		})
	workload = unstructured.Unstructured{}
	workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
	definition, err = FetchWorkloadDefinition(ctx, cli, ctrl.Log, &workload)
	mocker.Finish()
	assert.Error(err)
	assert.Equal("test-error", err.Error())
	assert.Nil(definition)
}

// TestFetchWorkloadChildren tests the FetchWorkloadChildren function.
func TestFetchWorkloadChildren(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()
	var err error
	var workload unstructured.Unstructured
	var children []*unstructured.Unstructured

	// GIVEN a nil workload parameter
	// WHEN the workload children are fetched
	// THEN verify an error is returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	children, err = FetchWorkloadChildren(ctx, cli, ctrl.Log, nil)
	mocker.Finish()
	assert.Error(err)
	assert.Len(children, 0)

	// GIVEN a valid list of workload children
	// WHEN the a workloads children are fetched
	// THEN verify that the workload children are returned correctly.
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	// Expect a call to get the containerized workload definition and return one that populates the child resource kinds.
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, wlDef *oamcore.WorkloadDefinition) error {
			wlDef.SetNamespace(key.Namespace)
			wlDef.SetName(key.Name)
			wlDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{{APIVersion: "apps/v1", Kind: "Deployment"}}
			return nil
		})
	// Expect a call to list the children resources and return a list.
	cli.EXPECT().
		List(gomock.Eq(ctx), gomock.Not(gomock.Nil()), gomock.Eq(client.InNamespace("test-namespace")), gomock.Any()).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, namespace client.InNamespace, labels client.MatchingLabels) error {
			assert.Equal("Deployment", resources.GetKind())
			return AppendAsUnstructured(resources, k8sapps.Deployment{
				TypeMeta: metav1.TypeMeta{
					APIVersion: k8sapps.SchemeGroupVersion.String(),
					Kind:       "test-invalid-kind"},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-deployment-name",
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: oamcore.ContainerizedWorkloadKindAPIVersion,
						Kind:       oamcore.ContainerizedWorkloadKind,
						Name:       "test-workload-name",
						UID:        "test-workload-uid"}}}})
		})
	workload = unstructured.Unstructured{}
	workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
	workload.SetNamespace("test-namespace")
	workload.SetName("test-workload-name")
	workload.SetUID("test-workload-uid")
	children, err = FetchWorkloadChildren(ctx, cli, ctrl.Log, &workload)
	mocker.Finish()
	assert.NoError(err)
	assert.Len(children, 1)
	assert.Equal("test-deployment-name", children[0].GetName())

	// GIVEN a request to fetch a workload's children
	// WHEN an underlying kubernetes api error occurs
	// THEN verify that the error is propagated to the caller.
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "", Name: "containerizedworkloads.core.oam.dev"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, wlDef *oamcore.WorkloadDefinition) error {
			wlDef.SetNamespace(key.Namespace)
			wlDef.SetName(key.Name)
			wlDef.Spec.ChildResourceKinds = []oamcore.ChildResourceKind{{APIVersion: "apps/v1", Kind: "Deployment"}}
			return nil
		})
	cli.EXPECT().
		List(gomock.Eq(ctx), gomock.Not(gomock.Nil()), gomock.Eq(client.InNamespace("test-namespace")), gomock.Any()).
		DoAndReturn(func(ctx context.Context, resources *unstructured.UnstructuredList, namespace client.InNamespace, labels client.MatchingLabels) error {
			return fmt.Errorf("test-error")
		})
	workload = unstructured.Unstructured{}
	workload.SetGroupVersionKind(oamcore.ContainerizedWorkloadGroupVersionKind)
	workload.SetNamespace("test-namespace")
	workload.SetName("test-workload-name")
	workload.SetUID("test-workload-uid")
	children, err = FetchWorkloadChildren(ctx, cli, ctrl.Log, &workload)
	mocker.Finish()
	assert.Error(err)
	assert.Equal("test-error", err.Error())
	assert.Len(children, 0)
}

// TestComponentFromWorkloadLabels tests the ComponentFromWorkloadLabels function.
func TestComponentFromWorkloadLabels(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()

	// GIVEN a nil workload labels
	// WHEN an attempt is made to get the component
	// THEN expect an error
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	component, err := ComponentFromWorkloadLabels(ctx, cli, "unit-test-namespace", nil)

	mocker.Finish()
	assert.EqualError(err, "OAM component label missing from metadata")
	assert.Nil(component)

	// GIVEN workload labels with just the component name
	// WHEN an attempt is made to get the component
	// THEN expect an error
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	labels := map[string]string{oam.LabelAppComponent: "unit-test-component"}

	component, err = ComponentFromWorkloadLabels(ctx, cli, "unit-test-namespace", labels)

	mocker.Finish()
	assert.EqualError(err, "OAM app name label missing from metadata")
	assert.Nil(component)

	// GIVEN workload labels
	// WHEN an attempt is made to get the component but there are no matching components in the returned app config
	// THEN expect an error
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	labels = map[string]string{oam.LabelAppComponent: "unit-test-component", oam.LabelAppName: "unit-test-app-config"}

	// expect a call to fetch the oam application configuration
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "unit-test-namespace", Name: "unit-test-app-config"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: "does-not-match"}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})

	component, err = ComponentFromWorkloadLabels(ctx, cli, "unit-test-namespace", labels)

	mocker.Finish()
	assert.EqualError(err, "Unable to find application component for workload")
	assert.Nil(component)

	// GIVEN workload labels
	// WHEN an attempt is made to get the component
	// THEN validate that the expected component is returned
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)
	componentName := "unit-test-component"
	labels = map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: "unit-test-app-config"}

	// expect a call to fetch the oam application configuration
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: "unit-test-namespace", Name: "unit-test-app-config"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})

	component, err = ComponentFromWorkloadLabels(ctx, cli, "unit-test-namespace", labels)

	mocker.Finish()
	assert.NoError(err)
	assert.NotNil(component)
	assert.Equal(componentName, component.ComponentName)
}

// TestLoggingScopeFromWorkloadLabels tests the LoggingScopeFromWorkloadLabels function.
func TestLoggingScopeFromWorkloadLabels(t *testing.T) {
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

	loggingScope, err := LoggingScopeFromWorkloadLabels(ctx, cli, "unit-test-namespace", labels)

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
			return nil
		})

	loggingScope, err = LoggingScopeFromWorkloadLabels(ctx, cli, "unit-test-namespace", labels)

	mocker.Finish()
	assert.NoError(err)
	assert.NotNil(loggingScope)
	assert.Equal(fluentdImage, loggingScope.Spec.FluentdImage)

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

	loggingScope, err = LoggingScopeFromWorkloadLabels(ctx, cli, "unit-test-namespace", labels)

	mocker.Finish()
	assert.True(k8serrors.IsNotFound(err))
	assert.Nil(loggingScope)
}

// TestIsVerrazzanoWorkloadKind tests the IsVerrazzanoWorkloadKind function.
func TestIsVerrazzanoWorkloadKind(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN a Verrazzano workload
	// WHEN a call is made to check if the workload is a Verrazzano workload kind
	// THEN expect the call to return true
	workloadKind := "VerrazzanoCoherenceWorkload"

	u := &unstructured.Unstructured{}
	u.SetKind(workloadKind)

	assert.True(IsVerrazzanoWorkloadKind(u))

	// GIVEN a non-Verrazzano workload
	// WHEN a call is made to check if the workload is a Verrazzano workload kind
	// THEN expect the call to return false
	workloadKind = "ContainerizedWorkload"

	u = &unstructured.Unstructured{}
	u.SetKind(workloadKind)

	assert.False(IsVerrazzanoWorkloadKind(u))
}

// TestGetContainedWorkloadVersionKindName tests the GetContainedWorkloadVersionKindName function.
func TestGetContainedWorkloadVersionKindName(t *testing.T) {
	assert := asserts.New(t)

	// GIVEN a Verrazzano workload containing another workload
	// WHEN a call is made to get the api version, kind, and name of the contained workload
	// THEN the api version, kind, and name are returned
	workloadAPIVersion := "oam.verrazzano.io/v1alpha1"
	workloadKind := "VerrazzanoCoherenceWorkload"

	containedAPIVersion := "oam.verrazzano.io/v1alpha1"
	containedKind := "VerrazzanoCoherenceWorkload"
	containedName := "unit-test-resource"

	containedResource := map[string]interface{}{
		"apiVersion": containedAPIVersion,
		"kind":       containedKind,
		"metadata": map[string]interface{}{
			"name": containedName,
		},
	}

	u := &unstructured.Unstructured{}
	u.SetAPIVersion(workloadAPIVersion)
	u.SetKind(workloadKind)
	unstructured.SetNestedMap(u.Object, containedResource, "spec", "coherence")

	apiVersion, kind, name, err := GetContainedWorkloadVersionKindName(u)

	assert.Nil(err)
	assert.Equal(containedAPIVersion, apiVersion)
	assert.Equal(containedKind, kind)
	assert.Equal(containedName, name)

	// GIVEN a Verrazzano workload containing another workload where the Verrazzano workload is of an unknown kind
	// WHEN a call is made to get the api version, kind, and name of the contained workload
	// THEN an error is returned
	workloadKind = "VerrazzanoBogusWorkload"

	u = &unstructured.Unstructured{}
	u.SetAPIVersion(workloadAPIVersion)
	u.SetKind(workloadKind)

	apiVersion, kind, name, err = GetContainedWorkloadVersionKindName(u)

	assert.Error(err)
	assert.True(strings.HasPrefix(err.Error(), "Unable to find spec property for workload type"))
	assert.Empty(apiVersion)
	assert.Empty(kind)
	assert.Empty(name)

	// GIVEN a Verrazzano workload containing another workload missing the apiVersion field
	// WHEN a call is made to get the api version, kind, and name of the contained workload
	// THEN an error is returned
	workloadAPIVersion = "oam.verrazzano.io/v1alpha1"
	workloadKind = "VerrazzanoCoherenceWorkload"

	containedResource = map[string]interface{}{}

	u = &unstructured.Unstructured{}
	u.SetAPIVersion(workloadAPIVersion)
	u.SetKind(workloadKind)
	unstructured.SetNestedMap(u.Object, containedResource, "spec", "coherence")

	apiVersion, kind, name, err = GetContainedWorkloadVersionKindName(u)

	assert.Error(err)
	assert.True(strings.HasPrefix(err.Error(), "Unable to find api version"))
	assert.Empty(apiVersion)
	assert.Empty(kind)
	assert.Empty(name)

	// GIVEN a Verrazzano workload containing another workload missing the kind field
	// WHEN a call is made to get the api version, kind, and name of the contained workload
	// THEN an error is returned
	workloadAPIVersion = "oam.verrazzano.io/v1alpha1"
	workloadKind = "VerrazzanoCoherenceWorkload"

	containedResource = map[string]interface{}{
		"apiVersion": containedAPIVersion,
	}

	u = &unstructured.Unstructured{}
	u.SetAPIVersion(workloadAPIVersion)
	u.SetKind(workloadKind)
	unstructured.SetNestedMap(u.Object, containedResource, "spec", "coherence")

	apiVersion, kind, name, err = GetContainedWorkloadVersionKindName(u)

	assert.Error(err)
	assert.True(strings.HasPrefix(err.Error(), "Unable to find kind"))
	assert.Empty(apiVersion)
	assert.Empty(kind)
	assert.Empty(name)

	// GIVEN a Verrazzano workload containing another workload missing the metadata name field
	// WHEN a call is made to get the api version, kind, and name of the contained workload
	// THEN an error is returned
	workloadAPIVersion = "oam.verrazzano.io/v1alpha1"
	workloadKind = "VerrazzanoCoherenceWorkload"

	containedResource = map[string]interface{}{
		"apiVersion": containedAPIVersion,
		"kind":       containedKind,
	}

	u = &unstructured.Unstructured{}
	u.SetAPIVersion(workloadAPIVersion)
	u.SetKind(workloadKind)
	unstructured.SetNestedMap(u.Object, containedResource, "spec", "coherence")

	apiVersion, kind, name, err = GetContainedWorkloadVersionKindName(u)

	assert.Error(err)
	assert.True(strings.HasPrefix(err.Error(), "Unable to find metadata name"))
	assert.Empty(apiVersion)
	assert.Empty(kind)
	assert.Empty(name)
}

// TestFetchContainedWorkload tests the FetchContainedWorkload function.
func TestFetchContainedWorkload(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller
	var cli *mocks.MockClient
	var ctx = context.TODO()

	namespace := "unit-test-namespace"
	workloadAPIVersion := "oam.verrazzano.io/v1alpha1"
	workloadKind := "VerrazzanoCoherenceWorkload"

	containedAPIVersion := "coherence.oracle.com/v1"
	containedKind := "Coherence"
	containedName := "unit-test-resource"

	containedResource := map[string]interface{}{
		"apiVersion": containedAPIVersion,
		"kind":       containedKind,
		"metadata": map[string]interface{}{
			"name": containedName,
		},
	}

	u := &unstructured.Unstructured{}
	u.SetNamespace(namespace)
	u.SetAPIVersion(workloadAPIVersion)
	u.SetKind(workloadKind)
	unstructured.SetNestedMap(u.Object, containedResource, "spec", "coherence")

	// GIVEN a Verrazzano workload containing another workload
	// WHEN a call is made to fetch the contained workload and there is an error
	// THEN validate that the call returns an error
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// expect a call to get the contained resource, return an error
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: containedName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, contained *unstructured.Unstructured) error {
			return errors.NewNotFound(schema.GroupResource{}, "Unable to fetch resource")
		})

	contained, err := FetchContainedWorkload(ctx, cli, u)

	assert.True(errors.IsNotFound(err))
	assert.Nil(contained)

	// GIVEN a Verrazzano workload containing another workload
	// WHEN a call is made to fetch the contained workload
	// THEN the call returns the contained workload
	mocker = gomock.NewController(t)
	cli = mocks.NewMockClient(mocker)

	// expect a call to get the contained resource, return it as unstructured
	cli.EXPECT().
		Get(gomock.Eq(ctx), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: containedName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, contained *unstructured.Unstructured) error {
			contained.SetUnstructuredContent(containedResource)
			return nil
		})

	contained, err = FetchContainedWorkload(ctx, cli, u)

	assert.Nil(err)
	assert.Equal(containedAPIVersion, contained.GetAPIVersion())
	assert.Equal(containedKind, contained.GetKind())
	assert.Equal(containedName, contained.GetName())
}
