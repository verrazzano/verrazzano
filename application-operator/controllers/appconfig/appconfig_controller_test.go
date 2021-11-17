// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appconfig

import (
	"context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/golang/mock/gomock"
	"github.com/verrazzano/verrazzano/application-operator/mocks"

	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	asserts "github.com/stretchr/testify/assert"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace         = "test-ns"
	testAppConfigName     = "test-appconfig"
	testComponentName     = "test-component"
	testNewRestartVersion = "test-new-restart"
	testDeploymentName    = "test-deployment"
	testStatefulSetName   = "test-statefulset"
	testDaemonSetName     = "test-daemonset"
)

// newScheme creates a new scheme that includes this package's object to use for testing
func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	oamcore.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
func newReconciler(c client.Client) Reconciler {
	return Reconciler{
		Client: c,
		Log:    ctrl.Log.WithName("test"),
		Scheme: newScheme(),
	}
}

// newRequest creates a new reconciler request for testing
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: testNamespace,
			Name:      name,
		},
	}
}

// newAppConfig creates a minimal ApplicationConfiguration struct
func newAppConfig() *oamv1.ApplicationConfiguration {
	return &oamv1.ApplicationConfiguration{
		ObjectMeta: v1.ObjectMeta{
			Name:        testAppConfigName,
			Namespace:   testNamespace,
			Annotations: make(map[string]string),
		},
	}
}

func TestReconcileApplicationConfigurationNotFound(t *testing.T) {
	assert := asserts.New(t)
	oamcore.AddToScheme(k8scheme.Scheme)
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	reconciler := newReconciler(client)
	request := newRequest(testNamespace, testAppConfigName)

	_, err := reconciler.Reconcile(request)
	assert.NoError(err)
}

func TestReconcileNoRestartVersion(t *testing.T) {
	assert := asserts.New(t)
	oamcore.AddToScheme(k8scheme.Scheme)
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	reconciler := newReconciler(client)
	request := newRequest(testNamespace, testAppConfigName)

	err := client.Create(context.TODO(), newAppConfig())
	assert.NoError(err)

	_, err = reconciler.Reconcile(request)
	assert.NoError(err)
}

func TestReconcileRestartVersion(t *testing.T) {
	assert := asserts.New(t)
	oamcore.AddToScheme(k8scheme.Scheme)
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	reconciler := newReconciler(client)
	request := newRequest(testNamespace, testAppConfigName)

	appConfig := newAppConfig()
	appConfig.Annotations[vzconst.RestartVersionAnnotation] = "1"
	err := client.Create(context.TODO(), appConfig)
	assert.NoError(err)

	_, err = reconciler.Reconcile(request)
	assert.NoError(err)

	err = client.Get(context.TODO(), request.NamespacedName, appConfig)
	assert.NoError(err)
}

func TestReconcileEmptyRestartVersion(t *testing.T) {
	assert := asserts.New(t)
	oamcore.AddToScheme(k8scheme.Scheme)
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	reconciler := newReconciler(client)
	request := newRequest(testNamespace, testAppConfigName)

	appConfig := newAppConfig()
	appConfig.Annotations[vzconst.RestartVersionAnnotation] = ""
	err := client.Create(context.TODO(), appConfig)
	assert.NoError(err)

	_, err = reconciler.Reconcile(request)
	assert.NoError(err)

	err = client.Get(context.TODO(), request.NamespacedName, appConfig)
	assert.NoError(err)
}

const weblogicWorkload = `
{
   "kind": "VerrazzanoWebLogicWorkload"
}
`

const coherenceWorkload = `
{
   "kind": "VerrazzanoCoherenceWorkload"
}
`

const helidonWorkload = `
{
   "kind": "VerrazzanoHelidonWorkload"
}
`

const deploymentWorkload = `
{
   "kind": "Deployment",
   "metadata": {
      "name": "test-deployment"
   }
}
`

const daemonsetWorkload = `
{
   "kind": "DaemonSet",
   "metadata": {
      "name": "test-daemonset"
   }
}
`

const statefulsetWorkload = `
{
   "kind": "StatefulSet",
   "metadata": {
      "name": "test-statefulset"
   }
}
`

