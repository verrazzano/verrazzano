// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cohworkload

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus/testutil"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/application-operator/controllers/logging"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzstring "github.com/verrazzano/verrazzano/pkg/string"
	"go.uber.org/zap"
	istionet "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const namespace = "unit-test-namespace"
const coherenceAPIVersion = "coherence.oracle.com/v1"
const coherenceKind = "Coherence"
const testRestartVersion = "new-restart"
const loggingTrait = `
{
	"apiVersion": "oam.verrazzano.io/v1alpha1",
	"kind": "LoggingTrait",
	"name": "my-logging-trait"
}
`

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
	_ = vzapi.AddToScheme(scheme)
	metricsReconciler := &metricstrait.Reconciler{Client: cli, Scheme: scheme, Scraper: "verrazzano-system/vmi-system-prometheus-0"}
	reconciler = Reconciler{Client: cli, Scheme: scheme, Metrics: metricsReconciler}
	mgr.EXPECT().GetControllerOptions().AnyTimes()
	mgr.EXPECT().GetScheme().Return(scheme)
	mgr.EXPECT().GetLogger().Return(logr.Discard())
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
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// expect call to fetch existing coherence StatefulSet
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *v1.StatefulSet) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			coherenceJSON := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(coherenceJSON)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), getUnstructuredConfigMapList(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			// return no resources
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(strings.Join(strings.Split(cohFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		})
	// expect a call to attempt to get the Coherence CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: "unit-test-cluster"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
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
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			namespace.Name = "test-namespace"
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	// expect a call to update the status upgrade version
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	fluentdImage := "unit-test-image:latest"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

	// expect call to fetch existing coherence StatefulSet
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *v1.StatefulSet) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})

	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
			return nil
		})
	// expect a call to fetch the OAM application configuration (and the component has an attached logging scope)
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), getUnstructuredConfigMapList(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			// return no resources
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(strings.Join(strings.Split(cohFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		})
	// expect a call to attempt to get the Coherence CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: "unit-test-cluster"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure JVM args were added
			jvmArgs, _, _ := unstructured.NestedSlice(u.Object, specJvmArgsFields...)
			assert.Equal(additionalJvmArgs, jvmArgs)

			// make sure side car was added
			sideCars, _, _ := unstructured.NestedSlice(u.Object, specField, "sideCars")
			assert.Equal(1, len(sideCars))
			// assert correct Fluentd image
			assert.Equal(fluentdImage, sideCars[0].(map[string]interface{})["image"])

			// make sure sidecar.istio.io/inject annotation was added
			annotations, _, _ := unstructured.NestedStringMap(u.Object, specAnnotationsFields...)
			assert.Equal(annotations, map[string]string{"sidecar.istio.io/inject": "false"})
			return nil
		})
	// expect a call to get the namespace for the Coherence resource
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	// expect a call to update the status upgrade version
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
func TestReconcileCreateCoherenceWithCustomLogging(t *testing.T) {

	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	fluentdImage := "unit-test-image:latest"
	workloadName := "unit-test-verrazzano-coherence-workload"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

	// expect call to fetch existing coherence StatefulSet
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *v1.StatefulSet) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})

	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			workload.Namespace = namespace
			workload.Name = workloadName
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
			workload.OwnerReferences = []metav1.OwnerReference{
				{
					UID: types.UID(namespace),
				},
			}
			return nil
		})
	// expect a call to list the logging traits
	cli.EXPECT().
		List(gomock.Any(), &vzapi.LoggingTraitList{TypeMeta: metav1.TypeMeta{Kind: "LoggingTrait", APIVersion: "oam.verrazzano.io/v1alpha1"}}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, loggingTraitList *vzapi.LoggingTraitList, inNamespace client.InNamespace) error {
			loggingTraitList.Items = []vzapi.LoggingTrait{
				{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								UID: types.UID(namespace),
							},
						},
					},
					Spec: vzapi.LoggingTraitSpec{
						WorkloadReference: oamrt.TypedReference{
							Name: workloadName,
						},
					},
				},
			}
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{
				{
					ComponentName: componentName,
					Traits: []oamcore.ComponentTrait{
						{
							Trait: runtime.RawExtension{
								Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(loggingTrait, " ", ""), "\n", "")),
							},
						},
					},
				},
			}
			return nil
		})
	// expect a call to get the ConfigMap for logging - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: loggingNamePart + "-unit-test-cluster-coherence"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{
				Group:    "",
				Resource: "ConfigMap",
			},
				"logging-stdout-unit-test-cluster-coherence")
		})
	// expect a call to fetch the OAM application configuration (and the component has an attached logging scope)
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), getUnstructuredConfigMapList(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			// return no resources
			return nil
		})
	// Define expected ConfigMap
	data := make(map[string]string)
	data["custom.conf"] = ""
	customLoggingConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "",
			APIVersion: "",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:              loggingNamePart + "-unit-test-cluster-coherence",
			Namespace:         namespace,
			CreationTimestamp: metav1.Time{},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "oam.verrazzano.io/v1alpha1",
					Kind:               "VerrazzanoCoherenceWorkload",
					Name:               "unit-test-verrazzano-coherence-workload",
					UID:                "",
					Controller:         newTrue(),
					BlockOwnerDeletion: newTrue(),
				},
			},
		},
		Data: data,
	}
	// expect a call to create the custom logging config map
	cli.EXPECT().
		Create(gomock.Any(), customLoggingConfigMap, gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(strings.Join(strings.Split(cohFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to attempt to get the Coherence CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: "unit-test-cluster"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
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
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	// expect a call to update the status upgrade version
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
func TestReconcileCreateCoherenceWithCustomLoggingConfigMapExists(t *testing.T) {

	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	fluentdImage := "unit-test-image:latest"
	workloadName := "unit-test-verrazzano-coherence-workload"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

	// expect call to fetch existing coherence StatefulSet
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *v1.StatefulSet) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})

	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			workload.Namespace = namespace
			workload.Name = workloadName
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
			workload.OwnerReferences = []metav1.OwnerReference{
				{
					UID: types.UID(namespace),
				},
			}
			return nil
		})
	// expect a call to list the logging traits
	cli.EXPECT().
		List(gomock.Any(), &vzapi.LoggingTraitList{TypeMeta: metav1.TypeMeta{Kind: "LoggingTrait", APIVersion: "oam.verrazzano.io/v1alpha1"}}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, loggingTraitList *vzapi.LoggingTraitList, inNamespace client.InNamespace) error {
			loggingTraitList.Items = []vzapi.LoggingTrait{
				{
					ObjectMeta: metav1.ObjectMeta{
						OwnerReferences: []metav1.OwnerReference{
							{
								UID: types.UID(namespace),
							},
						},
					},
					Spec: vzapi.LoggingTraitSpec{
						WorkloadReference: oamrt.TypedReference{
							Name: workloadName,
						},
					},
				},
			}
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {

			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{
				{
					ComponentName: componentName,
					Traits: []oamcore.ComponentTrait{
						{
							Trait: runtime.RawExtension{
								Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(loggingTrait, " ", ""), "\n", "")),
							},
						},
					},
				},
			}
			return nil
		})
	// expect a call to get the ConfigMap for logging - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: loggingNamePart + "-unit-test-cluster-coherence"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, configMap *corev1.ConfigMap) error {
			return nil
		})
	// expect a call to fetch the OAM application configuration (and the component has an attached logging scope)
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), getUnstructuredConfigMapList(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			// return no resources
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(strings.Join(strings.Split(cohFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to attempt to get the Coherence CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: "unit-test-cluster"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
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
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	// expect a call to update the status upgrade version
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileUpdateFluentdImage tests reconciling a VerrazzanoCoherenceWorkload when the Fluentd image
// in the Coherence sidecar is old and a new image is available. This should result in the latest Fluentd
// image being pulled from the env and replaced in the sidecar
// GIVEN a VerrazzanoCoherenceWorkload resource that is using an old Fluentd image
// WHEN the controller Reconcile function is called
// THEN the Fluentd image should be replaced in the Fluentd sidecar
func TestReconcileUpdateFluentdImage(t *testing.T) {

	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	fluentdImage := "unit-test-image:latest"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

	// expect call to fetch existing coherence StatefulSet
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *v1.StatefulSet) error {
			// return nil error because Coherence StatefulSet exists
			return nil
		})
	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
			return nil
		})
	// expect a call to fetch the OAM application configuration (and the component has an attached logging scope)
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), getUnstructuredConfigMapList(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			// return no resources
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(strings.Join(strings.Split(cohFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to attempt to get the Coherence CR
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, u *unstructured.Unstructured) error {
			// set the old Fluentd image on the returned obj
			containers, _, _ := unstructured.NestedSlice(u.Object, "spec", "sideCars")
			unstructured.SetNestedField(containers[0].(map[string]interface{}), "unit-test-image:existing", "image")
			unstructured.SetNestedSlice(u.Object, containers, "spec", "sideCars")
			// return nil error because Coherence StatefulSet exists
			return nil
		})
	// expect a call to update the Coherence CR
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.UpdateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure JVM args were added
			jvmArgs, _, _ := unstructured.NestedSlice(u.Object, specJvmArgsFields...)
			assert.Equal(additionalJvmArgs, jvmArgs)

			// make sure side car was added
			sideCars, _, _ := unstructured.NestedSlice(u.Object, specField, "sideCars")
			assert.Equal(1, len(sideCars))
			// assert correct Fluentd image
			assert.Equal(fluentdImage, sideCars[0].(map[string]interface{})["image"])

			// make sure sidecar.istio.io/inject annotation was added
			annotations, _, _ := unstructured.NestedStringMap(u.Object, specAnnotationsFields...)
			assert.Equal(annotations, map[string]string{"sidecar.istio.io/inject": "false"})
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		})
	// expect a call to get the namespace for the Coherence resource
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	existingFluentdImage := "unit-test-image:existing"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	containers := []corev1.Container{{Name: logging.FluentdStdoutSidecarName, Image: existingFluentdImage}}

	// simulate the "replicas" field changing to 3
	replicasFromWorkload := int64(3)

	// expect call to fetch existing coherence StatefulSet
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *v1.StatefulSet) error {
			coherence.Spec.Template.Spec.Containers = containers
			// return nil error because Coherence StatefulSet exists
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
	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":` + fmt.Sprint(replicasFromWorkload) + `}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), getUnstructuredConfigMapList(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			// return no resources
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(strings.Join(strings.Split(cohFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		})
	// expect a call to attempt to get the Coherence CR and return an existing resource
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: "unit-test-cluster"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			// note this resource has a "replicas" field currently set to 1
			spec := map[string]interface{}{"replicas": int64(1)}
			return unstructured.SetNestedField(u.Object, spec, specField)
		})
	// expect a call to update the Coherence CR
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.UpdateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure the replicas field is set to the correct value (from our workload spec)
			spec, _, _ := unstructured.NestedMap(u.Object, specField)
			assert.Equal(replicasFromWorkload, spec["replicas"])
			return nil
		})
	// expect a call to get the namespace for the Coherence resource
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	// expect a call to update the status upgrade version
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	fluentdImage := "unit-test-image:latest"
	existingJvmArg := "-Dcoherence.test=unit-test"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}

	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

	// expect call to fetch existing coherence StatefulSet
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *v1.StatefulSet) error {
			// return nil error because Coherence StatefulSet exists
			return nil
		})
	// expect a call to fetch the OAM application configuration (and the component has an attached logging scope)
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
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"jvm":{"args": ["` + existingJvmArg + `"]}}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), getUnstructuredConfigMapList(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			// return no resources
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(strings.Join(strings.Split(cohFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		})
	// expect a call to attempt to get the Coherence CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: "unit-test-cluster"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the Coherence CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure JVM args were added and that the existing arg is still present
			jvmArgs, _, _ := unstructured.NestedStringSlice(u.Object, specJvmArgsFields...)
			assert.Equal(len(additionalJvmArgs)+1, len(jvmArgs))
			assert.True(vzstring.SliceContainsString(jvmArgs, existingJvmArg))

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
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	// expect a call to update the status upgrade version
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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

	// expect call to fetch existing coherence StatefulSet
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *v1.StatefulSet) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
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
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), getUnstructuredConfigMapList(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			// return no resources
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(strings.Join(strings.Split(cohFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		})
	// expect a call to attempt to get the Coherence CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: "unit-test-cluster"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the Coherence CR and return an error
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())
			return k8serrors.NewBadRequest("An error has occurred")
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.Nil(err)
	assert.True(result.Requeue)
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
	result, err := reconciler.Reconcile(context.TODO(), request)

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
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.Nil(err)
	assert.True(result.Requeue)
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
		Return(k8serrors.NewNotFound(k8sschema.GroupResource{Group: "test-space", Resource: "DestinationRule"}, "test-space-myapp-dr"))

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
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
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
	err := reconciler.createOrUpdateDestinationRule(context.Background(), vzlog.DefaultLogger(), "test-namespace", namespaceLabels, workloadLabels)
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
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, dr *istioclient.DestinationRule, opts ...client.UpdateOption) error {
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
	err := reconciler.createOrUpdateDestinationRule(context.Background(), vzlog.DefaultLogger(), "test-namespace", namespaceLabels, workloadLabels)
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
	err := reconciler.createOrUpdateDestinationRule(context.Background(), vzlog.DefaultLogger(), "test-namespace", namespaceLabels, workloadLabels)
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
	err := reconciler.createOrUpdateDestinationRule(context.Background(), vzlog.DefaultLogger(), "test-namespace", namespaceLabels, workloadLabels)
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
		Log:     zap.S().With("test"),
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

// Used for bool in struct literal
func newTrue() *bool {
	b := true
	return &b
}

func getUnstructuredConfigMapList() *unstructured.UnstructuredList {
	unstructuredConfigMapList := &unstructured.UnstructuredList{}
	unstructuredConfigMapList.SetAPIVersion("v1")
	unstructuredConfigMapList.SetKind("ConfigMap")
	return unstructuredConfigMapList
}

// TestReconcileRestart tests reconciling a VerrazzanoCoherenceWorkload with the restart-version specified in its annotations.
// This should result in restart-version annotation written to the Coherence CR.
// GIVEN a VerrazzanoCoherenceWorkload resource
// WHEN the controller Reconcile function is called and the restart-version is specified in annotations
// THEN the restart-version annotation written  to the Coherence CR
func TestReconcileRestart(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	fluentdImage := "unit-test-image:latest"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName}
	annotations := map[string]string{vzconst.RestartVersionAnnotation: testRestartVersion}

	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

	// expect call to fetch existing coherence StatefulSet
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *v1.StatefulSet) error {
			// return nil error because Coherence StatefulSet exists
			return nil
		})
	// expect a call to fetch the VerrazzanoCoherenceWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-coherence-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoCoherenceWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":{"replicas":3}}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.ObjectMeta.Labels = labels
			workload.ObjectMeta.Annotations = annotations
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoCoherenceWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
			return nil
		})
	// expect a call to fetch the OAM application configuration (and the component has an attached logging scope)
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, appConfig *oamcore.ApplicationConfiguration) error {
			component := oamcore.ApplicationConfigurationComponent{ComponentName: componentName}
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), getUnstructuredConfigMapList(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list *unstructured.UnstructuredList, opts ...client.ListOption) error {
			// return no resources
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(strings.Join(strings.Split(cohFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to attempt to get the Coherence CR
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, u *unstructured.Unstructured) error {
			// set the old Fluentd image on the returned obj
			containers, _, _ := unstructured.NestedSlice(u.Object, "spec", "sideCars")
			unstructured.SetNestedField(containers[0].(map[string]interface{}), "unit-test-image:existing", "image")
			unstructured.SetNestedSlice(u.Object, containers, "spec", "sideCars")
			// return nil error because Coherence StatefulSet exists
			return nil
		})
	// expect a call to update the Coherence CR
	cli.EXPECT().
		Update(gomock.Any(), gomock.AssignableToTypeOf(&unstructured.Unstructured{}), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.UpdateOption) error {
			assert.Equal(coherenceAPIVersion, u.GetAPIVersion())
			assert.Equal(coherenceKind, u.GetKind())

			// make sure JVM args were added
			jvmArgs, _, _ := unstructured.NestedSlice(u.Object, specJvmArgsFields...)
			assert.Equal(additionalJvmArgs, jvmArgs)

			// make sure side car was added
			sideCars, _, _ := unstructured.NestedSlice(u.Object, specField, "sideCars")
			assert.Equal(1, len(sideCars))
			// assert correct Fluentd image
			assert.Equal(fluentdImage, sideCars[0].(map[string]interface{})["image"])

			// make sure sidecar.istio.io/inject annotation was added
			annotations, _, _ := unstructured.NestedStringMap(u.Object, specAnnotationsFields...)
			assert.Equal(annotations, map[string]string{"sidecar.istio.io/inject": "false", vzconst.RestartVersionAnnotation: testRestartVersion})
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		})
	// expect a call to get the namespace for the Coherence resource
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	// expect a call to update the status upgrade version
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoCoherenceWorkload, opts ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(namespace, "unit-test-verrazzano-coherence-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileKubeSystem tests to make sure we do not reconcile
// Any resource that belong to the kube-system namespace
func TestReconcileKubeSystem(t *testing.T) {

	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// create a request and reconcile it
	request := newRequest(vzconst.KubeSystem, "unit-test-verrazzano-helidon-workload")
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.Nil(err)
	assert.True(result.IsZero())
}

// TestReconcileFailed tests to make sure the failure metric is being exposed
func TestReconcileFailed(t *testing.T) {
	testAppConfigName := "unit-test-app-config"
	testNamespace := "test-ns"

	scheme := k8scheme.Scheme
	assert := asserts.New(t)
	clientBuilder := fake.NewClientBuilder().WithScheme(scheme).Build()
	// Create a request and reconcile it
	reconciler := newReconciler(clientBuilder)
	request := newRequest(testNamespace, testAppConfigName)
	reconcileerrorCounterObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.CohworkloadReconcileError)
	assert.NoError(err)
	// Expect a call to fetch the error
	reconcileFailedCounterBefore := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	reconciler.Reconcile(context.TODO(), request)
	reconcileFailedCounterAfter := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	assert.Equal(reconcileFailedCounterBefore, reconcileFailedCounterAfter-1)
}
