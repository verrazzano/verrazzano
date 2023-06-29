// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"testing"

	"github.com/stretchr/testify/assert"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/handlers/common"
	"github.com/verrazzano/verrazzano-modules/module-operator/internal/handlerspi"
	vzhelm "github.com/verrazzano/verrazzano-modules/pkg/helm"
	"github.com/verrazzano/verrazzano-modules/pkg/vzlog"
)

const (
	releaseName = "test-release"
	namespace   = "test-ns"
	moduleName  = "test-module"
)

// TestGetWorkName tests the install handler GetWorkName function
func TestGetWorkName(t *testing.T) {
	asserts := assert.New(t)
	handler := NewHandler()

	// GIVEN an install handler
	// WHEN the GetWorkName function is called
	// THEN it returns the expected work name
	workName := handler.GetWorkName()
	asserts.Equal("install", workName)
}

// TestIsWorkNeeded tests the install handler IsWorkNeeded function
func TestIsWorkNeeded(t *testing.T) {
	asserts := assert.New(t)
	handler := NewHandler()

	// GIVEN an install handler
	// WHEN the IsWorkNeeded function is called
	// THEN no error occurs and the function returns true and an empty ctrl.Result
	needed, res := handler.IsWorkNeeded(handlerspi.HandlerContext{})
	asserts.NoError(res.GetError())
	asserts.True(needed)
	asserts.Equal(result.NewResult(), res)
}

// TestPreWorkUpdateStatus tests the install handler PreWorkUpdateStatus function
func TestPreWorkUpdateStatus(t *testing.T) {
	asserts := assert.New(t)
	handler := NewHandler()

	// GIVEN an install handler
	// WHEN the PreWorkUpdateStatus function is called
	// THEN no error occurs and the function returns an empty ctrl.Result and the Module status
	// has the expected state and condition
	module := &v1alpha1.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      moduleName,
			Namespace: namespace,
		},
	}

	cli := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(module).Build()
	ctx := handlerspi.HandlerContext{
		Log:    vzlog.DefaultLogger(),
		Client: cli,
		CR:     module,
		HelmInfo: handlerspi.HelmInfo{
			HelmRelease: &handlerspi.HelmRelease{
				Name:      releaseName,
				Namespace: namespace,
			},
		},
	}

	res := handler.PreWorkUpdateStatus(ctx)
	asserts.NoError(res.GetError())
	asserts.Equal(result.NewResult(), res)
}

// TestPreWork tests the install handler PreWork function
func TestPreWork(t *testing.T) {
	asserts := assert.New(t)
	handler := NewHandler()

	// GIVEN an install handler and a Module with an empty version
	// WHEN the PreWork function is called
	// THEN no error occurs and the function returns a ctrl.Result for requeue and the Module spec version
	// has been set and the Module target namespace has been created
	const targetNamespace = "target-namespace"
	module := &v1alpha1.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      moduleName,
			Namespace: namespace,
		},
		Spec: v1alpha1.ModuleSpec{
			TargetNamespace: targetNamespace,
		},
	}

	cli := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(module).Build()
	const chartVersion = "1.2.3"
	ctx := handlerspi.HandlerContext{
		Log:    vzlog.DefaultLogger(),
		Client: cli,
		CR:     module,
		HelmInfo: handlerspi.HelmInfo{
			HelmRelease: &handlerspi.HelmRelease{
				ChartInfo: handlerspi.HelmChart{
					Version: chartVersion,
				},
			},
		},
	}

	res := handler.PreWork(ctx)
	asserts.NoError(res.GetError())
	asserts.Equal(ctrl.Result{Requeue: true, RequeueAfter: 1000000000}, res.GetCtrlRuntimeResult())

	// fetch the Module and validate that the spec version has been set
	err := cli.Get(context.TODO(), types.NamespacedName{Name: moduleName, Namespace: namespace}, module)
	asserts.NoError(err)
	asserts.Equal(chartVersion, module.Spec.Version)

	// validate that the namespace was created
	ns := &corev1.Namespace{}
	err = cli.Get(context.TODO(), types.NamespacedName{Name: targetNamespace}, ns)
	asserts.NoError(err)

	// GIVEN an install handler and a Module with a version set
	// WHEN the PreWork function is called
	// THEN no error occurs and the function returns an empty ctrl.Result
	res = handler.PreWork(ctx)
	asserts.NoError(err)
	asserts.Equal(result.NewResult(), res)
}