func TestReconcileRestartWeblogic(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			component := oamv1.ApplicationConfigurationComponent{ComponentName: testComponentName}
			appConfig.Spec.Components = []oamv1.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the component
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1.Component) error {
			component.Spec.Workload = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(weblogicWorkload, " ", ""), "\n", ""))}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
			return nil
		})

	// expect a call to update the workload
	cli.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, component *unstructured.Unstructured) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func TestReconcileRestartCoherence(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			component := oamv1.ApplicationConfigurationComponent{ComponentName: testComponentName}
			appConfig.Spec.Components = []oamv1.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the component
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1.Component) error {
			component.Spec.Workload = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(coherenceWorkload, " ", ""), "\n", ""))}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
			return nil
		})

	// expect a call to update the workload
	cli.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, component *unstructured.Unstructured) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func TestReconcileRestartHelidon(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			component := oamv1.ApplicationConfigurationComponent{ComponentName: testComponentName}
			appConfig.Spec.Components = []oamv1.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the component
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1.Component) error {
			component.Spec.Workload = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(helidonWorkload, " ", ""), "\n", ""))}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
			return nil
		})

	// expect a call to update the workload
	cli.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, component *unstructured.Unstructured) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func TestReconcileDeploymentRestart(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			component := oamv1.ApplicationConfigurationComponent{ComponentName: testComponentName}
			appConfig.Spec.Components = []oamv1.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the component
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1.Component) error {
			component.Spec.Workload = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(deploymentWorkload, " ", ""), "\n", ""))}
			return nil
		})
	// expect a call to fetch the deployment
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testDeploymentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *appsv1.Deployment) error {
			deploy.Name = testDeploymentName
			deploy.Namespace = testNamespace
			return nil
		})
	// expect a call to fetch the deployment
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testDeploymentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *appsv1.Deployment) error {
			deploy.Name = testDeploymentName
			deploy.Namespace = testNamespace
			return nil
		})
	// expect a call to update the deployment
	cli.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, deploy *appsv1.Deployment) error {
			assert.Equal(testNewRestartVersion, deploy.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation])
			return nil
		})
	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func TestReconcileDeploymentNoRestart(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			component := oamv1.ApplicationConfigurationComponent{ComponentName: testComponentName}
			appConfig.Spec.Components = []oamv1.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the component
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1.Component) error {
			component.Spec.Workload = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(deploymentWorkload, " ", ""), "\n", ""))}
			return nil
		})
	// expect a call to fetch the deployment
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testDeploymentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *appsv1.Deployment) error {
			deploy.Name = testDeploymentName
			deploy.Namespace = testNamespace
			deploy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			deploy.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = testNewRestartVersion
			return nil
		})
	// expect a call to fetch the deployment
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testDeploymentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *appsv1.Deployment) error {
			deploy.Name = testDeploymentName
			deploy.Namespace = testNamespace
			deploy.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			deploy.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = testNewRestartVersion
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func TestReconcileDaemonSetRestartDaemonSet(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			component := oamv1.ApplicationConfigurationComponent{ComponentName: testComponentName}
			appConfig.Spec.Components = []oamv1.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the component
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1.Component) error {
			component.Spec.Workload = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(daemonsetWorkload, " ", ""), "\n", ""))}
			return nil
		})
	// expect a call to fetch the daemonset
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testDaemonSetName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, daemonset *appsv1.DaemonSet) error {
			daemonset.Name = testDaemonSetName
			daemonset.Namespace = testNamespace
			return nil
		})
	// expect a call to fetch the daemonset
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testDaemonSetName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, daemonset *appsv1.DaemonSet) error {
			daemonset.Name = testDaemonSetName
			daemonset.Namespace = testNamespace
			return nil
		})
	// expect a call to update the daemonset
	cli.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, daemonset *appsv1.DaemonSet) error {
			assert.Equal(testNewRestartVersion, daemonset.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation])
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func TestReconcileDaemonSetNoRestartDaemonSet(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			component := oamv1.ApplicationConfigurationComponent{ComponentName: testComponentName}
			appConfig.Spec.Components = []oamv1.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the component
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1.Component) error {
			component.Spec.Workload = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(daemonsetWorkload, " ", ""), "\n", ""))}
			return nil
		})
	// expect a call to fetch the daemonset
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testDaemonSetName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, daemonset *appsv1.DaemonSet) error {
			daemonset.Name = testDaemonSetName
			daemonset.Namespace = testNamespace
			daemonset.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			daemonset.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = testNewRestartVersion
			return nil
		})
	// expect a call to fetch the daemonset
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testDaemonSetName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, daemonset *appsv1.DaemonSet) error {
			daemonset.Name = testDaemonSetName
			daemonset.Namespace = testNamespace
			daemonset.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			daemonset.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = testNewRestartVersion
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func TestReconcileStatefulSetRestart(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			component := oamv1.ApplicationConfigurationComponent{ComponentName: testComponentName}
			appConfig.Spec.Components = []oamv1.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the component
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1.Component) error {
			component.Spec.Workload = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(statefulsetWorkload, " ", ""), "\n", ""))}
			return nil
		})
	// expect a call to fetch the statefulset
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testStatefulSetName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, statefulset *appsv1.StatefulSet) error {
			statefulset.Name = testStatefulSetName
			statefulset.Namespace = testNamespace
			return nil
		})
	// expect a call to fetch the statefulset
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testStatefulSetName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, statefulset *appsv1.StatefulSet) error {
			statefulset.Name = testStatefulSetName
			statefulset.Namespace = testNamespace
			return nil
		})
	// expect a call to update the statefulset
	cli.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, statefulset *appsv1.StatefulSet) error {
			assert.Equal(testNewRestartVersion, statefulset.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation])
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func TestReconcileStatefulSetNoRestart(t *testing.T) {
	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			component := oamv1.ApplicationConfigurationComponent{ComponentName: testComponentName}
			appConfig.Spec.Components = []oamv1.ApplicationConfigurationComponent{component}
			return nil
		})
	// expect a call to fetch the component
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testComponentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *oamv1.Component) error {
			component.Spec.Workload = runtime.RawExtension{Raw: []byte(strings.ReplaceAll(strings.ReplaceAll(statefulsetWorkload, " ", ""), "\n", ""))}
			return nil
		})
	// expect a call to fetch the statefulset
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testStatefulSetName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, statefulset *appsv1.StatefulSet) error {
			statefulset.Name = testStatefulSetName
			statefulset.Namespace = testNamespace
			statefulset.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			statefulset.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = testNewRestartVersion
			return nil
		})
	// expect a call to fetch the statefulset
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testStatefulSetName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, statefulset *appsv1.StatefulSet) error {
			statefulset.Name = testStatefulSetName
			statefulset.Namespace = testNamespace
			statefulset.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
			statefulset.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation] = testNewRestartVersion
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}
