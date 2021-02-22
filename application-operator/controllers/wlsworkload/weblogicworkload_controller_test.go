// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsworkload

import (
	"context"
	"testing"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/loggingscope"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const namespace = "unit-test-namespace"
const weblogicAPIVersion = "weblogic.oracle/v8"
const weblogicKind = "Domain"

// TestReconcilerSetupWithManager test the creation of the VerrazzanoWebLogicWorkload reconciler.
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

// TestReconcileCreateWebLogicDomain tests the basic happy path of reconciling a VerrazzanoWebLogicWorkload. We
// expect to write out a WebLogic domain CR but we aren't adding logging or any other scopes or traits.
// GIVEN a VerrazzanoWebLogicWorkload resource is created
// WHEN the controller Reconcile function is called
// THEN expect a WebLogic domain CR to be written
func TestReconcileCreateWebLogicDomain(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			weblogicJSON := `{"metadata":{"name":"unit-test-cluster"},"spec":{"domainUID":"unit-test-domain"}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(weblogicJSON)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			return nil
		})
	// expect a call to fetch the oam application configuration
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to create the WebLogic domain CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(weblogicAPIVersion, u.GetAPIVersion())
			assert.Equal(weblogicKind, u.GetKind())

			// make sure the OAM component and app name labels were copied
			specLabels, _, _ := unstructured.NestedStringMap(u.Object, specServerPodLabelsFields...)
			assert.Equal(labels, specLabels)

			// make sure configuration.istio.enabled is false
			specIstioEnabled, _, _ := unstructured.NestedBool(u.Object, specConfigurationIstioEnabledFields...)
			assert.Equal(specIstioEnabled, false)
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-weblogic-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateWebLogicDomainWithLogging tests the happy path of reconciling a VerrazzanoWebLogicWorkload
// with an attached logging scope. We expect to write out a WebLogic domain CR with the FLUENTD sidecar and
// associated volumes and mounts.
// GIVEN a VerrazzanoWebLogicWorkload resource is created with a logging scope
// WHEN the controller Reconcile function is called
// THEN expect a WebLogic domain CR to be written with logging extras.
func TestReconcileCreateWebLogicDomainWithLogging(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	loggingScopeName := "unit-test-logging-scope"
	fluentdImage := "unit-test-image:latest"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			weblogicJSON := `{"metadata":{"name":"unit-test-cluster"},"spec":{"domainUID":"unit-test-domain"}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(weblogicJSON)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			return nil
		})
	// expect a call to fetch the oam application configuration (and the component has an attached logging scope)
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			loggingScope := oamcore.ComponentScope{ScopeReference: oamrt.TypedReference{Kind: vzapi.LoggingScopeKind, Name: loggingScopeName}}
			component.Scopes = []oamcore.ComponentScope{loggingScope}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the logging scope
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: loggingScopeName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, loggingScope *vzapi.LoggingScope) error {
			loggingScope.Spec.FluentdImage = fluentdImage
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
			// return no resources
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(loggingscope.WlsFluentdParsingRules, configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to create the WebLogic domain CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(weblogicAPIVersion, u.GetAPIVersion())
			assert.Equal(weblogicKind, u.GetKind())

			// make sure the OAM component and app name labels were copied
			specLabels, _, _ := unstructured.NestedStringMap(u.Object, specServerPodLabelsFields...)
			assert.Equal(labels, specLabels)

			// make sure the FLUENTD sidecar was added
			containers, _, _ := unstructured.NestedSlice(u.Object, specServerPodContainersFields...)
			assert.Equal(1, len(containers))
			assert.Equal(fluentdImage, containers[0].(map[string]interface{})["image"])
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-weblogic-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileAlreadyExists tests reconciling a VerrazzanoWebLogicWorkload when the WebLogic
// domain CR already exists. We ignore the error and return success.
// GIVEN a VerrazzanoWebLogicWorkload resource
// WHEN the controller Reconcile function is called and the WebLogic domain CR already exists
// THEN ignore the error on create and return success
func TestReconcileAlreadyExists(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			weblogicJSON := `{"metadata":{"name":"unit-test-cluster"},"spec":{"domainUID":"unit-test-domain"}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(weblogicJSON)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			return nil
		})
	// expect a call to fetch the oam application configuration
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to create the WebLogic domain CR and return an AlreadyExists error
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(weblogicAPIVersion, u.GetAPIVersion())
			assert.Equal(weblogicKind, u.GetKind())
			return k8serrors.NewAlreadyExists(k8sschema.GroupResource{}, "")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-weblogic-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileErrorOnCreate tests reconciling a VerrazzanoWebLogicWorkload and an
// error occurs attempting to create the WebLogic domain CR.
// GIVEN a VerrazzanoWebLogicWorkload resource is created
// WHEN the controller Reconcile function is called and there is an error creating the WebLogic domain CR
// THEN expect an error to be returned
func TestReconcileErrorOnCreate(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			weblogicJSON := `{"metadata":{"name":"unit-test-cluster"},"spec":{"domainUID":"unit-test-domain"}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(weblogicJSON)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			return nil
		})
	// expect a call to fetch the oam application configuration
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to create the WebLogic domain CR and return an AlreadyExists error
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(weblogicAPIVersion, u.GetAPIVersion())
			assert.Equal(weblogicKind, u.GetKind())
			return k8serrors.NewBadRequest("an error has occurred")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-weblogic-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.Error(err)
	assert.Equal("an error has occurred", err.Error())
	assert.Equal(false, result.Requeue)
}

// TestReconcileWorkloadNotFound tests reconciling a VerrazzanoWebLogicWorkload when the workload
// cannot be fetched. This happens when the workload has been deleted by the OAM runtime.
// GIVEN a VerrazzanoWebLogicWorkload resource has been deleted
// WHEN the controller Reconcile function is called and we attempt to fetch the workload
// THEN return success from the controller as there is nothing more to do
func TestReconcileWorkloadNotFound(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-weblogic-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileFetchWorkloadError tests reconciling a VerrazzanoWebLogicWorkload when the workload
// cannot be fetched due to an unexpected error.
// GIVEN a VerrazzanoWebLogicWorkload resource has been created
// WHEN the controller Reconcile function is called and we attempt to fetch the workload and get an error
// THEN return the error
func TestReconcileFetchWorkloadError(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			return k8serrors.NewBadRequest("an error has occurred")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-weblogic-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.Equal("an error has occurred", err.Error())
	assert.Equal(false, result.Requeue)
}

// TestCopyLabelsFailure tests reconciling a VerrazzanoWebLogicWorkload and we are
// not able to copy labels to the WebLogic domain CR.
// GIVEN a VerrazzanoWebLogicWorkload resource
// WHEN the controller Reconcile function is called and the labels cannot be copied
// THEN expect an error to be returned
func TestCopyLabelsFailure(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	// expect a call to fetch the VerrazzanoWebLogicWorkload - return a malformed WebLogic resource (spec should be an object
	// so when we attempt to set the labels field inside spec it will fail) - this is a contrived example but it's the easiest
	// way to force error on copying labels
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":27}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-weblogic-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.EqualError(err, "value cannot be set because .spec is not a map[string]interface{}")
	assert.Equal(false, result.Requeue)
}

// TestAddLoggingFailure tests reconciling a VerrazzanoWebLogicWorkload with an attached logging scope
// and we fail to fetch the logging scope data.
// GIVEN a VerrazzanoWebLogicWorkload resource is created with a logging scope
// WHEN the controller Reconcile function is called and there is an error fetching the logging scope
// THEN expect an error to be returned
func TestAddLoggingFailure(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	loggingScopeName := "unit-test-logging-scope"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			weblogicJSON := `{"metadata":{"name":"unit-test-cluster"},"spec":{"domainUID":"unit-test-domain"}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(weblogicJSON)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			return nil
		})
	// expect a call to fetch the oam application configuration (and the component has an attached logging scope)
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			loggingScope := oamcore.ComponentScope{ScopeReference: oamrt.TypedReference{Kind: vzapi.LoggingScopeKind, Name: loggingScopeName}}
			component.Scopes = []oamcore.ComponentScope{loggingScope}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the logging scope and return a NotFound error
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: loggingScopeName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, loggingScope *vzapi.LoggingScope) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-weblogic-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.Error(err)
	assert.True(k8serrors.IsNotFound(err))
	assert.Equal(false, result.Requeue)
}

// TestIstioEnabled tests that domain resource spec.configuration.istio.enabled is set correctly.
// GIVEN a namespace with the label istio-injection equals enabled
// WHEN the label istio-injection equals enabled
// THEN the domain resource to spec.configuration.istio.enabled is set to true
func TestIstioEnabled(t *testing.T) {
	assert := asserts.New(t)
	reconciler := Reconciler{}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Labels: map[string]string{
				"istio-injection": "enabled",
			},
		},
	}
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "Domain",
		},
	}
	err := reconciler.istioEnabled(namespace, u)
	assert.NoError(err, "Unexpected error setting istio enabled")
	specIstioEnabled, _, _ := unstructured.NestedBool(u.Object, specConfigurationIstioEnabledFields...)
	assert.Equal(specIstioEnabled, true)
}

// TestIstioDisabled tests that domain resource spec.configuration.istio.enabled is set correctly.
// GIVEN a namespace with the label istio-injection not set or set to disable
// THEN the domain resource to spec.configuration.istio.enabled is set to false
func TestIstioDisabled(t *testing.T) {
	assert := asserts.New(t)
	reconciler := Reconciler{}

	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
			Labels: map[string]string{
				"istio-injection": "disabled",
			},
		},
	}
	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "Domain",
		},
	}
	err := reconciler.istioEnabled(namespace, u)
	assert.NoError(err, "Unexpected error setting istio enabled")
	specIstioEnabled, _, _ := unstructured.NestedBool(u.Object, specConfigurationIstioEnabledFields...)
	assert.Equal(specIstioEnabled, false)

	namespace = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	u = &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "Domain",
		},
	}
	err = reconciler.istioEnabled(namespace, u)
	assert.NoError(err, "Unexpected error setting istio enabled")
	specIstioEnabled, _, _ = unstructured.NestedBool(u.Object, specConfigurationIstioEnabledFields...)
	assert.Equal(specIstioEnabled, false)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	vzapi.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
// c - The K8s client to inject into the reconciler
func newReconciler(c client.Client) Reconciler {
	return Reconciler{
		Client: c,
		Log:    ctrl.Log.WithName("test"),
		Scheme: newScheme(),
	}
}

// newRequest creates a new reconciler request for testing
// namespace - The namespace to use in the request
// name - The name to use in the request
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	}
}