// TestDoWorkUpdateStatus tests the install handler DoWorkUpdateStatus function
func TestDoWorkUpdateStatus(t *testing.T) {
	asserts := assert.New(t)
	handler := NewHandler()

	// GIVEN an install handler
	// WHEN the DoWorkUpdateStatus function is called
	// THEN no error occurs and the function returns an empty ctrl.Result and the Module status
	// has the expected state and condition
	module := &v1alpha1.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      moduleName,
			Namespace: namespace,
		},
	}

	cli := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(module).Build()
	ctx := handlerspi.HandlerContext{
		Log:    vzlog.DefaultLogger(),
		Client: cli,
		CR:     module,
		HelmInfo: handlerspi.HelmInfo{
			HelmRelease: &handlerspi.HelmRelease{
				Name:      releaseName,
				Namespace: namespace,
			},
		},
	}

	res := handler.DoWorkUpdateStatus(ctx)
	asserts.NoError(res.GetError())
	asserts.Equal(result.NewResult(), res)

	// fetch the Module and validate that the condition and state are set
	err := cli.Get(context.TODO(), types.NamespacedName{Name: moduleName, Namespace: namespace}, module)
	asserts.NoError(err)
	asserts.Equal(v1alpha1.ReadyReasonInstallStarted, module.Status.Conditions[0].Reason)
}

func getChart() *chart.Chart {
	return &chart.Chart{
		Metadata: &chart.Metadata{
			APIVersion: "v1",
			Name:       "hello",
			Version:    "0.1.0",
			AppVersion: "1.0",
		},
		Templates: []*chart.File{
			{Name: "templates/hello", Data: []byte("hello: world")},
		},
	}
}

func createRelease(name string, status release.Status) *release.Release {
	now := time.Now()
	return &release.Release{
		Name:      releaseName,
		Namespace: namespace,
		Info: &release.Info{
			FirstDeployed: now,
			LastDeployed:  now,
			Status:        status,
			Description:   "Named Release Stub",
		},
		Chart: getChart(),
		Config: map[string]interface{}{
			"name1": "value1",
			"name2": "value2",
		},
		Version: 1,
	}
}

// testActionConfigWithRelease is a fake action that returns an installed Helm release
func testActionConfigWithRelease(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return vzhelm.CreateActionConfig(true, releaseName, release.StatusDeployed, log, createRelease)
}

// testActionConfigWithNoRelease is a fake action that returns an uninstalled Helm release
func testActionConfigWithNoRelease(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return vzhelm.CreateActionConfig(false, releaseName, release.StatusUninstalled, log, createRelease)
}

// TestDoWork tests the install handler DoWork function
func TestDoWork(t *testing.T) {
	asserts := assert.New(t)

	vzhelm.SetActionConfigFunction(testActionConfigWithRelease)
	defer vzhelm.SetDefaultActionConfigFunction()

	handler := NewHandler()

	// GIVEN an install handler and a Helm release that is already installed
	// WHEN the DoWork function is called
	// THEN no error occurs and the function returns an empty ctrl.Result
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	ctx := handlerspi.HandlerContext{
		Log:    vzlog.DefaultLogger(),
		Client: cli,
		CR:     &v1alpha1.Module{},
		HelmInfo: handlerspi.HelmInfo{
			HelmRelease: &handlerspi.HelmRelease{
				Name:      releaseName,
				Namespace: namespace,
			},
		},
	}

	// GIVEN an install handler and a Helm release that is not installed
	// WHEN the DoWork function is called
	// THEN no error occurs and the Helm upgrade function has been called to install the release
	var upgradeFuncCalled = false
	common.SetUpgradeFunc(func(log vzlog.VerrazzanoLogger, releaseOpts *vzhelm.HelmReleaseOpts, wait bool, dryRun bool) (*release.Release, error) {
		upgradeFuncCalled = true
		return nil, nil
	})
	defer common.ResetUpgradeFunc()
	vzhelm.SetActionConfigFunction(testActionConfigWithNoRelease)

	res := handler.DoWork(ctx)
	asserts.NoError(res.GetError())
	asserts.True(upgradeFuncCalled)
}

