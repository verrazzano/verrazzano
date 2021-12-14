// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package wlsworkload

import (
	"context"
	"strings"
	"testing"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	"github.com/crossplane/oam-kubernetes-runtime/apis/core"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/crossplane/oam-kubernetes-runtime/pkg/oam"
	"github.com/golang/mock/gomock"
	asserts "github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	wls "github.com/verrazzano/verrazzano/application-operator/apis/weblogic/v8"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/application-operator/controllers/logging"
	"github.com/verrazzano/verrazzano/application-operator/controllers/metricstrait"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	istionet "istio.io/api/networking/v1alpha3"
	istioclient "istio.io/client-go/pkg/apis/networking/v1alpha3"
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
	"sigs.k8s.io/yaml"
)

const namespace = "unit-test-namespace"
const restartVersion = "new-restart"
const weblogicAPIVersion = "weblogic.oracle/v8"
const weblogicKind = "Domain"
const weblogicDomainName = "unit-test-domain"
const weblogicDomain = `
{
   "metadata": {
      "name": "unit-test-cluster"
   },
   "spec": {
      "domainUID": "unit-test-domain"
   }
}
`
const weblogicDomainWithMonitoringExporter = `
{
   "metadata": {
      "name": "unit-test-cluster"
   },
   "spec": {
      "domainUID": "unit-test-domain",
      "monitoringExporter": {
         "imagePullPolicy": "IfNotPresent",
         "configuration": {
            "metricsNameSnakeCase": true,
            "domainQualifier": true,
            "queries": [
               {
                  "JVMRuntime": {
                     "prefix": "wls_jvm_",
                     "key": "name"
                  }
               }
            ]
         }
      }
   }
}
`
const weblogicDomainWithWDTConfigMap = `
{
   "metadata": {
      "name": "unit-test-cluster"
   },
   "spec": {
      "domainUID": "unit-test-domain",
      "configuration": {
         "model": {
            "configMap": "wdt-config-map"
         }
      }
   }
}
`
const loggingTrait = `
{
	"apiVersion": "oam.verrazzano.io/v1alpha1",
	"kind": "LoggingTrait",
	"name": "my-logging-trait"
}
`

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

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *wls.Domain) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomain, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
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
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect call to fetch the WDT config Map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getWDTConfigMapName(weblogicDomainName)}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, getWDTConfigMapName(weblogicDomainName))
		})
	// no WDT config maps found, so expect a call to create a WDT config map
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			assert.Equal(weblogicDomainName, configMap.ObjectMeta.Labels[webLogicDomainUIDLabel])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		}).Times(2)
	// expect a call to attempt to get the WebLogic CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
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

			// make sure the restartVersion is empty
			domainRestartVersion, _, _ := unstructured.NestedString(u.Object, specRestartVersionFields...)
			assert.Equal("", domainRestartVersion)

			// make sure monitoringExporter exists
			validateDefaultMonitoringExporter(u, t)

			// make sure default WDT configMap exists
			validateDefaultWDTConfigMap(u, t)

			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapi.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
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

// TestReconcileCreateWebLogicDomainWithMonitoringExporter tests the basic happy path of reconciling a VerrazzanoWebLogicWorkload
// with monitoringExporter. We expect to write out a WebLogic domain CR with this monitoringExporter intact.
// GIVEN a VerrazzanoWebLogicWorkload resource is created
// WHEN the controller Reconcile function is called
// THEN expect a WebLogic domain CR to be written
func TestReconcileCreateWebLogicDomainWithMonitoringExporter(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *wls.Domain) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomainWithMonitoringExporter, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
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
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect call to fetch the WDT config Map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getWDTConfigMapName(weblogicDomainName)}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, getWDTConfigMapName(weblogicDomainName))
		})
	// no WDT config maps found, so expect a call to create a WDT config map
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		}).Times(2)
	// expect a call to attempt to get the WebLogic CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
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

			// make sure the restartVersion is empty
			domainRestartVersion, _, _ := unstructured.NestedString(u.Object, specRestartVersionFields...)
			assert.Equal("", domainRestartVersion)

			// make sure monitoringExporter exists
			validateTestMonitoringExporter(u, t)

			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapi.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
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

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	fluentdImage := "unit-test-image:latest"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}

	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *wls.Domain) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomain, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
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
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect call to fetch the WDT config Map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getWDTConfigMapName(weblogicDomainName)}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, getWDTConfigMapName(weblogicDomainName))
		})
	// no WDT config maps found, so expect a call to create a WDT config map
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		}).Times(2)
	// expect a call to attempt to get the WebLogic CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
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

			// make sure the FLUENTD sidecar was added
			containers, _, _ := unstructured.NestedSlice(u.Object, specServerPodContainersFields...)
			assert.Equal(1, len(containers))
			assert.Equal(fluentdImage, containers[0].(map[string]interface{})["image"])

			// make sure the restartVersion is empty
			domainRestartVersion, _, _ := unstructured.NestedString(u.Object, specRestartVersionFields...)
			assert.Equal("", domainRestartVersion)

			// make sure monitoringExporter exists
			validateDefaultMonitoringExporter(u, t)

			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapi.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
			//		asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
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

