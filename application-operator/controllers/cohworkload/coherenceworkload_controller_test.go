// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cohworkload

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers"
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
const coherenceAPIVersion = "coherence.oracle.com/v1"
const coherenceKind = "Coherence"

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

// TestReconcileCreateCoherence tests the basic happy path of reconciling a VerrazzanoCoherenceWorkload. We
// expect to write out a Coherence CR but we aren't adding logging or any other scopes or traits.
// GIVEN a VerrazzanoCoherenceWorkload resource is created
// WHEN the controller Reconcile function is called
// THEN expect a Coherence CR to be written
func TestReconcileCreateCoherence(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			labelsJSON, _ := json.Marshal(labels)
			coherenceJSON := `{"metadata":{"name":"unit-test-cluster","labels":` + string(labelsJSON) + `},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(coherenceJSON)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			return nil
		})
	// expect a call to add a finalizer
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			assert.Equal(workload.ObjectMeta.Finalizers[0], finalizer)
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
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure the OAM component and app name labels were copied
			assert.Equal(labels, u.GetLabels())
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

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	loggingScopeName := "unit-test-logging-scope"
	fluentdImage := "unit-test-image:latest"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

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
	// expect a call to add a finalizer
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			assert.Equal(workload.ObjectMeta.Finalizers[0], finalizer)
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
			assert.Equal(fluentdParsingRules, configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure JVM args were added
			jvmArgs, _, _ := unstructured.NestedSlice(u.Object, "spec", "jvm", "args")
			assert.Equal(additionalJvmArgs, jvmArgs)

			// make sure side car was added
			sideCars, _, _ := unstructured.NestedSlice(u.Object, "spec", "sideCars")
			assert.Equal(1, len(sideCars))
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

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	loggingScopeName := "unit-test-logging-scope"
	fluentdImage := "unit-test-image:latest"
	existingJvmArg := "-Dcoherence.test=unit-test"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

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
	// expect a call to add a finalizer
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			assert.Equal(workload.ObjectMeta.Finalizers[0], finalizer)
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
			assert.Equal(fluentdParsingRules, configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure JVM args were added and that the existing arg is still present
			jvmArgs, _, _ := unstructured.NestedStringSlice(u.Object, "spec", "jvm", "args")
			assert.Equal(len(additionalJvmArgs)+1, len(jvmArgs))
			assert.True(controllers.StringSliceContainsString(jvmArgs, existingJvmArg))

			// make sure side car was added
			sideCars, _, _ := unstructured.NestedSlice(u.Object, "spec", "sideCars")
			assert.Equal(1, len(sideCars))
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

// TestReconcileDeleteResources tests the happy path of reconciling a VerrazzanoCoherenceWorkload when
// the workload is being deleted. We delete resources that were created for FLUENTD as well as
// the Coherence CR we created.
// GIVEN a VerrazzanoCoherenceWorkload resource is being deleted
// WHEN the controller Reconcile function is called
// THEN expect delete calls for resources we created
func TestReconcileDeleteResources(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	// expect a call to fetch the VerrazzanoCoherenceWorkload - set the deletion timestamp to trigger the
	// delete workflow
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
			workload.ObjectMeta.Finalizers = []string{finalizer}
			workload.APIVersion = vzapi.GroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
			// one item in the list is enough to cause the FLUENTD code to try delete the config map
			configMaps := list.(*unstructured.UnstructuredList)
			configMaps.Items = []unstructured.Unstructured{{}}
			return nil
		})
	// expect a call to delete the FLUENTD config map
	cli.EXPECT().
		Delete(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.DeleteOption) error {
			return nil
		})
	// expect a call to delete the Coherence CR
	cli.EXPECT().
		Delete(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.DeleteOption) error {
			return nil
		})
	// expect a call to update the workload to remove the finalizer
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			assert.Equal(0, len(workload.ObjectMeta.Finalizers))
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

// TestReconcileAlreadyExists tests reconciling a VerrazzanoCoherenceWorkload when the Coherence
// CR already exists. We ignore the error and return success.
// GIVEN a VerrazzanoCoherenceWorkload resource is created
// WHEN the controller Reconcile function is called and the Coherence CR already exists
// THEN ignore the error on create and return success
func TestReconcileAlreadyExists(t *testing.T) {
	assert := asserts.New(t)

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

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
	// expect a call to add a finalizer
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			assert.Equal(workload.ObjectMeta.Finalizers[0], finalizer)
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
	// expect a call to create the Coherence CR and return an AlreadyExists error
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())
			return k8serrors.NewAlreadyExists(k8sschema.GroupResource{}, "")
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

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

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
	// expect a call to add a finalizer
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			assert.Equal(workload.ObjectMeta.Finalizers[0], finalizer)
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

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

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

	var mocker *gomock.Controller = gomock.NewController(t)
	var cli *mocks.MockClient = mocks.NewMockClient(mocker)

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
