// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package appconfig

import (
	"context"
	"fmt"
	"testing"
	"time"

	oamrt "github.com/crossplane/crossplane-runtime/apis/core/v1alpha1"
	certapiv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	"github.com/verrazzano/verrazzano/application-operator/constants"
	"go.uber.org/zap"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

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

	// Expect a call to update the app config resource with a finalizer.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 1)
			assert.Equal(finalizerName, appConfig.Finalizers[0])
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

	// Expect a call to update the app config resource with a finalizer.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 1)
			assert.Equal(finalizerName, appConfig.Finalizers[0])
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

	// Expect a call to update the app config resource with a finalizer.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 1)
			assert.Equal(finalizerName, appConfig.Finalizers[0])
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

	// Expect a call to update the app config resource with a finalizer.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 1)
			assert.Equal(finalizerName, appConfig.Finalizers[0])
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

	// Expect a call to update the app config resource with a finalizer.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 1)
			assert.Equal(finalizerName, appConfig.Finalizers[0])
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
	result, err := reconciler.Reconcile(request)

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

	// Expect a call to update the app config resource with a finalizer.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 1)
			assert.Equal(finalizerName, appConfig.Finalizers[0])
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

	// Expect a call to update the app config resource with a finalizer.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 1)
			assert.Equal(finalizerName, appConfig.Finalizers[0])
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

	// Expect a call to update the app config resource with a finalizer.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 1)
			assert.Equal(finalizerName, appConfig.Finalizers[0])
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

	// Expect a call to update the app config resource with a finalizer.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 1)
			assert.Equal(finalizerName, appConfig.Finalizers[0])
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

	// Expect a call to update the app config resource with a finalizer.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 1)
			assert.Equal(finalizerName, appConfig.Finalizers[0])
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
	result, err := reconciler.Reconcile(request)

	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
}

// TestDeleteTraitResourcesWhenAppConfigIsDeleted tests the Reconcile method for the following use case.
// GIVEN a request to reconcile an app config resource that is marked for deletion
// WHEN the app config exists
// THEN ensure that the cert and secret trait resources associated with the app config are also deleted
func TestDeleteCertAndSecretWhenAppConfigIsDeleted(t *testing.T) {
	assert := asserts.New(t)
	mocker := gomock.NewController(t)
	cli := mocks.NewMockClient(mocker)
	// expect a call to fetch the ApplicationConfiguration
	cli.EXPECT().
		Get(gomock.Any(), types.NamespacedName{Namespace: testNamespace, Name: testAppConfigName}, gomock.Not(gomock.Nil())).
		DoAndReturn(func(ctx context.Context, name types.NamespacedName, appConfig *oamv1.ApplicationConfiguration) error {
			appConfig.ObjectMeta = ctrl.ObjectMeta{
				Namespace:         testNamespace,
				Name:              testAppConfigName,
				Finalizers:        []string{finalizerName},
				DeletionTimestamp: &v1.Time{Time: time.Now()}}
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
	// Expect a call to delete the cert
	cli.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, cert *certapiv1.Certificate, opt *client.DeleteOptions) error {
			assert.Equal(constants.IstioSystemNamespace, cert.Namespace)
			assert.Equal(fmt.Sprintf("%s-%s-cert", testNamespace, testAppConfigName), cert.Name)
			return nil
		})
	// Expect a call to delete the secret
	cli.EXPECT().
		Delete(gomock.Any(), gomock.Not(gomock.Nil()), gomock.Any()).
		DoAndReturn(func(ctx context.Context, sec *k8score.Secret, opt *client.DeleteOptions) error {
			assert.Equal(constants.IstioSystemNamespace, sec.Namespace)
			assert.Equal(fmt.Sprintf("%s-%s-cert-secret", testNamespace, testAppConfigName), sec.Name)
			return nil
		})

	// Expect a call to update the app config resource with the finalizer removed.
	cli.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, appConfig *oamv1.ApplicationConfiguration) error {
			assert.Equal(testNamespace, appConfig.Namespace)
			assert.Equal(testAppConfigName, appConfig.Name)
			assert.Len(appConfig.Finalizers, 0)
			return nil
		})

	// Create and make the request
	request := newRequest(testNamespace, testAppConfigName)
	reconciler := newReconciler(cli)
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	assert.NoError(err)
	assert.Equal(false, result.Requeue)
	assert.Equal(time.Duration(0), result.RequeueAfter)
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
	result, err := reconciler.Reconcile(request)

	// Validate the results
	mocker.Finish()
	assert.Nil(err)
	assert.True(result.IsZero())
}