// TestReconcileCreateWebLogicDomainWithCustomLogging tests the happy path of reconciling a VerrazzanoWebLogicWorkload
// with a custom logging trait. We expect to write out a WebLogic domain CR with an extra FLUENTD sidecar,
// ConfigMap, and associated volumes and mounts.
// GIVEN a VerrazzanoWebLogicWorkload resource is created with a custom logging trait
// WHEN the controller Reconcile function is called
// THEN expect a WebLogic domain CR to be written with custom logging extras.
func TestReconcileCreateWebLogicDomainWithCustomLogging(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	workloadName := "unit-test-verrazzano-weblogic-workload"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *wls.Domain) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomain, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
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
	// expect a call to list the FLUENTD config maps
	cli.EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, list runtime.Object, opts ...client.ListOption) error {
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
			Name:              loggingNamePart + "-unit-test-cluster-domain",
			Namespace:         namespace,
			CreationTimestamp: metav1.Time{},
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "oam.verrazzano.io/v1alpha1",
					Kind:               "VerrazzanoWebLogicWorkload",
					Name:               "unit-test-verrazzano-weblogic-workload",
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
		Create(gomock.Any(), customLoggingConfigMap).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			return nil
		})
	// no config maps found, so expect a call to create a config map with our parsing rules
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap, opts ...client.CreateOption) error {
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect call to fetch the WDT config Map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getWDTConfigMapName(weblogicDomainName)}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, getWDTConfigMapName(weblogicDomainName))
		})
	// no WDT config maps found, so expect a call to create a WDT config map
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
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
		}).Times(2)
	// expect a call to get the ConfigMap for logging - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: "logging-stdout-unit-test-cluster-domain"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{
				Group:    "",
				Resource: "ConfigMap",
			},
				"logging-stdout-unit-test-cluster-domain")
		})
	// expect a call to attempt to get the WebLogic CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
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

			// make sure the restartVersion is empty
			domainRestartVersion, _, _ := unstructured.NestedString(u.Object, specRestartVersionFields...)
			assert.Equal("", domainRestartVersion)

			// make sure monitoringExporter exists
			validateDefaultMonitoringExporter(u, t)

			// make sure default WDT configMap exists
			validateDefaultWDTConfigMap(u, t)

			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapi.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
			//		asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
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

// TestReconcileCreateWebLogicDomainWithCustomLogging tests the happy path of reconciling a VerrazzanoWebLogicWorkload
// with a custom logging trait. We expect to write out a WebLogic domain CR with an extra FLUENTD sidecar
// and associated volumes and mounts. This test, we are testing the case when the ConfigMap already exists
// GIVEN a VerrazzanoWebLogicWorkload resource is created with a custom logging trait
// WHEN the controller Reconcile function is called
// THEN expect a WebLogic domain CR to be written with custom logging extras.
func TestReconcileCreateWebLogicDomainWithCustomLoggingConfigMapExists(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	workloadName := "unit-test-verrazzano-weblogic-workload"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *wls.Domain) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: workloadName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomain, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
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
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect call to fetch the WDT config Map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getWDTConfigMapName(weblogicDomainName)}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, getWDTConfigMapName(weblogicDomainName))
		})
	// no WDT config maps found, so expect a call to create a WDT config map
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
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
		}).Times(2)
	// expect a call to get the ConfigMap for logging
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: "logging-stdout-unit-test-cluster-domain"}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return nil
		})
	// expect a call to attempt to get the WebLogic CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
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

			// make sure the restartVersion is empty
			domainRestartVersion, _, _ := unstructured.NestedString(u.Object, specRestartVersionFields...)
			assert.Equal("", domainRestartVersion)

			// make sure monitoringExporter exists
			validateDefaultMonitoringExporter(u, t)

			// make sure default WDT configMap exists
			validateDefaultWDTConfigMap(u, t)

			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapi.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
			//		asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
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

