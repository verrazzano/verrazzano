// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cohworkload

import (
	"context"
	"fmt"
	"testing"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	istionet "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const namespace = "unit-test-namespace"
const coherenceAPIVersion = "coherence.oracle.com/v1"
const coherenceKind = "Coherence"

var specJvmArgsFields = []string{specField, jvmField, argsField}

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
	metricsReconciler := &metricstrait.Reconciler{Client: cli, Scheme: scheme, Scraper: "verrazzano-system/vmi-system-prometheus-0"}
	reconciler = Reconciler{Client: cli, Scheme: scheme, Metrics: metricsReconciler}
	mgr.EXPECT().GetConfig().Return(&rest.Config{})
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(log.NullLogger{})
	mgr.EXPECT().SetFields(gomock.Any()).Return(nil).AnyTimes()
	mgr.EXPECT().Add(gomock.Any()).Return(nil).AnyTimes()
	err = reconciler.SetupWithManager(mgr)
	mocker.Finish()
	assert.NoError(err)
}

// TestReconcileCreateCoherence tests the basic happy path of reconciling a VerrazzanoCoherenceWorkload. We
// expect to write out a Coherence CR but we aren't adding logging or any other scopes or traits.
// GIVEN a VerrazzanoCoherenceWorkload resource is created
// WHEN the controller Reconcile function is called
// THEN expect a Coherence CR to be written
func TestReconcileCreateCoherence(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect a call to fetch the OAM application configuration
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})

	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			coherenceJSON := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(coherenceJSON)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			return nil
		})
	// expect a call to fetch the OAM application configuration
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to attempt to get the Coherence CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return errors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure the OAM component and app name labels were copied
			specLabels, _, _ := unstructured.NestedStringMap(u.Object, specLabelsFields...)
			assert.Equal(labels, specLabels)

			// make sure sidecar.istio.io/inject annotation was added
			annotations, _, _ := unstructured.NestedStringMap(u.Object, specAnnotationsFields...)
			assert.Equal(annotations, map[string]string{"sidecar.istio.io/inject": "false"})
			return nil
		})
	// expect a call to get the namespace for the Coherence resource
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileCreateCoherenceWithLogging tests the happy path of reconciling a VerrazzanoCoherenceWorkload with
// an attached logging scope. We expect to write out a Coherence CR with the FLUENTD sidecar and
// additional JVM args set.
// GIVEN a VerrazzanoCoherenceWorkload resource is created with a logging scope
// WHEN the controller Reconcile function is called
// THEN expect a Coherence CR to be written
func TestReconcileCreateCoherenceWithLogging(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	loggingScopeName := "unit-test-logging-scope"
	fluentdImage := "unit-test-image:latest"
	loggingSecretName := "logging-secret"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect a call to fetch the OAM application configuration (and the component has an attached logging scope)
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			loggingScope := oamcore.ComponentScope{ScopeReference: oamrt.TypedReference{Kind: vzapi.LoggingScopeKind, Name: loggingScopeName}}
			component.Scopes = []oamcore.ComponentScope{loggingScope}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			return nil
		})
	// expect a call to fetch the OAM application configuration (and the component has an attached logging scope)
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
			loggingScope.Spec.SecretName = loggingSecretName
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
			assert.Equal(fluentdParsingRules, configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to get the elasticsearch secret in app namespace - return not found
	testLoggingSecretFullName := types.NamespacedName{Namespace: namespace, Name: loggingSecretName}
	cli.EXPECT().
		Get(gomock.Any(), testLoggingSecretFullName, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.ParseGroupResource("v1.Secret"), loggingSecretName))

	// expect a call to create an empty elasticsearch secret in app namespace (default behavior, so
	// that fluentd volume mount works)
	cli.EXPECT().
		Create(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *corev1.Secret, options *client.CreateOptions) error {
			asserts.Equal(t, namespace, sec.Namespace)
			asserts.Equal(t, loggingSecretName, sec.Name)
			asserts.Nil(t, sec.Data)
			asserts.Equal(t, client.CreateOptions{}, *options)
			return nil
		})
	// expect a call to attempt to get the Coherence CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return errors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure JVM args were added
			jvmArgs, _, _ := unstructured.NestedSlice(u.Object, specJvmArgsFields...)
			assert.Equal(additionalJvmArgs, jvmArgs)

			// make sure side car was added
			sideCars, _, _ := unstructured.NestedSlice(u.Object, specField, "sideCars")
			assert.Equal(1, len(sideCars))

			// make sure sidecar.istio.io/inject annotation was added
			annotations, _, _ := unstructured.NestedStringMap(u.Object, specAnnotationsFields...)
			assert.Equal(annotations, map[string]string{"sidecar.istio.io/inject": "false"})
			return nil
		})
	// expect a call to get the namespace for the Coherence resource
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileWithLoggingWithJvmArgs tests the happy path of reconciling a VerrazzanoCoherenceWorkload with
// an attached logging scope and the Coherence spec as existing JVM args. We expect to write out a Coherence CR
// with the FLUENTD sidecar and a JVM args list that has the user-specified args and the args we add
// for logging.
// GIVEN a VerrazzanoCoherenceWorkload resource is created with a logging scope and JVM args
// WHEN the controller Reconcile function is called
// THEN expect a Coherence CR to be written with the combined JVM args
func TestReconcileWithLoggingWithJvmArgs(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	loggingScopeName := "unit-test-logging-scope"
	fluentdImage := "unit-test-image:latest"
	existingJvmArg := "-Dcoherence.test=unit-test"
	loggingSecretName := "logging-secret"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect a call to fetch the OAM application configuration (and the component has an attached logging scope)
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			loggingScope := oamcore.ComponentScope{ScopeReference: oamrt.TypedReference{Kind: vzapi.LoggingScopeKind, Name: loggingScopeName}}
			component.Scopes = []oamcore.ComponentScope{loggingScope}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"jvm":{"args": ["` + existingJvmArg + `"]}}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			return nil
		})
	// expect a call to fetch the OAM application configuration (and the component has an attached logging scope)
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
			loggingScope.Spec.SecretName = loggingSecretName
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
			assert.Equal(fluentdParsingRules, configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to get the elasticsearch secret in app namespace - return not found
	testLoggingSecretFullName := types.NamespacedName{Namespace: namespace, Name: loggingSecretName}
	cli.EXPECT().
		Get(gomock.Any(), testLoggingSecretFullName, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.ParseGroupResource("v1.Secret"), loggingSecretName))

	// expect a call to create an empty elasticsearch secret in app namespace (default behavior, so
	// that fluentd volume mount works)
	cli.EXPECT().
		Create(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, sec *corev1.Secret, options *client.CreateOptions) error {
			asserts.Equal(t, namespace, sec.Namespace)
			asserts.Equal(t, loggingSecretName, sec.Name)
			asserts.Nil(t, sec.Data)
			asserts.Equal(t, client.CreateOptions{}, *options)
			return nil
		})
	// expect a call to attempt to get the Coherence CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return errors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure JVM args were added and that the existing arg is still present
			jvmArgs, _, _ := unstructured.NestedStringSlice(u.Object, specJvmArgsFields...)
			assert.Equal(len(additionalJvmArgs)+1, len(jvmArgs))
			assert.True(controllers.StringSliceContainsString(jvmArgs, existingJvmArg))

			// make sure side car was added
			sideCars, _, _ := unstructured.NestedSlice(u.Object, specField, "sideCars")
			assert.Equal(1, len(sideCars))

			// make sure sidecar.istio.io/inject annotation was added
			annotations, _, _ := unstructured.NestedStringMap(u.Object, specAnnotationsFields...)
			assert.Equal(annotations, map[string]string{"sidecar.istio.io/inject": "false"})
			return nil
		})
	// expect a call to get the namespace for the Coherence resource
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileUpdateCR tests reconciling a VerrazzanoCoherenceWorkload when the Coherence
// CR already exists. We expect the CR to be updated.
// GIVEN a VerrazzanoCoherenceWorkload resource is updated
// WHEN the controller Reconcile function is called and the Coherence CR already exists
// THEN the Coherence CR is updated
func TestReconcileUpdateCR(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// simulate the "replicas" field changing to 3
	replicasFromWorkload := int64(3)

	// expect a call to fetch the OAM application configuration
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":` + fmt.Sprint(replicasFromWorkload) + `}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			return nil
		})
	// expect a call to fetch the OAM application configuration
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to attempt to get the Coherence CR and return an existing resource
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			// note this resource has a "replicas" field currently set to 1
			spec := map[string]interface{}{"replicas": int64(1)}
			return unstructured.SetNestedField(u.Object, spec, specField)
		})
	// expect a call to update the Coherence CR
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure the replicas field is set to the correct value (from our workload spec)
			spec, _, _ := unstructured.NestedMap(u.Object, specField)
			assert.Equal(replicasFromWorkload, spec["replicas"])
			return nil
		})
	// expect a call to get the namespace for the Coherence resource
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileErrorOnCreate tests reconciling a VerrazzanoCoherenceWorkload and an
// error occurs attempting to create the Coherence CR.
// GIVEN a VerrazzanoCoherenceWorkload resource is created
// WHEN the controller Reconcile function is called and there is an error creating the Coherence CR
// THEN expect an error to be returned
func TestReconcileErrorOnCreate(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect a call to fetch the OAM application configuration
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			return nil
		})
	// expect a call to fetch the OAM application configuration
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to attempt to get the Coherence CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return errors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the Coherence CR and return an error
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())
			return k8serrors.NewBadRequest("An error has occurred")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.Error(err)
	assert.Equal("An error has occurred", err.Error())
	assert.Equal(false, result.Requeue)
}

