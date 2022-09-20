// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appconfig

import (
	"context"
	"fmt"
	"testing"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	oamcore "github.com/crossplane/oam-kubernetes-runtime/apis/core"
	oamv1 "github.com/crossplane/oam-kubernetes-runtime/apis/core/v1alpha2"
	"github.com/golang/mock/gomock"
	"github.com/prometheus/client_golang/prometheus/testutil"
	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/metricsexporter"
	"github.com/verrazzano/verrazzano/application-operator/mocks"
	vzconst "github.com/verrazzano/verrazzano/pkg/constants"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	_ = oamcore.AddToScheme(scheme)
	return scheme
}

// newReconciler creates a new reconciler for testing
func newReconciler(c client.Client) Reconciler {
	return Reconciler{
		Client: c,
		Log:    zap.S().With("test"),
		Scheme: newScheme(),
	}
}

// newRequest creates a new reconciler request for testing
func newRequest(namespace string, name string) ctrl.Request {
	return ctrl.Request{
		NamespacedName: types.NamespacedName{
			Namespace: namespace,
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
	_ = oamcore.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	reconciler := newReconciler(c)
	request := newRequest(testNamespace, testAppConfigName)

	_, err := reconciler.Reconcile(context.TODO(), request)
	assert.NoError(err)
}
func TestReconcileNoRestartVersion(t *testing.T) {

	assert := asserts.New(t)
	_ = oamcore.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	reconciler := newReconciler(c)
	request := newRequest(testNamespace, testAppConfigName)

	err := c.Create(context.TODO(), newAppConfig())
	assert.NoError(err)

	_, err = reconciler.Reconcile(context.TODO(), request)
	assert.NoError(err)
}

func TestReconcileRestartVersion(t *testing.T) {

	assert := asserts.New(t)
	_ = oamcore.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	reconciler := newReconciler(c)
	request := newRequest(testNamespace, testAppConfigName)

	appConfig := newAppConfig()
	appConfig.Annotations[vzconst.RestartVersionAnnotation] = "1"
	err := c.Create(context.TODO(), appConfig)
	assert.NoError(err)

	_, err = reconciler.Reconcile(context.TODO(), request)
	assert.NoError(err)

	err = c.Get(context.TODO(), request.NamespacedName, appConfig)
	assert.NoError(err)
}

func TestReconcileEmptyRestartVersion(t *testing.T) {

	assert := asserts.New(t)
	_ = oamcore.AddToScheme(k8scheme.Scheme)
	c := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	reconciler := newReconciler(c)
	request := newRequest(testNamespace, testAppConfigName)

	appConfig := newAppConfig()
	appConfig.Annotations[vzconst.RestartVersionAnnotation] = ""
	err := c.Create(context.TODO(), appConfig)
	assert.NoError(err)

	_, err = reconciler.Reconcile(context.TODO(), request)
	assert.NoError(err)

	err = c.Get(context.TODO(), request.NamespacedName, appConfig)
	assert.NoError(err)
}

func TestReconcileRestartWeblogic(t *testing.T) {

	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Name = testAppConfigName
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			appConfig.Status.Workloads = []oamv1.WorkloadStatus{{
				ComponentName: testComponentName,
				Reference: oamrt.TypedReference{
					APIVersion: "v1",
					Kind:       vzconst.VerrazzanoWebLogicWorkloadKind,
					Name:       testComponentName,
				},
			}}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
			return nil
		}).Times(2)

	// expect a call to update the workload
	cli.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, component *unstructured.Unstructured, options ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
			appConfig.Name = testAppConfigName
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			appConfig.Status.Workloads = []oamv1.WorkloadStatus{{
				ComponentName: testComponentName,
				Reference: oamrt.TypedReference{
					APIVersion: "v1",
					Kind:       vzconst.VerrazzanoCoherenceWorkloadKind,
					Name:       testComponentName,
				},
			}}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
			return nil
		}).Times(2)

	// expect a call to update the workload
	cli.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, component *unstructured.Unstructured, options ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
			appConfig.Name = testAppConfigName
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			appConfig.Status.Workloads = []oamv1.WorkloadStatus{{
				ComponentName: testComponentName,
				Reference: oamrt.TypedReference{
					APIVersion: "v1",
					Kind:       vzconst.VerrazzanoHelidonWorkloadKind,
					Name:       testComponentName,
				},
			}}
			return nil
		})

	// Expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
			return nil
		}).Times(2)

	// expect a call to update the workload
	cli.EXPECT().
		Update(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, component *unstructured.Unstructured, options ...client.UpdateOption) error {
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
			appConfig.Name = testAppConfigName
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			appConfig.Status.Workloads = []oamv1.WorkloadStatus{{
				ComponentName: testDeploymentName,
				Reference: oamrt.TypedReference{
					APIVersion: "v1",
					Kind:       vzconst.DeploymentWorkloadKind,
					Name:       testDeploymentName,
				},
			}}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
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
		Update(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, deploy *appsv1.Deployment, options ...client.UpdateOption) error {
			assert.Equal(testNewRestartVersion, deploy.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation])
			return nil
		})
	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