// TestReconcileCreateWebLogicDomainWithWDTConfigMap tests the basic happy path of reconciling a VerrazzanoWebLogicWorkload
// with WDT configMap. We expect to update this configMap with default WebLogic plugin configuration details.
// GIVEN a VerrazzanoWebLogicWorkload resource is created
// WHEN the controller Reconcile function is called
// THEN expect a WebLogic domain CR to be written with WDT configMap updated with WebLogic plugin configuration details.
func TestReconcileCreateWebLogicDomainWithWDTConfigMap(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *wls.Domain) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomainWithWDTConfigMap, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
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
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		})

	// expect call to fetch the WDT config map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "wdt-config-map"}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			// setup a scaled down existing scrape config entry for cluster1
			configMap.Data = map[string]string{
				"resources": "test",
			}
			return nil
		})
	// WDT config map found, so expect a call to update a WDT config map
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			return nil
		})

	// expect a call to attempt to get the WebLogic CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the WebLogic domain CR
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			validateWDTConfigMap(u, t)
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapi.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
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

// TestReconcileUpdateFluentdImage tests reconciling a VerrazzanoWebLogicWorkload when the Fluentd image
// in the managed server pod sidecar is old and a new image is available. This should result in the latest Fluentd
// image being pulled from the env and replaced in the sidecar
// GIVEN a VerrazzanoWebLogicWorkload resource that is using an old Fluentd image
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
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}

	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *wls.Domain) error {
			// return nil error to simulate domain existing
			return nil
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomain, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
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
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect call to fetch the WDT config Map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getWDTConfigMapName(weblogicDomainName)}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, getWDTConfigMapName(weblogicDomainName))
		})
	// no WDT config maps found, so expect a call to create a WDT config map
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to attempt to get the Coherence CR
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, u *unstructured.Unstructured) error {
			// set the old Fluentd image on the returned obj
			containers, _, _ := unstructured.NestedSlice(u.Object, "spec", "serverPod", "containers")
			unstructured.SetNestedField(containers[0].(map[string]interface{}), "unit-test-image:existing", "image")
			unstructured.SetNestedSlice(u.Object, containers, "spec", "serverPod", "containers")
			// return nil error because Coherence StatefulSet exists
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		}).Times(2)
	// expect a call to create the WebLogic domain CR
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(weblogicAPIVersion, u.GetAPIVersion())
			assert.Equal(weblogicKind, u.GetKind())

			// make sure the OAM component and app name labels were copied and the WebLogic type lobel applied
			specLabels, _, _ := unstructured.NestedStringMap(u.Object, specServerPodLabelsFields...)
			assert.Equal(3, len(specLabels))
			assert.Equal("unit-test-component", specLabels["app.oam.dev/component"])
			assert.Equal("unit-test-app-config", specLabels["app.oam.dev/name"])
			assert.Equal(constants.WorkloadTypeWeblogic, specLabels[constants.LabelWorkloadType])

			// make sure the FLUENTD sidecar was added
			containers, _, _ := unstructured.NestedSlice(u.Object, specServerPodContainersFields...)
			assert.Equal(1, len(containers))
			assert.Equal(fluentdImage, containers[0].(map[string]interface{})["image"])

			// make sure the restartVersion is empty
			domainRestartVersion, _, _ := unstructured.NestedString(u.Object, specRestartVersionFields...)
			assert.Equal("", domainRestartVersion)

			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()
	// expect a call to update the status upgrade version
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

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

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	mockStatus := mocks.NewMockStatusWriter(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, coherence *wls.Domain) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "test")
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomain, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
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
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect call to fetch the WDT config Map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getWDTConfigMapName(weblogicDomainName)}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, getWDTConfigMapName(weblogicDomainName))
		})
	// no WDT config maps found, so expect a call to create a WDT config map
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		}).Times(2)
	// expect a call to attempt to get the WebLogic CR - return not found
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, u *unstructured.Unstructured) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, "")
		})
	// expect a call to create the WebLogic domain CR and return a BadRequest error
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(weblogicAPIVersion, u.GetAPIVersion())
			assert.Equal(weblogicKind, u.GetKind())

			// make sure the restartVersion is empty
			domainRestartVersion, _, _ := unstructured.NestedString(u.Object, specRestartVersionFields...)
			assert.Equal("", domainRestartVersion)

			return k8serrors.NewBadRequest("an error has occurred")
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

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

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

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

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

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

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)
	//	mockStatus := mocks.NewMockStatusWriter(mocker)

	// expect a call to fetch the VerrazzanoWebLogicWorkload - return a malformed WebLogic resource (spec should be an object
	// so when we attempt to set the labels field inside spec it will fail) - this is a contrived example but it's the easiest
	// way to force error on copying labels
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			json := `{"metadata":{"name":"unit-test-cluster"},"spec":27}`
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(json)}
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
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