// TestReconcileWorkloadNotFound tests reconciling a VerrazzanoCoherenceWorkload when the workload
// cannot be fetched. This happens when the workload has been deleted by the OAM runtime.
// GIVEN a VerrazzanoCoherenceWorkload resource has been deleted
// WHEN the controller Reconcile function is called and we attempt to fetch the workload
// THEN return success from the controller as there is nothing more to do
func TestReconcileWorkloadNotFound(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileFetchWorkloadError tests reconciling a VerrazzanoCoherenceWorkload when the workload
// cannot be fetched due to an unexpected error.
// GIVEN a VerrazzanoCoherenceWorkload resource has been created
// WHEN the controller Reconcile function is called and we attempt to fetch the workload and get an error
// THEN return the error
func TestReconcileFetchWorkloadError(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			return k8serrors.NewBadRequest("An error has occurred")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.Equal("An error has occurred", err.Error())
	assert.Equal(false, result.Requeue)
}

// TestCreateUpdateDestinationRuleCreate tests creation of a destination rule
// GIVEN the destination rule does not exist
// WHEN the controller createOrUpdateDestinationRule function is called
// THEN expect no error to be returned and destination rule is created
func TestCreateUpdateDestinationRuleCreate(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// Expect a call to get a destination rule and return that it is not found.
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-app"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(schema.GroupResource{Group: "test-space", Resource: "DestinationRule"}, "test-space-myapp-dr"))

	// Expect a call to get the appconfig resource to set the owner reference
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-app"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, app *oamcore.ApplicationConfiguration) error {
			app.TypeMeta = metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ApplicationConfiguration",
			}
			return nil
		})

	// Expect a call to create the destinationRule and return success
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, dr *istioclient.DestinationRule, opts ...client.CreateOption) error {
			assert.Equal(destinationRuleKind, dr.Kind)
			assert.Equal(destinationRuleAPIVersion, dr.APIVersion)
			assert.Equal("*.test-namespace.svc.cluster.local", dr.Spec.Host)
			assert.Equal(istionet.ClientTLSSettings_ISTIO_MUTUAL, dr.Spec.TrafficPolicy.Tls.Mode)
			assert.Equal(uint32(coherenceExtendPort), dr.Spec.TrafficPolicy.PortLevelSettings[0].Port.Number)
			assert.Equal(istionet.ClientTLSSettings_DISABLE, dr.Spec.TrafficPolicy.PortLevelSettings[0].Tls.Mode)
			assert.Equal(1, len(dr.OwnerReferences))
			assert.Equal("ApplicationConfiguration", dr.OwnerReferences[0].Kind)
			assert.Equal("core.oam.dev/v1alpha2", dr.OwnerReferences[0].APIVersion)
			return nil
		})

	scheme := runtime.NewScheme()
	istioclient.AddToScheme(scheme)
	core.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	reconciler := Reconciler{Client: cli, Scheme: scheme}

	namespaceLabels := make(map[string]string)
	namespaceLabels["istio-injection"] = "enabled"
	workloadLabels := make(map[string]string)
	workloadLabels["app.oam.dev/name"] = "test-app"
	err := reconciler.createOrUpdateDestinationRule(context.Background(), ctrl.Log, "test-namespace", namespaceLabels, workloadLabels)
	mocker.Finish()
	assert.NoError(err)
}

