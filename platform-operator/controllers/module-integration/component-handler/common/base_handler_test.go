// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	goerrors "errors"
	"github.com/stretchr/testify/assert"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/internal/handlerspi"
	vzhelm "github.com/verrazzano/verrazzano-modules/pkg/helm"
	"github.com/verrazzano/verrazzano-modules/pkg/vzlog"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/time"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakes "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	releaseName      = "release"
	releaseNamespace = "releaseNS"
	namespace        = "test-ns"
	moduleName       = "test-module"
	fakeError        = "fake-err"
)

type fakeHandler struct {
	BaseHandler
	*vzhelm.HelmReleaseOpts
	err   error
	ready bool
}

// TestHelmUpgradeOrInstall tests the Helm upgrade and install
// GIVEN a chart and release information
// WHEN HelmUpgradeOrInstall is called
// THEN ensure that correct parameters are passed to the upgradeFunc
func TestHelmUpgradeOrInstall(t *testing.T) {
	asserts := assert.New(t)
	tests := []struct {
		name             string
		releaseName      string
		releaseNamespace string
		chartPath        string
		chartVersion     string
		repoURL          string
		err              error
	}{
		{
			name:             "test-success",
			releaseName:      "rel1",
			releaseNamespace: "testns",
			chartPath:        "testpath",
			chartVersion:     "v1.0",
			repoURL:          "url",
		},
		{
			name:             "test-err",
			releaseName:      "rel1",
			releaseNamespace: "testns",
			chartPath:        "testpath",
			chartVersion:     "v1.0",
			repoURL:          "url",
			err:              goerrors.New("fake-error"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cli := fakes.NewClientBuilder().WithScheme(newScheme()).WithObjects().Build()
			module := &moduleapi.Module{
				ObjectMeta: metav1.ObjectMeta{
					Name:      moduleName,
					Namespace: namespace,
				},
			}

			rctx := handlerspi.HandlerContext{
				Client: cli,
				Log:    vzlog.DefaultLogger(),
				DryRun: false,
				CR:     module,
				HelmInfo: handlerspi.HelmInfo{
					HelmRelease: &handlerspi.HelmRelease{
						Name:      test.releaseName,
						Namespace: test.releaseNamespace,
						ChartInfo: handlerspi.HelmChart{
							Version: test.chartVersion,
							Path:    test.chartPath,
						},
						Repository: handlerspi.HelmChartRepository{
							URI: test.repoURL,
						},
					},
				},
			}
			defer ResetUpgradeFunc()
			h := fakeHandler{err: test.err}
			SetUpgradeFunc(h.upgradeFunc)

			result := h.HelmUpgradeOrInstall(rctx)
			asserts.Equal(test.err, result.GetError())
			asserts.Equal(test.chartPath, h.ChartPath)
			asserts.Equal(test.chartVersion, h.ChartVersion)
			asserts.Equal(test.repoURL, h.RepoURL)
			asserts.Equal(test.releaseNamespace, h.Namespace)
			asserts.Equal(test.releaseName, h.ReleaseName)
		})
	}
}

// TestCheckReleaseDeployedAndReady tests the Helm release is deployed and ready
// GIVEN a Helm release
// WHEN CheckReleaseDeployedAndReady is called
// THEN ensure that correct result is returned
func TestCheckReleaseDeployedAndReady(t *testing.T) {
	asserts := assert.New(t)
	tests := []struct {
		name             string
		releaseName      string
		releaseNamespace string
		err              error
		ready            bool
		dryRun           bool
		actionFunc       vzhelm.ActionConfigFnType
	}{
		{
			name:             "test-ready",
			releaseName:      releaseName,
			releaseNamespace: releaseNamespace,
			ready:            true,
			actionFunc:       testActionConfigWithRelease,
		},
		{
			name:             "test-not-ready",
			releaseName:      releaseName,
			releaseNamespace: releaseNamespace,
			ready:            false,
			actionFunc:       testActionConfigWithRelease,
		},
		{
			name:             "test-no-release",
			releaseName:      releaseName,
			releaseNamespace: releaseNamespace,
			ready:            false,
			actionFunc:       testActionConfigWithNoRelease,
		},
		{
			name:             "test-release-error",
			releaseName:      releaseName,
			releaseNamespace: releaseNamespace,
			ready:            false,
			err:              goerrors.New(fakeError),
			actionFunc:       testActionConfigWithReleaseError,
		},
		{
			name:             "test-dryrun",
			releaseName:      releaseName,
			releaseNamespace: releaseNamespace,
			ready:            true,
			dryRun:           true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {

			vzhelm.SetActionConfigFunction(test.actionFunc)
			defer vzhelm.SetDefaultActionConfigFunction()

			cli := fakes.NewClientBuilder().WithScheme(newScheme()).WithObjects().Build()
			rctx := handlerspi.HandlerContext{
				Client: cli,
				Log:    vzlog.DefaultLogger(),
				DryRun: test.dryRun,
				HelmInfo: handlerspi.HelmInfo{
					HelmRelease: &handlerspi.HelmRelease{
						Name:      test.releaseName,
						Namespace: test.releaseNamespace,
					},
				},
			}
			defer ResetCheckReadyFunc()
			h := fakeHandler{err: test.err, ready: test.ready}
			SetCheckReadyFunc(h.checkWorkLoadsReady)

			ready, result := h.CheckReleaseDeployedAndReady(rctx)
			asserts.Equal(test.err, result.GetError())
			asserts.Equal(test.ready, ready)
		})
	}
}