// TestCreateDestinationRuleCreate tests creation of a destination rule
// GIVEN the destination rule does not exist
// WHEN the controller createDestinationRule function is called
// THEN expect no error to be returned and destination rule is created
func TestCreateDestinationRuleCreate(t *testing.T) {
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
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, dr *istioclient.DestinationRule, opts ...client.CreateOption) error {
			assert.Equal(destinationRuleKind, dr.Kind)
			assert.Equal(destinationRuleAPIVersion, dr.APIVersion)
			assert.Equal("*.test-namespace.svc.cluster.local", dr.Spec.Host)
			assert.Equal(istionet.ClientTLSSettings_ISTIO_MUTUAL, dr.Spec.TrafficPolicy.Tls.Mode)
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
	err := reconciler.createDestinationRule(context.Background(), ctrl.Log, "test-namespace", namespaceLabels, workloadLabels)
	mocker.Finish()
	assert.NoError(err)
}

// TestCreateDestinationRuleNoCreate tests that a destination rule already exist
// GIVEN the destination rule exist
// WHEN the controller createDestinationRule function is called
// THEN expect no error to be returned and destination rule is not created
func TestCreateDestinationRuleNoCreate(t *testing.T) {
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

	scheme := runtime.NewScheme()
	istioclient.AddToScheme(scheme)
	core.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	reconciler := Reconciler{Client: cli, Scheme: scheme}

	namespaceLabels := make(map[string]string)
	namespaceLabels["istio-injection"] = "enabled"
	workloadLabels := make(map[string]string)
	workloadLabels["app.oam.dev/name"] = "test-app"
	err := reconciler.createDestinationRule(context.Background(), ctrl.Log, "test-namespace", namespaceLabels, workloadLabels)
	mocker.Finish()
	assert.NoError(err)
}

// TestCreateDestinationRuleNoOamLabel tests creation of a destination rule with no oam label found
// GIVEN no app.oam.dev/name label specified
// WHEN the controller createDestinationRule function is called
// THEN expect an error to be returned
func TestCreateDestinationRuleNoOamLabel(t *testing.T) {
	assert := asserts.New(t)

	reconciler := Reconciler{}
	namespaceLabels := make(map[string]string)
	namespaceLabels["istio-injection"] = "enabled"
	workloadLabels := make(map[string]string)
	err := reconciler.createDestinationRule(context.Background(), ctrl.Log, "test-namespace", namespaceLabels, workloadLabels)
	assert.Equal("OAM app name label missing from metadata, unable to generate destination rule name", err.Error())
}

// TestCreateDestinationRuleNoIstioLabel tests creation of a destination rule with no istio label found
// GIVEN no istio-injection label specified
// WHEN the controller createDestinationRule function is called
// THEN expect an error to be returned
func TestCreateDestinationRuleNoIstioLabel(t *testing.T) {
	assert := asserts.New(t)

	reconciler := Reconciler{}
	namespaceLabels := make(map[string]string)
	workloadLabels := make(map[string]string)
	err := reconciler.createDestinationRule(context.Background(), ctrl.Log, "test-namespace", namespaceLabels, workloadLabels)
	assert.NoError(err)
}

// TestCreateRuntimeEncryptionSecretCreate tests creation of a runtimeEncryptionSecret
// GIVEN the runtime encryption secret does not exist
// WHEN the controller CreateRuntimeEncryptionSecret function is called
// THEN expect no error to be returned and runtime encryption secret is created
func TestCreateRuntimeEncryptionSecretCreate(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// Expect a call to get a secret and return that it is not found.
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-secret"}, gomock.Not(gomock.Nil())).
		Return(k8serrors.NewNotFound(k8sschema.GroupResource{Group: "test-space", Resource: "Secret"}, "test-space-secret"))

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

	// Expect a call to create the secret and return success
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, secret *corev1.Secret, opts ...client.CreateOption) error {
			assert.Equal("Secret", secret.Kind)
			assert.Equal("v1", secret.APIVersion)
			assert.Len(secret.Data, 1)
			assert.Equal(1, len(secret.OwnerReferences))
			assert.Equal("ApplicationConfiguration", secret.OwnerReferences[0].Kind)
			assert.Equal("core.oam.dev/v1alpha2", secret.OwnerReferences[0].APIVersion)
			return nil
		})

	scheme := runtime.NewScheme()
	core.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	reconciler := Reconciler{Client: cli, Scheme: scheme}

	workloadLabels := make(map[string]string)
	workloadLabels["app.oam.dev/name"] = "test-app"
	err := reconciler.createRuntimeEncryptionSecret(context.Background(), ctrl.Log, "test-namespace", "test-secret", workloadLabels)
	mocker.Finish()
	assert.NoError(err)
}