// TestCreateUpdateDestinationRuleUpdate tests update of a destination rule
// GIVEN the destination rule exist
// WHEN the controller createOrUpdateDestinationRule function is called
// THEN expect no error to be returned and destination rule is updated
func TestCreateUpdateDestinationRuleUpdate(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// Expect a call to get a destination rule and return that it was found.
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-app"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dr *istioclient.DestinationRule) error {
			dr.TypeMeta = metav1.TypeMeta{
				APIVersion: destinationRuleAPIVersion,
				Kind:       destinationRuleKind}
			return nil
		})

	// Expect a call to get the appconfig resource to set the owner reference
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-app"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, app *oamcore.ApplicationConfiguration) error {
			app.TypeMeta = metav1.TypeMeta{
				APIVersion: "core.oam.dev/v1alpha2",
				Kind:       "ApplicationConfiguration",
			}
			return nil
		})

	// Expect a call to update the destinationRule and return success
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, dr *istioclient.DestinationRule, opts ...client.CreateOption) error {
			assert.Equal(destinationRuleKind, dr.Kind)
			assert.Equal(destinationRuleAPIVersion, dr.APIVersion)
			assert.Equal("*.test-namespace.svc.cluster.local", dr.Spec.Host)
			assert.Equal(istionet.ClientTLSSettings_ISTIO_MUTUAL, dr.Spec.TrafficPolicy.Tls.Mode)
			assert.Equal(uint32(coherenceExtendPort), dr.Spec.TrafficPolicy.PortLevelSettings[0].Port.Number)
			assert.Equal(istionet.ClientTLSSettings_DISABLE, dr.Spec.TrafficPolicy.PortLevelSettings[0].Tls.Mode)
			assert.Equal(1, len(dr.OwnerReferences))
			assert.Equal("ApplicationConfiguration", dr.OwnerReferences[0].Kind)
			assert.Equal("core.oam.dev/v1alpha2", dr.OwnerReferences[0].APIVersion)
			return nil
		})

	scheme := runtime.NewScheme()
	istioclient.AddToScheme(scheme)
	core.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	reconciler := Reconciler{Client: cli, Scheme: scheme}

	namespaceLabels := make(map[string]string)
	namespaceLabels["istio-injection"] = "enabled"
	workloadLabels := make(map[string]string)
	workloadLabels["app.oam.dev/name"] = "test-app"
	err := reconciler.createOrUpdateDestinationRule(context.Background(), ctrl.Log, "test-namespace", namespaceLabels, workloadLabels)
	mocker.Finish()
	assert.NoError(err)
}