func TestFailedReconcileDeploymentRestart(t *testing.T) {

	assert := asserts.New(t)

	var mocker = gomock.NewController(t)
	var cli = mocks.NewMockClient(mocker)

	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.Namespace = testNamespace
			appConfig.Name = testAppConfigName
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			appConfig.Status.Workloads = []oamv1.WorkloadStatus{{
				ComponentName: testDeploymentName,
				Reference: oamrt.TypedReference{
					APIVersion: "v1",
					Kind:       vzconst.DeploymentWorkloadKind,
					Name:       testDeploymentName,
				},
			}}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
			return nil
		})

	// expect a call to fetch the deployment
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testDeploymentName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, deploy *appsv1.Deployment) error {
			return fmt.Errorf("Could not return %s in namespace %s", testDeploymentName, testNamespace)
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(true, result.Requeue)
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
			appConfig.Name = testAppConfigName
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			appConfig.Status.Workloads = []oamv1.WorkloadStatus{{
				ComponentName: testDeploymentName,
				Reference: oamrt.TypedReference{
					APIVersion: "v1",
					Kind:       vzconst.DeploymentWorkloadKind,
					Name:       testDeploymentName,
				},
			}}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
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
	result, err := reconciler.Reconcile(context.TODO(), request)

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
			appConfig.Name = testAppConfigName
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			appConfig.Status.Workloads = []oamv1.WorkloadStatus{{
				ComponentName: testDaemonSetName,
				Reference: oamrt.TypedReference{
					APIVersion: "v1",
					Kind:       vzconst.DaemonSetWorkloadKind,
					Name:       testDaemonSetName,
				},
			}}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
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
		Update(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, daemonset *appsv1.DaemonSet, options ...client.UpdateOption) error {
			assert.Equal(testNewRestartVersion, daemonset.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation])
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
			appConfig.Name = testAppConfigName
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			appConfig.Status.Workloads = []oamv1.WorkloadStatus{{
				ComponentName: testDaemonSetName,
				Reference: oamrt.TypedReference{
					APIVersion: "v1",
					Kind:       vzconst.DaemonSetWorkloadKind,
					Name:       testDaemonSetName,
				},
			}}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
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
	result, err := reconciler.Reconcile(context.TODO(), request)

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
			appConfig.Name = testAppConfigName
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			appConfig.Status.Workloads = []oamv1.WorkloadStatus{{
				ComponentName: testStatefulSetName,
				Reference: oamrt.TypedReference{
					APIVersion: "v1",
					Kind:       vzconst.StatefulSetWorkloadKind,
					Name:       testStatefulSetName,
				},
			}}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
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
		Update(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, statefulset *appsv1.StatefulSet, options ...client.UpdateOption) error {
			assert.Equal(testNewRestartVersion, statefulset.Spec.Template.ObjectMeta.Annotations[vzconst.RestartVersionAnnotation])
			return nil
		})

	// create a request and reconcile it
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

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
			appConfig.Name = testAppConfigName
			appConfig.Annotations = map[string]string{vzconst.RestartVersionAnnotation: testNewRestartVersion}
			appConfig.Status.Workloads = []oamv1.WorkloadStatus{{
				ComponentName: testStatefulSetName,
				Reference: oamrt.TypedReference{
					APIVersion: "v1",
					Kind:       vzconst.StatefulSetWorkloadKind,
					Name:       testStatefulSetName,
				},
			}}
			return nil
		})

	// expect a call to fetch the workload
	cli.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, component *unstructured.Unstructured) error {
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
	result, err := reconciler.Reconcile(context.TODO(), request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestReconcileKubeSystem tests to make sure we do not reconcile
// Any resource that belong to the kube-system namespace
func TestReconcileKubeSystem(t *testing.T) {

	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)

	// create a request and reconcile it
	request := newRequest(vzconst.KubeSystem, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(context.TODO(), request)

	// Validate the results
	mocker.Finish()
	assert.Nil(err)
	assert.True(result.IsZero())
}

// TestReconcileFailed tests to make sure the failure metric is being exposed
func TestReconcileFailed(t *testing.T) {

	assert := asserts.New(t)
	clientBuilder := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	// Create a request and reconcile it
	reconciler := newReconciler(clientBuilder)
	request := newRequest(testNamespace, testAppConfigName)
	reconcileerrorCounterObject, err := metricsexporter.GetSimpleCounterMetric(metricsexporter.AppconfigReconcileError)
	assert.NoError(err)
	// Expect a call to fetch the error
	reconcileFailedCounterBefore := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	reconcileerrorCounterObject.Get().Inc()
	reconciler.Reconcile(context.TODO(), request)
	reconcileFailedCounterAfter := testutil.ToFloat64(reconcileerrorCounterObject.Get())
	assert.Equal(reconcileFailedCounterBefore, reconcileFailedCounterAfter-1)
}