// TestCreateRuntimeEncryptionSecretNoCreate tests that a runtimeEncryptionSecret already exist
// GIVEN the runtime encryption secret exist
// WHEN the controller createRuntimeEncryptionSecret function is called
// THEN expect no error to be returned and runtime encryption secret is not created
func TestCreateRuntimeEncryptionSecretNoCreate(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// Expect a call to get a secret and return that it was found.
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: "test-namespace", Name: "test-secret"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, dr *corev1.Secret) error {
			dr.TypeMeta = metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret"}
			return nil
		})

	scheme := runtime.NewScheme()
	core.AddToScheme(scheme)
	vzapi.AddToScheme(scheme)
	reconciler := Reconciler{Client: cli, Scheme: scheme}

	workloadLabels := make(map[string]string)
	workloadLabels["app.oam.dev/name"] = "test-app"
	err := reconciler.createRuntimeEncryptionSecret(context.Background(), ctrl.Log, "test-namespace", "test-secret", workloadLabels)
	mocker.Finish()
	assert.NoError(err)
}

// TestCreateRuntimeEncryptionSecretNoOamLabel tests creation of a runtime encryption secret with no oam label found
// GIVEN no app.oam.dev/name label specified
// WHEN the controller createRuntimeEncryptionSecret function is called
// THEN expect an error to be returned
func TestCreateRuntimeEncryptionSecretNoOamLabel(t *testing.T) {
	assert := asserts.New(t)

	reconciler := Reconciler{}
	workloadLabels := make(map[string]string)
	err := reconciler.createRuntimeEncryptionSecret(context.Background(), ctrl.Log, "test-namespace", "test-secret", workloadLabels)
	assert.Equal("OAM app name label missing from metadata, unable to create owner reference to appconfig", err.Error())
}

// TestIstioEnabled tests that domain resource spec.configuration.istio.enabled is set correctly.
// GIVEN istio-injection is enabled
// THEN the domain resource to spec.configuration.istio.enabled is set to true
func TestIstioEnabled(t *testing.T) {
	assert := asserts.New(t)

	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "Domain",
		},
	}
	namespaceLabels := make(map[string]string)
	namespaceLabels["istio-injection"] = "enabled"
	err := updateIstioEnabled(namespaceLabels, u)
	assert.NoError(err, "Unexpected error setting istio enabled")
	specIstioEnabled, _, _ := unstructured.NestedBool(u.Object, specConfigurationIstioEnabledFields...)
	assert.Equal(specIstioEnabled, true)
}

// TestIstioDisabled tests that domain resource spec.configuration.istio.enabled is set correctly.
// GIVEN istio-injection is disabled
// THEN the domain resource to spec.configuration.istio.enabled is set to false
func TestIstioDisabled(t *testing.T) {
	assert := asserts.New(t)

	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"kind": "Domain",
		},
	}
	namespaceLabels := make(map[string]string)
	namespaceLabels["istio-injection"] = "disabled"
	err := updateIstioEnabled(namespaceLabels, u)
	assert.NoError(err, "Unexpected error setting istio enabled")
	specIstioEnabled, _, _ := unstructured.NestedBool(u.Object, specConfigurationIstioEnabledFields...)
	assert.Equal(specIstioEnabled, false)
}

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	core.AddToScheme(scheme)
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

// validateDefaultMonitoringExporter validates the default monitoringExporter in the WebLogic domain spec
func validateDefaultMonitoringExporter(u *unstructured.Unstructured, t *testing.T) {
	_, found, err := unstructured.NestedFieldNoCopy(u.Object, specMonitoringExporterFields...)
	asserts.Nil(t, err, "Expect no error finding monitoringExporter in WebLogic domain CR")
	asserts.True(t, found, "Found monitoringExporter in WebLogic domain CR")
	imagePullPolicy, _, _ := unstructured.NestedFieldNoCopy(u.Object, append(specMonitoringExporterFields, "imagePullPolicy")...)
	asserts.Equal(t, "IfNotPresent", imagePullPolicy, "monitoringExporter.imagePullPolicy should be IfNotPresent in WebLogic domain CR")
	domainQualifier, _, _ := unstructured.NestedBool(u.Object, append(specMonitoringExporterFields, "configuration", "domainQualifier")...)
	asserts.True(t, domainQualifier, "monitoringExporter.configuration.domainQualifier should be TRUE")
	metricsNameSnakeCase, _, _ := unstructured.NestedBool(u.Object, append(specMonitoringExporterFields, "configuration", "metricsNameSnakeCase")...)
	asserts.True(t, metricsNameSnakeCase, "monitoringExporter.configuration.metricsNameSnakeCase should be TRUE")
	queries, _, _ := unstructured.NestedSlice(u.Object, append(specMonitoringExporterFields, "configuration", "queries")...)
	asserts.Equal(t, 9, len(queries), "there should be 9 queries")
	query, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&queries[0])
	runtimeType, _, _ := unstructured.NestedString(query, "applicationRuntimes", "componentRuntimes", "type")
	asserts.Equal(t, "WebAppComponentRuntime", runtimeType, "query runtime type should be WebAppComponentRuntime")
}