// TestCreateUpdateDestinationRuleNoOamLabel tests failure when no OAM label found
// GIVEN no app.oam.dev/name label specified
// WHEN the controller createOrUpdateDestinationRule function is called
// THEN expect an error to be returned
func TestCreateUpdateDestinationRuleNoOamLabel(t *testing.T) {
	assert := asserts.New(t)

	reconciler := Reconciler{}
	namespaceLabels := make(map[string]string)
	namespaceLabels["istio-injection"] = "enabled"
	workloadLabels := make(map[string]string)
	err := reconciler.createOrUpdateDestinationRule(context.Background(), ctrl.Log, "test-namespace", namespaceLabels, workloadLabels)
	assert.Equal("OAM app name label missing from metadata, unable to generate destination rule name", err.Error())
}

// TestCreateUpdateDestinationRuleNoIstioLabel tests failure when no istio label found
// GIVEN no istio-injection label specified
// WHEN the controller createOrUpdateDestinationRule function is called
// THEN expect an error to be returned
func TestCreateUpdateDestinationRuleNoLabel(t *testing.T) {
	assert := asserts.New(t)

	reconciler := Reconciler{}
	namespaceLabels := make(map[string]string)
	workloadLabels := make(map[string]string)
	err := reconciler.createOrUpdateDestinationRule(context.Background(), ctrl.Log, "test-namespace", namespaceLabels, workloadLabels)
	assert.NoError(err)
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
	scheme := newScheme()
	metricsReconciler := &metricstrait.Reconciler{Client: c, Scheme: scheme, Scraper: "verrazzano-system/vmi-system-prometheus-0"}
	return Reconciler{
		Client:  c,
		Log:     ctrl.Log.WithName("test"),
		Scheme:  scheme,
		Metrics: metricsReconciler,
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