// TestBuildOverrides tests the Helm overricdes
// GIVEN module overrides
// WHEN buildOverrides is called
// THEN ensure that correct result is returned
func TestBuildOverrides(t *testing.T) {
	cmRef := &corev1.ConfigMapKeySelector{}
	secretRef := &corev1.SecretKeySelector{}

	cmRef2 := &corev1.ConfigMapKeySelector{}
	secretRef2 := &corev1.SecretKeySelector{}

	asserts := assert.New(t)
	valJSON := &apiextensionsv1.JSON{
		Raw: []byte(`{"key":"val"}`),
	}
	tests := []struct {
		name              string
		values            *apiextensionsv1.JSON
		valuesFrom        []moduleapi.ValuesFromSource
		expectedOverrides []vzhelm.ValueOverrides
	}{
		{
			name: "test-no-val",
		},
		{
			name:   "test-val",
			values: valJSON,
			expectedOverrides: []vzhelm.ValueOverrides{{
				Values: valJSON,
			}},
		},
		{
			name:   "test-val-first",
			values: valJSON,
			valuesFrom: []moduleapi.ValuesFromSource{
				{ConfigMapRef: cmRef, SecretRef: secretRef},
			},
			expectedOverrides: []vzhelm.ValueOverrides{
				{Values: valJSON},
				{ConfigMapRef: cmRef, SecretRef: secretRef},
			},
		},
		{
			name:   "test-val-first-plus-overrides",
			values: valJSON,
			valuesFrom: []moduleapi.ValuesFromSource{
				{ConfigMapRef: cmRef, SecretRef: secretRef},
				{ConfigMapRef: cmRef2, SecretRef: secretRef2},
			},
			expectedOverrides: []vzhelm.ValueOverrides{
				{Values: valJSON},
				{ConfigMapRef: cmRef, SecretRef: secretRef},
				{ConfigMapRef: cmRef2, SecretRef: secretRef2},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			module := moduleapi.Module{
				Spec: moduleapi.ModuleSpec{
					Values:     test.values,
					ValuesFrom: test.valuesFrom,
				},
			}

			ov := buildOverrides(&module)
			i := 0
			// Assert remaining values from are in order
			for _, vf := range test.valuesFrom {
				asserts.Equal(ov[i].SecretRef, vf.SecretRef)
				asserts.Equal(ov[i].ConfigMapRef, vf.ConfigMapRef)
				i++
			}
			// Assert values is last
			if test.values != nil {
				asserts.Equal(ov[i].Values, test.values)
			}

		})
	}
}

func (f *fakeHandler) upgradeFunc(log vzlog.VerrazzanoLogger, releaseOpts *vzhelm.HelmReleaseOpts, wait bool, dryRun bool) (*release.Release, error) {
	f.HelmReleaseOpts = releaseOpts
	return nil, f.err
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

func (f *fakeHandler) checkWorkLoadsReady(ctx handlerspi.HandlerContext, releaseName string, namespace string) (bool, error) {
	return f.ready, f.err

}

// testActionConfigWithRelease is a fake action that returns an installed Helm release
func testActionConfigWithRelease(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return vzhelm.CreateActionConfig(true, releaseName, release.StatusDeployed, log, createRelease)
}

// testActionConfigWithReleaseError is a fake action that returns an installed Helm release error
func testActionConfigWithReleaseError(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return nil, goerrors.New(fakeError)
}

// testActionConfigWithNoRelease is a fake action that returns an uninstalled Helm release
func testActionConfigWithNoRelease(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	return vzhelm.CreateActionConfig(false, releaseName, release.StatusUninstalled, log, createRelease)
}