// validateTestMonitoringExporter validates the test monitoringExporter in the WebLogic domain spec
func validateTestMonitoringExporter(u *unstructured.Unstructured, t *testing.T) {
	_, found, err := unstructured.NestedFieldNoCopy(u.Object, specMonitoringExporterFields...)
	asserts.Nil(t, err, "Expect no error finding monitoringExporter in WebLogic domain CR")
	asserts.True(t, found, "Found monitoringExporter in WebLogic domain CR")
	imagePullPolicy, _, _ := unstructured.NestedFieldNoCopy(u.Object, append(specMonitoringExporterFields, "imagePullPolicy")...)
	asserts.Equal(t, "IfNotPresent", imagePullPolicy, "monitoringExporter.imagePullPolicy should be IfNotPresent in WebLogic domain CR")
	domainQualifier, _, _ := unstructured.NestedBool(u.Object, append(specMonitoringExporterFields, "configuration", "domainQualifier")...)
	asserts.True(t, domainQualifier, "monitoringExporter.configuration.domainQualifier should be TRUE")
	metricsNameSnakeCase, _, _ := unstructured.NestedBool(u.Object, append(specMonitoringExporterFields, "configuration", "metricsNameSnakeCase")...)
	asserts.True(t, metricsNameSnakeCase, "monitoringExporter.configuration.metricsNameSnakeCase should be TRUE")
	queries, _, _ := unstructured.NestedSlice(u.Object, append(specMonitoringExporterFields, "configuration", "queries")...)
	asserts.Equal(t, 1, len(queries), "there should be 1 query")
	query, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&queries[0])
	jvmRuntimePrefix, _, _ := unstructured.NestedString(query, "JVMRuntime", "prefix")
	asserts.Equal(t, "wls_jvm_", jvmRuntimePrefix, "query JVMRuntime prefix should be wls_jvm_")
}

// validateDefaultWDTConfigMap validates the default WDT config map in the WebLogic domain spec
func validateDefaultWDTConfigMap(u *unstructured.Unstructured, t *testing.T) {
	mapName, found, err := unstructured.NestedString(u.Object, specConfigurationWDTConfigMap...)
	asserts.Nil(t, err, "Expect no error finding WDTConfigMap in WebLogic domain CR")
	asserts.True(t, found, "Found WDTConfigMap in WebLogic domain CR")
	asserts.Equal(t, mapName, getWDTConfigMapName(weblogicDomainName), "mapName should be ")
}

// validateWDTConfigMap validates the WDT config name in the WebLogic domain spec
func validateWDTConfigMap(u *unstructured.Unstructured, t *testing.T) {
	mapName, found, err := unstructured.NestedString(u.Object, specConfigurationWDTConfigMap...)
	asserts.Nil(t, err, "Expect no error finding WDTConfigMap in WebLogic domain CR")
	asserts.True(t, found, "Found WDTConfigMap in WebLogic domain CR")
	asserts.Equal(t, mapName, "wdt-config-map", "mapName should be ")
}

// Used for bool in struct literal
func newTrue() *bool {
	b := true
	return &b
}

// TestGetWLSLogPath tests getWLSLogPath correctly.
// GIVEN a weblogic env
// THEN the log path is built
func TestGetWLSLogPath(t *testing.T) {
	assert := asserts.New(t)
	logPath := getWLSLogPath("test-domain")
	assert.Equal("/scratch/logs/test-domain/$(SERVER_NAME).log,/scratch/logs/test-domain/$(SERVER_NAME)_access.log,/scratch/logs/test-domain/$(SERVER_NAME)_nodemanager.log,/scratch/logs/test-domain/$(DOMAIN_UID).log", logPath)
}