// TestIsWorkDone tests the install handler IsWorkDone function
func TestIsWorkDone(t *testing.T) {
	asserts := assert.New(t)

	vzhelm.SetActionConfigFunction(testActionConfigWithRelease)
	defer vzhelm.SetDefaultActionConfigFunction()

	handler := NewHandler()

	// GIVEN an install handler and no deployed workloads
	// WHEN the IsWorkDone function is called
	// THEN no error occurs and the function returns true and an empty ctrl.Result
	cli := fake.NewClientBuilder().WithScheme(newScheme()).Build()
	ctx := handlerspi.HandlerContext{
		Log:    vzlog.DefaultLogger(),
		Client: cli,
		CR:     &v1alpha1.Module{},
		HelmInfo: handlerspi.HelmInfo{
			HelmRelease: &handlerspi.HelmRelease{
				Name:      releaseName,
				Namespace: namespace,
			},
		},
	}

	done, res := handler.IsWorkDone(ctx)
	asserts.NoError(res.GetError())
	asserts.True(done)
	asserts.Equal(result.NewResult(), res)
}

// TestPostWorkUpdateStatus tests the install handler PostWorkUpdateStatus function
func TestPostWorkUpdateStatus(t *testing.T) {
	asserts := assert.New(t)

	handler := NewHandler()

	// GIVEN an install handler
	// WHEN the PostWorkUpdateStatus function is called
	// THEN no error occurs and the function returns an empty ctrl.Result
	res := handler.PostWorkUpdateStatus(handlerspi.HandlerContext{})
	asserts.NoError(res.GetError())
	asserts.Equal(result.NewResult(), res)
}

// TestPostWork tests the install handler PostWork function
func TestPostWork(t *testing.T) {
	asserts := assert.New(t)

	handler := NewHandler()

	// GIVEN an install handler
	// WHEN the PostWork function is called
	// THEN no error occurs and the function returns an empty ctrl.Result
	res := handler.PostWork(handlerspi.HandlerContext{})
	asserts.NoError(res.GetError())
	asserts.Equal(result.NewResult(), res)
}

// TestWorkCompletedUpdateStatus tests the install handler WorkCompletedUpdateStatus function
func TestWorkCompletedUpdateStatus(t *testing.T) {
	asserts := assert.New(t)
	handler := NewHandler()

	// GIVEN an install handler
	// WHEN the WorkCompletedUpdateStatus function is called
	// THEN no error occurs and the function returns an empty ctrl.Result and the Module status
	// has the expected state and condition
	module := &v1alpha1.Module{
		ObjectMeta: metav1.ObjectMeta{
			Name:      moduleName,
			Namespace: namespace,
		},
	}

	cli := fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(module).Build()
	ctx := handlerspi.HandlerContext{
		Log:    vzlog.DefaultLogger(),
		Client: cli,
		CR:     module,
		HelmInfo: handlerspi.HelmInfo{
			HelmRelease: &handlerspi.HelmRelease{
				Name:      releaseName,
				Namespace: namespace,
			},
		},
	}

	res := handler.WorkCompletedUpdateStatus(ctx)
	asserts.NoError(res.GetError())
	asserts.Equal(result.NewResult(), res)

	// fetch the Module and validate that the condition and state are set
	err := cli.Get(context.TODO(), types.NamespacedName{Name: moduleName, Namespace: namespace}, module)
	asserts.NoError(err)
	asserts.Equal(v1alpha1.ReadyReasonInstallSucceeded, module.Status.Conditions[0].Reason)
}

func newScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	return scheme
}