// TestReconcileRestart tests reconciling a VerrazzanoWebLogicWorkload when the WebLogic
// domain CR already exists and the restart-version specified in the annotations.
// This should result in restartVersion written to the WLS domain .
// GIVEN a VerrazzanoWebLogicWorkload resource
// WHEN the controller Reconcile function is called and the WebLogic domain CR already exists and the restart-version is specified
// THEN the WLS domain has restartVersion
func TestReconcileRestart(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"
	fluentdImage := "unit-test-image:latest"
	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}
	annotations := map[string]string{vzconst.RestartVersionAnnotation: restartVersion}
	mockStatus := mocks.NewMockStatusWriter(mocker)

	// set the Fluentd image which is obtained via env then reset at end of test
	initialDefaultFluentdImage := logging.DefaultFluentdImage
	logging.DefaultFluentdImage = fluentdImage
	defer func() { logging.DefaultFluentdImage = initialDefaultFluentdImage }()

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, domain *wls.Domain) error {
			// return nil error to simulate domain existing
			return nil
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomain, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.ObjectMeta.Annotations = annotations
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
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
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect call to fetch the WDT config Map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getWDTConfigMapName(weblogicDomainName)}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, getWDTConfigMapName(weblogicDomainName))
		})
	// no WDT config maps found, so expect a call to create a WDT config map
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to attempt to get the Coherence CR
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, u *unstructured.Unstructured) error {
			// set the old Fluentd image on the returned obj
			containers, _, _ := unstructured.NestedSlice(u.Object, "spec", "serverPod", "containers")
			unstructured.SetNestedField(containers[0].(map[string]interface{}), "unit-test-image:existing", "image")
			unstructured.SetNestedSlice(u.Object, containers, "spec", "serverPod", "containers")
			// return nil error because Coherence StatefulSet exists
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		}).Times(2)
	// expect a call to create the WebLogic domain CR
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(weblogicAPIVersion, u.GetAPIVersion())
			assert.Equal(weblogicKind, u.GetKind())

			// make sure the OAM component and app name labels were copied and the WebLogic type lobel applied
			specLabels, _, _ := unstructured.NestedStringMap(u.Object, specServerPodLabelsFields...)
			assert.Equal(3, len(specLabels))
			assert.Equal("unit-test-component", specLabels["app.oam.dev/component"])
			assert.Equal("unit-test-app-config", specLabels["app.oam.dev/name"])
			assert.Equal(constants.WorkloadTypeWeblogic, specLabels[constants.LabelWorkloadType])

			// make sure the FLUENTD sidecar was added
			containers, _, _ := unstructured.NestedSlice(u.Object, specServerPodContainersFields...)
			assert.Equal(1, len(containers))
			assert.Equal(fluentdImage, containers[0].(map[string]interface{})["image"])

			// make sure the restartVersion was added to the domain
			domainRestartVersion, _, _ := unstructured.NestedString(u.Object, specRestartVersionFields...)
			assert.Equal(restartVersion, domainRestartVersion)

			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapi.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
			//		asserts.NotZero(len(verrazzano.Status.Components), "Status.Components len should not be zero")
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

// TestReconcileStopDomain tests reconciling a VerrazzanoWebLogicWorkload when the WebLogic
// domain CR already exists and the lifecycle-action==stop is specified in the annotations.
// GIVEN a VerrazzanoWebLogicWorkload resource
// WHEN the controller Reconcile function is called and the WebLogic domain CR already exists and the restart-version is specified
// THEN the WLS domain has restartVersion
//   This should result in:
//     1. NEVER written to the WLS domain serverStartPolicy
//     2. The old serverStartPolicy saved in the domain annotation
//     3. The WebLogic workload.Status.LastLifeCycleAction should have stop
func TestReconcileStopDomain(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"

	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}
	annotations := map[string]string{vzconst.LifecycleActionAnnotation: vzconst.LifecycleActionStop}
	mockStatus := mocks.NewMockStatusWriter(mocker)

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, domain *wls.Domain) error {
			// return nil error to simulate domain existing
			return nil
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomain, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.ObjectMeta.Annotations = annotations
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
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
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect call to fetch the WDT config Map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getWDTConfigMapName(weblogicDomainName)}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, getWDTConfigMapName(weblogicDomainName))
		})
	// no WDT config maps found, so expect a call to create a WDT config map
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to attempt to get the WebLogic CR
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, u *unstructured.Unstructured) error {
			// set the old Fluentd image on the returned obj
			containers, _, _ := unstructured.NestedSlice(u.Object, "spec", "serverPod", "containers")
			unstructured.SetNestedField(containers[0].(map[string]interface{}), "unit-test-image:existing", "image")
			unstructured.SetNestedSlice(u.Object, containers, "spec", "serverPod", "containers")
			// return nil error because Coherence StatefulSet exists
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		}).Times(2)
	// expect a call to update the WebLogic domain CR
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(weblogicAPIVersion, u.GetAPIVersion())
			assert.Equal(weblogicKind, u.GetKind())

			// make sure the restartVersion was added to the domain
			policy, _, _ := unstructured.NestedString(u.Object, specServerStartPolicyFields...)
			assert.Equal(Never, policy)

			annos, _, _ := unstructured.NestedStringMap(u.Object, metaAnnotationFields...)
			assert.Equal(annos[lastServerStartPolicyAnnotation], IfNeeded)
			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapi.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
			assert.Equal(vzconst.LifecycleActionStop, wl.Status.LastLifecycleAction)
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

// TestReconcileStartDomain tests reconciling a VerrazzanoWebLogicWorkload when the WebLogic
// domain CR already exists and the lifecycle-action==start is specified in the annotations.
// GIVEN a VerrazzanoWebLogicWorkload resource
// WHEN the controller Reconcile function is called and the WebLogic domain CR already exists and the restart-version is specified
// THEN the WLS domain has restartVersion
//   This should result in:
//     1. IF_NEEDED written to the WLS domain serverStartPolicy
//     2. The WebLogic workload.Status.LastLifeCycleAction should have start
func TestReconcileStartDomain(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	appConfigName := "unit-test-app-config"
	componentName := "unit-test-component"

	labels := map[string]string{oam.LabelAppComponent: componentName, oam.LabelAppName: appConfigName,
		constants.LabelWorkloadType: constants.WorkloadTypeWeblogic}
	annotations := map[string]string{vzconst.LifecycleActionAnnotation: vzconst.LifecycleActionStart}
	mockStatus := mocks.NewMockStatusWriter(mocker)

	// expect call to fetch existing WebLogic Domain
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, domain *wls.Domain) error {
			// return nil error to simulate domain existing
			return nil
		})
	// expect a call to fetch the VerrazzanoWebLogicWorkload
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-verrazzano-weblogic-workload"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, workload *vzapi.VerrazzanoWebLogicWorkload) error {
			workload.Spec.Template = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicDomain, " ", ""), "\n", ""))}
			workload.ObjectMeta.Labels = labels
			workload.ObjectMeta.Annotations = annotations
			workload.APIVersion = vzapi.SchemeGroupVersion.String()
			workload.Kind = "VerrazzanoWebLogicWorkload"
			workload.Namespace = namespace
			workload.ObjectMeta.Generation = 2
			workload.Status.LastGeneration = "1"
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
			assert.Equal(strings.Join(strings.Split(WlsFluentdParsingRules, "{{ .CAFile}}"), ""), configMap.Data["fluentd.conf"])
			return nil
		})
	// expect call to fetch the WDT config Map
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: getWDTConfigMapName(weblogicDomainName)}, gomock.Any()).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, configMap *corev1.ConfigMap) error {
			return k8serrors.NewNotFound(k8sschema.GroupResource{}, getWDTConfigMapName(weblogicDomainName))
		})
	// no WDT config maps found, so expect a call to create a WDT config map
	cli.EXPECT().
		Create(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, configMap *corev1.ConfigMap) error {
			bytes, _ := yaml.JSONToYAML([]byte(defaultWDTConfigMapData))
			assert.Equal(string(bytes), configMap.Data[webLogicPluginConfigYamlKey])
			return nil
		})
	// expect a call to get the namespace for the domain
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(client.ObjectKey{Namespace: "", Name: namespace}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, key client.ObjectKey, namespace *corev1.Namespace) error {
			return nil
		})
	// expect a call to attempt to get the WebLogic CR
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: namespace, Name: "unit-test-cluster"}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, u *unstructured.Unstructured) error {
			// set the old Fluentd image on the returned obj
			containers, _, _ := unstructured.NestedSlice(u.Object, "spec", "serverPod", "containers")
			unstructured.SetNestedField(containers[0].(map[string]interface{}), "unit-test-image:existing", "image")
			unstructured.SetNestedSlice(u.Object, containers, "spec", "serverPod", "containers")
			// return nil error because Coherence StatefulSet exists
			return nil
		})
	// expect a call to get the application configuration for the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Eq(types.NamespacedName{Namespace: namespace, Name: appConfigName}), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamcore.ApplicationConfiguration) error {
			appConfig.Spec.Components = []oamcore.ApplicationConfigurationComponent{{ComponentName: componentName}}
			return nil
		}).Times(2)
	// expect a call to update the WebLogic domain CR
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, u *unstructured.Unstructured, opts ...client.CreateOption) error {
			assert.Equal(weblogicAPIVersion, u.GetAPIVersion())
			assert.Equal(weblogicKind, u.GetKind())

			// make sure the restartVersion was added to the domain
			policy, _, _ := unstructured.NestedString(u.Object, specServerStartPolicyFields...)
			assert.Equal(IfNeeded, policy)

			return nil
		})

	// expect a call to status update
	cli.EXPECT().Status().Return(mockStatus).AnyTimes()

	// Expect a call to update the status of the Verrazzano resource to update components
	mockStatus.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, wl *vzapi.VerrazzanoWebLogicWorkload, opts ...client.UpdateOption) error {
			assert.Equal(vzconst.LifecycleActionStart, wl.Status.LastLifecycleAction)
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
