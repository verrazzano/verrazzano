// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/types"

	certv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/mocks"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Needed for unit tests
var fakeOverrides []string

// helmFakeRunner is used to test helm without actually running an OS exec command
type helmFakeRunner struct {
}

const (
	testBomFilePath     = "../../testdata/test_bom.json"
	overrideJSON        = "{\"serviceAccount\": {\"create\": false}}"
	unexpectedError     = "unexpected error"
	notFoundErrorString = "not found"
)

var (
	enabled   = true
	overrides = []v1alpha1.Overrides{
		{
			Values: &apiextensionsv1.JSON{
				Raw: []byte(overrideJSON),
			},
		},
	}
	betaOverrides = []installv1beta1.Overrides{
		{
			Values: &apiextensionsv1.JSON{
				Raw: []byte(overrideJSON),
			},
		},
	}
	testNs           = "testNamespace"
	releaseName      = "v1.0_test"
	clientWithSecret = fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Secret{
			Type:       "helm.sh/release.v1",
			ObjectMeta: v1.ObjectMeta{Name: "sh.helm.release.v1." + releaseName + "."},
		},
	).Build()
	fakeContext           = spi.NewFakeContext(fake.NewClientBuilder().Build(), &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false)
	fakeContextWithSecret = spi.NewFakeContext(clientWithSecret, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false)
)

// genericHelmTestRunner is used to run generic OS commands with expected results
type genericHelmTestRunner struct {
	stdOut []byte
	stdErr []byte
	err    error
}

var testScheme = runtime.NewScheme()

func init() {
	_ = k8scheme.AddToScheme(testScheme)
	_ = v1alpha1.AddToScheme(testScheme)
	_ = certv1.AddToScheme(testScheme)
	// +kubebuilder:scaffold:testScheme
}

// Run genericHelmTestRunner executor
func (r genericHelmTestRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return r.stdOut, r.stdErr, r.err
}

// TestName tests the component name
// GIVEN a Verrazzano component
//
//	WHEN I call Name
//	THEN the correct Verrazzano name is returned
func TestName(t *testing.T) {
	comp := HelmComponent{
		ReleaseName: "release1",
	}

	a := assert.New(t)
	a.Equal("release1", comp.Name(), "Wrong component name")
}

// TestUpgrade tests the component upgrade
// GIVEN a component
//
//	WHEN I call Upgrade
//	THEN the upgrade returns success and passes the correct values to the upgrade function
func TestUpgrade(t *testing.T) {
	a := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "rancher",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ImagePullSecretKeyname:  "imagePullSecrets",
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
	}

	// This string is built from the Key:Value array returned by the bom.buildImageOverrides() function
	fakeOverrides = []string{
		"rancherImage=ghcr.io/verrazzano/rancher",
		"rancherImageTag=v2.5.7-20210407205410-1c7b39d0c",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	SetUpgradeFunc(fakeUpgrade)
	defer SetDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	err := comp.Upgrade(spi.NewFakeContext(newFakeClient(), &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false))
	a.NoError(err, "Upgrade returned an error")
}

func newFakeClient() clipkg.Client {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects(
		&corev1.Secret{ObjectMeta: v1.ObjectMeta{Name: constants.GlobalImagePullSecName, Namespace: "default"}},
	).Build()
	return client
}

// TestUpgradeIsInstalledUnexpectedError tests the component upgrade
// GIVEN a component
//
//	WHEN I call Upgrade and the chart status function returns an error
//	THEN the upgrade returns an error
func TestUpgradeIsInstalledUnexpectedError(t *testing.T) {
	a := assert.New(t)

	comp := HelmComponent{}

	SetUpgradeFunc(func(_ vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides []helm.HelmOverrides) (stdout []byte, stderr []byte, err error) {
		return nil, nil, nil
	})
	defer SetDefaultUpgradeFunc()

	helm.SetCmdRunner(genericHelmTestRunner{
		stdOut: []byte(""),
		stdErr: []byte("What happened?"),
		err:    fmt.Errorf("Unexpected error"),
	})
	defer helm.SetDefaultRunner()

	err := comp.Upgrade(spi.NewFakeContext(nil, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false))
	a.Error(err)
}

// TestUpgradeReleaseNotInstalled tests the component upgrade
// GIVEN a component
//
//	WHEN I call Upgrade and the chart is not installed
//	THEN the upgrade returns no error
func TestUpgradeReleaseNotInstalled(t *testing.T) {
	a := assert.New(t)

	comp := HelmComponent{}

	SetUpgradeFunc(func(_ vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides []helm.HelmOverrides) (stdout []byte, stderr []byte, err error) {
		return nil, nil, nil
	})
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	config.SetDefaultBomFilePath(testBomFilePath)
	defer config.SetDefaultBomFilePath("")

	err := comp.Upgrade(spi.NewFakeContext(newFakeClient(), &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false))
	a.NoError(err)
}

// TestUpgradeWithEnvOverrides tests the component upgrade
// GIVEN a component
//
//	WHEN I call Upgrade when the registry and repo overrides are set
//	THEN the upgrade returns success and passes the correct values to the upgrade function
func TestUpgradeWithEnvOverrides(t *testing.T) {
	a := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "rancher",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ImagePullSecretKeyname:  "imagePullSecrets",
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
		AppendOverridesFunc: func(context spi.ComponentContext, releaseName string, namespace string, chartDir string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
			return []bom.KeyValue{
				{
					Key:   "global.hub",
					Value: "myreg.io/myrepo/verrazzano",
				},
			}, nil
		},
	}

	_ = os.Setenv(constants.RegistryOverrideEnvVar, "myreg.io")
	defer func() { _ = os.Unsetenv(constants.RegistryOverrideEnvVar) }()

	_ = os.Setenv(constants.ImageRepoOverrideEnvVar, "myrepo")
	defer func() { _ = os.Unsetenv(constants.ImageRepoOverrideEnvVar) }()

	// This string is built from the Key:Value array returned by the bom.buildImageOverrides() function
	fakeOverrides = []string{
		"rancherImage=myreg.io/myrepo/verrazzano/rancher",
		"rancherImageTag=v2.5.7-20210407205410-1c7b39d0c",
		"global.hub=myreg.io/myrepo/verrazzano",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	SetUpgradeFunc(fakeUpgrade)
	defer SetDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	err := comp.Upgrade(spi.NewFakeContext(newFakeClient(), &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false))
	a.NoError(err, "Upgrade returned an error")
}

// TestInstall tests the component install
// GIVEN a component
//
//	WHEN I call Install and the chart is not installed
//	THEN the install runs and returns no error
func TestInstall(t *testing.T) {
	a := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "rancher",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
	}

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// This string is built from the Key:Value array returned by the bom.buildImageOverrides() function
	fakeOverrides = []string{
		"rancherImage=ghcr.io/verrazzano/rancher",
		"rancherImageTag=v2.5.7-20210407205410-1c7b39d0c",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	SetUpgradeFunc(fakeUpgrade)
	defer SetDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	helm.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStateFunction()
	err := comp.Install(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false))
	a.NoError(err, "Upgrade returned an error")
}

// TestInstallWithFileOverride tests the component install
// GIVEN a component
//
//	WHEN I call Install and the chart is not installed and has a custom overrides
//	THEN the overrides struct is populated correctly there is an error for trying to read a file that does not exist
func TestInstallWithAllOverride(t *testing.T) {
	a := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "rancher",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
		AppendOverridesFunc: func(context spi.ComponentContext, releaseName string, namespace string, chartDir string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
			kvs = append(kvs, bom.KeyValue{Key: "", Value: "my-overrides.yaml", IsFile: true})
			kvs = append(kvs, bom.KeyValue{Key: "setKey", Value: "setValue"})
			kvs = append(kvs, bom.KeyValue{Key: "setStringKey", Value: "setStringValue", SetString: true})
			kvs = append(kvs, bom.KeyValue{Key: "setFileKey", Value: "setFileValue", SetFile: true})
			return kvs, nil
		},
	}

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// This string is built from the Key:Value array returned by the bom.buildImageOverrides() function
	fakeOverrides = []string{
		"rancherImage=ghcr.io/verrazzano/rancher",
		"rancherImageTag=v2.5.7-20210407205410-1c7b39d0c",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()

	SetUpgradeFunc(fakeUpgrade)
	defer SetDefaultUpgradeFunc()

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	helm.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStateFunction()

	err := comp.Install(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false))
	a.Error(err, "Install did not return an open file error")
	a.Equal(err.Error(), "Could not open file setFileValue: open setFileValue: no such file or directory")
}

// TestInstallPreviousFailure tests the component install
// GIVEN a component
//
//	WHEN I call Install and the chart release is in a failed status
//	THEN the chart is uninstalled and then re-installed
func TestInstallPreviousFailure(t *testing.T) {
	a := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "rancher",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
	}

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	// This string is built from the Key:Value array returned by the bom.buildImageOverrides() function
	fakeOverrides = []string{
		"rancherImage=ghcr.io/verrazzano/rancher",
		"rancherImageTag=v2.5.7-20210407205410-1c7b39d0c",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	SetUpgradeFunc(fakeUpgrade)
	defer SetDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	helm.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusFailed, nil
	})
	defer helm.SetDefaultChartStateFunction()
	err := comp.Install(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false))
	a.NoError(err, "Upgrade returned an error")

	// When AddGlobalImagePullSecretHelmOverride fails
	mockController := gomock.NewController(t)
	fakeClient := mocks.NewMockClient(mockController)
	fakeClient.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(fmt.Errorf(unexpectedError))
	err = comp.Install(spi.NewFakeContext(fakeClient, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false))
	a.Error(err, "error during upgrade")

}

// TestInstallWithPreInstallFunc tests the component install
// GIVEN a component
//
//	WHEN I call Install and the component returns KVs from a preinstall func hook
//	THEN the chart is installed with the additional preInstall helm values
func TestInstallWithPreInstallFunc(t *testing.T) {
	a := assert.New(t)

	preInstallKVPairs := []bom.KeyValue{
		{Key: "preInstall1", Value: "value1"},
		{Key: "preInstall2", Value: "value2"},
	}

	comp := HelmComponent{
		ReleaseName:             "rancher",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		AppendOverridesFunc: func(context spi.ComponentContext, releaseName string, namespace string, chartDir string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
			return preInstallKVPairs, nil

		},
	}

	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	SetUpgradeFunc(fakeUpgrade)
	defer SetDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	helm.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStateFunction()
	err := comp.Install(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false))
	a.NoError(err, "Upgrade returned an error")
}

// TestOperatorInstallSupported tests IsOperatorInstallSupported
// GIVEN a component
//
//	WHEN I call IsOperatorInstallSupported
//	THEN the correct Value based on the component definition is returned
func TestOperatorInstallSupported(t *testing.T) {
	a := assert.New(t)

	comp := HelmComponent{
		SupportsOperatorInstall: true,
	}
	a.True(comp.IsOperatorInstallSupported())
	a.False(HelmComponent{}.IsOperatorInstallSupported())
}

// TestGetDependencies tests GetDependencies
// GIVEN a component
//
//	WHEN I call GetDependencies
//	THEN the correct Value based on the component definition is returned
func TestGetDependencies(t *testing.T) {
	a := assert.New(t)

	comp := HelmComponent{
		Dependencies: []string{"comp1", "comp2"},
	}
	a.Equal([]string{"comp1", "comp2"}, comp.GetDependencies())
	a.Nil(HelmComponent{}.GetDependencies())
}

// TestGetDependencies tests IsInstalled
// GIVEN a component
//
//	WHEN I call GetDependencies
//	THEN true is returned if it the helm release is deployed, false otherwise
func TestIsInstalled(t *testing.T) {
	a := assert.New(t)

	comp := HelmComponent{}
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()

	helm.SetCmdRunner(genericHelmTestRunner{
		stdOut: []byte(""),
		stdErr: []byte(""),
		err:    nil,
	})
	defer helm.SetDefaultRunner()
	config.SetDefaultBomFilePath(testBomFilePath)
	defer config.SetDefaultBomFilePath("")
	a.True(comp.IsInstalled(spi.NewFakeContext(nil, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false)))
	helm.SetCmdRunner(genericHelmTestRunner{
		stdOut: []byte(""),
		stdErr: []byte(""),
		err:    fmt.Errorf("Not installed"),
	})
	a.False(comp.IsInstalled(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false)))
	// when dry run is enabled
	a.True(comp.IsInstalled(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, true)))
}

// TestReady tests IsReady
// GIVEN a component
//
//	WHEN I call IsReady
//	THEN true is returned based on chart status and the status check function if defined for the component
func TestReady(t *testing.T) {
	defer helm.SetDefaultChartStatusFunction()
	defer helm.SetDefaultChartInfoFunction()
	defer helm.SetDefaultReleaseAppVersionFunction()
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	fakeCtx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false)
	dryRunCtx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, true)
	tests := []struct {
		name                string
		chartStatusFn       helm.ChartStatusFnType
		chartInfoFn         helm.ChartInfoFnType
		releaseAppVersionFn helm.ReleaseAppVersionFnType
		expectSuccess       bool
		ctx                 spi.ComponentContext
	}{
		{
			name: "IsReady when all conditions are met",
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusDeployed, nil
			},
			chartInfoFn: func(chartDir string) (helm.ChartInfo, error) {
				return helm.ChartInfo{
					AppVersion: "1.0",
				}, nil
			},
			releaseAppVersionFn: func(releaseName string, namespace string) (string, error) {
				return "1.0", nil
			},
			expectSuccess: true,
			ctx:           fakeCtx,
		},
		{
			name: "IsReady fail because chart not found",
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartNotFound, nil
			},
			chartInfoFn: func(chartDir string) (helm.ChartInfo, error) {
				return helm.ChartInfo{}, fmt.Errorf("chart not found")
			},
			releaseAppVersionFn: func(releaseName string, namespace string) (string, error) {
				return "1.0", nil
			},
			expectSuccess: false,
			ctx:           fakeCtx,
		},
		{
			name: "IsReady fail because GetReleaseAppVersion is failure",
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusFailed, nil
			},
			chartInfoFn: func(chartDir string) (helm.ChartInfo, error) {
				return helm.ChartInfo{
					AppVersion: "1.0",
				}, nil
			},
			releaseAppVersionFn: func(releaseName string, namespace string) (string, error) {
				return "", fmt.Errorf("error")
			},
			expectSuccess: false,
			ctx:           fakeCtx,
		},
		{
			name: "IsReady fail because error from getting chart status",
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return "", fmt.Errorf("Unexpected error")
			},
			chartInfoFn: func(chartDir string) (helm.ChartInfo, error) {
				return helm.ChartInfo{
					AppVersion: "1.0",
				}, nil
			},
			releaseAppVersionFn: func(releaseName string, namespace string) (string, error) {
				return "1.0", nil
			},
			expectSuccess: false,
			ctx:           fakeCtx,
		},
		{
			name: "IsReady fail because app version not matched between release and chart",
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusDeployed, nil
			},
			chartInfoFn: func(chartDir string) (helm.ChartInfo, error) {
				return helm.ChartInfo{
					AppVersion: "1.1",
				}, nil
			},
			releaseAppVersionFn: func(releaseName string, namespace string) (string, error) {
				return "1.0", nil
			},
			expectSuccess: false,
			ctx:           fakeCtx,
		},
		{
			name: "IsReady  when dry run is active",
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusDeployed, nil
			},
			chartInfoFn: func(chartDir string) (helm.ChartInfo, error) {
				return helm.ChartInfo{
					AppVersion: "1.1",
				}, nil
			},
			releaseAppVersionFn: func(releaseName string, namespace string) (string, error) {
				return "1.0", nil
			},
			expectSuccess: true,
			ctx:           dryRunCtx,
		},
	}

	a := assert.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := HelmComponent{}
			helm.SetChartStatusFunction(tt.chartStatusFn)
			helm.SetChartInfoFunction(tt.chartInfoFn)
			helm.SetReleaseAppVersionFunction(tt.releaseAppVersionFn)
			if tt.expectSuccess {
				a.True(comp.IsReady(tt.ctx))
			} else {
				a.False(comp.IsReady(tt.ctx))
			}
		})
	}
}

// TestOrganizeHelmValues tests OrganizeHelmValues
// GIVEN a key value list
//
//	WHEN I call OrganizeHelmValues
//	THEN I get a reverse list of my key value pairs as HelmComponent objects
func TestOrganizeHelmValues(t *testing.T) {
	tests := []struct {
		name              string
		kvs               []bom.KeyValue
		expectedHelmOrder []string
	}{
		{
			name:              "test empty values",
			kvs:               []bom.KeyValue{},
			expectedHelmOrder: []string{},
		},
		{
			name: "test one value",
			kvs: []bom.KeyValue{
				{
					Key:   "test1",
					Value: "expect1",
				},
			},
			expectedHelmOrder: []string{"test1=expect1"},
		},
		{
			name: "test multiple values",
			kvs: []bom.KeyValue{
				{
					Key:       "test1",
					Value:     "expect1",
					SetString: true,
				},
				{
					Key:   "test2",
					Value: "expect2",
				},
				{
					Key:     "test3",
					Value:   "expect3",
					SetFile: true,
				},
			},
			expectedHelmOrder: []string{"test3=expect3", "test2=expect2", "test1=expect1"},
		},
	}

	a := assert.New(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := HelmComponent{}
			overrides := comp.organizeHelmOverrides(tt.kvs)
			for i, override := range overrides {
				if override.SetOverrides != "" {
					a.Equal(tt.expectedHelmOrder[i], override.SetOverrides)
				}
				if override.FileOverride != "" {
					a.Equal(tt.expectedHelmOrder[i], override.FileOverride)
				}
				if override.SetStringOverrides != "" {
					a.Equal(tt.expectedHelmOrder[i], override.SetStringOverrides)
				}
				if override.SetFileOverrides != "" {
					a.Equal(tt.expectedHelmOrder[i], override.SetFileOverrides)
				}
			}
		})
	}
}

// TestFilesFromVerrazzanoHelm tests filesFromVerrazzanoHelm
// GIVEN an override list
// WHEN I call retrieveInstallOverrideResources
// THEN I get a list of key value pairs of files from the override sources
func TestFilesFromVerrazzanoHelm(t *testing.T) {

	tests := []struct {
		name                 string
		expectError          bool
		component            *HelmComponent
		additionalValues     []bom.KeyValue
		kvsLen               int
		expectedStringInFile string
	}{
		{
			name:             "test no overrides",
			expectError:      false,
			component:        &HelmComponent{},
			additionalValues: []bom.KeyValue{},
			kvsLen:           1,
		},
		{
			name:        "test append overrides",
			expectError: false,
			component: &HelmComponent{
				AppendOverridesFunc: func(_ spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
					kvs = append(kvs, bom.KeyValue{Key: "testKey1", Value: "testValue1"})
					kvs = append(kvs, bom.KeyValue{Key: "testKey2.testdir", Value: "testValue2"})
					return kvs, nil
				},
			},
			additionalValues: []bom.KeyValue{},
			kvsLen:           1,
		},
		{
			name:        "test image overrides",
			expectError: false,
			component: &HelmComponent{
				ReleaseName: "prometheus-operator",
			},
			additionalValues: []bom.KeyValue{},
			kvsLen:           1,
		},
		{
			name:        "test extra overrides",
			expectError: false,
			component:   &HelmComponent{},
			additionalValues: []bom.KeyValue{
				{Key: "test1", Value: "test1Value"},
				{Key: "test2", Value: "test2Value"},
			},
			kvsLen: 1,
		},
		{
			name:        "test file overrides",
			expectError: false,
			component: &HelmComponent{
				AppendOverridesFunc: func(_ spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
					kvs = append(kvs, bom.KeyValue{Value: "file1", IsFile: true})
					kvs = append(kvs, bom.KeyValue{Value: "file2", IsFile: true})
					return kvs, nil
				},
			},
			additionalValues: []bom.KeyValue{
				{Value: "file3", IsFile: true},
				{Value: "file4", IsFile: true},
			},
			kvsLen: 5,
		},
		{
			name:        "test file overrides boolean value with setString",
			expectError: false,
			component: &HelmComponent{
				AppendOverridesFunc: func(_ spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
					kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("extraEnv[%d].name", 0), Value: "SOME_BOOLEAN_AS_STRING"})
					kvs = append(kvs, bom.KeyValue{Key: fmt.Sprintf("extraEnv[%d].value", 0), Value: "true", SetString: true})
					return kvs, nil
				},
			},
			additionalValues:     []bom.KeyValue{},
			kvsLen:               1,
			expectedStringInFile: "value: \"true\"",
		},
		{
			name:        "test get file error",
			expectError: true,
			component:   &HelmComponent{},
			additionalValues: []bom.KeyValue{
				{Key: "key1", Value: "randomPath", SetFile: true},
			},
			kvsLen: 0,
		},
		{
			name:        "test everything",
			expectError: false,
			component: &HelmComponent{
				AppendOverridesFunc: func(_ spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
					kvs = append(kvs, bom.KeyValue{Value: "file1", IsFile: true})
					kvs = append(kvs, bom.KeyValue{Key: "key2", Value: "string2", SetString: true})
					return kvs, nil
				},
				ReleaseName: "prometheus-operator",
			},
			additionalValues: []bom.KeyValue{
				{Value: "file3", IsFile: true},
				{Key: "bomFile", Value: testBomFilePath, SetFile: true},
			},
			kvsLen: 3,
		},
	}

	a := assert.New(t)
	mock := gomock.NewController(t)
	client := mocks.NewMockClient(mock)
	config.SetDefaultBomFilePath(testBomFilePath)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verrazzano := &v1alpha1.Verrazzano{
				ObjectMeta: v1.ObjectMeta{
					Name:      "test",
					Namespace: "testns",
				},
			}

			ctx := spi.NewFakeContext(client, verrazzano, nil, false)

			kvs, err := tt.component.filesFromVerrazzanoHelm(ctx, verrazzano.Namespace, tt.additionalValues)
			a.Equal(tt.kvsLen, len(kvs))
			for _, kv := range kvs {
				a.True(kv.IsFile)
			}
			if tt.expectedStringInFile != "" {
				// assert that the given string occurs in the file generated
				content, err := os.ReadFile(kvs[0].Value)
				if err != nil {
					a.Error(err)
				} else {
					contentStr := string(content)
					a.Contains(contentStr, tt.expectedStringInFile)
				}
			}
			if tt.expectError {
				a.Error(err)
			} else {
				a.NoError(err)
			}
		})
	}
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeUpgrade(_ vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides []helm.HelmOverrides) (stdout []byte, stderr []byte, err error) {
	if releaseName != "rancher" {
		return []byte("error"), []byte(""), errors.New("Invalid release name")
	}
	if chartDir != "ChartDir" {
		return []byte("error"), []byte(""), errors.New("Invalid chart directory name")
	}
	if namespace != "chartNS" {
		return []byte("error"), []byte(""), errors.New("Invalid chart namespace")
	}

	for _, override := range overrides {
		if override.FileOverride == "" {
			return []byte("error"), []byte(""), errors.New("found empty filename or non-file override")
		}

	}
	return []byte("success"), []byte(""), nil
}

// TestHelmComponent tests all the getter function of HelmComponent e.g. Namespace, ShouldInstallBeforeUpgrade etc.
// GIVEN a HelmComponent
//
//	WHEN I call any getter function of HelmComponent
//	THEN the correct chart property is returned
func TestHelmComponent(t *testing.T) {
	compNamespace := "testNamespace"
	compJSONName := "testJsonName"
	compDependencies := []string{"Rancher"}
	compCertificates := []types.NamespacedName{
		{Namespace: "comp1NameSpace", Name: "comp1Name"},
		{Namespace: "comp2NameSpace", Name: "comp2Name"},
	}
	comVersion := "1.3.1"
	comp := HelmComponent{
		ReleaseName:               "release11",
		ChartNamespace:            compNamespace,
		InstallBeforeUpgrade:      enabled,
		JSONName:                  compJSONName,
		Dependencies:              compDependencies,
		SupportsOperatorUninstall: enabled,
		Certificates:              compCertificates,
		MinVerrazzanoVersion:      comVersion,
		SkipUpgrade:               true,
	}
	a := assert.New(t)
	a.Equal(compNamespace, comp.Namespace(), "Wrong component namespace")
	a.Equal(compJSONName, comp.GetJSONName(), "Wrong component jsonName")
	a.True(comp.ShouldInstallBeforeUpgrade(), "ShouldInstallBeforeUpgrade must be true")
	a.ElementsMatch(compDependencies, comp.Dependencies, "Wrong component dependencies")
	a.True(comp.IsOperatorUninstallSupported(), "SupportsOperatorUninstall must be true")
	a.ElementsMatch(compCertificates, comp.GetCertificateNames(nil), "Wrong component certificates")
	a.Equal(comVersion, comp.GetMinVerrazzanoVersion(), "wrong Verrazzano version")
	comp.MinVerrazzanoVersion = ""
	a.Equal(constants.VerrazzanoVersion1_0_0, comp.GetMinVerrazzanoVersion(), "wrong Verrazzano version")
	a.True(comp.IsEnabled(nil), "by default, IsEnabled must be true")
	a.True(comp.MonitorOverrides(nil), "by default, MonitorOverrides must be true")
	a.True(comp.GetSkipUpgrade(), "SkipUpgrade must be true")
}

// TestGetOverrides tests GetOverrides to fetch all the install Overrides
func TestGetOverrides(t *testing.T) {
	tests := []struct {
		name         string
		object       runtime.Object
		overrideFunc func(object runtime.Object) interface{}
		want         interface{}
	}{

		// GIVEN v1alpha1 VZ CR with no install overrides
		// WHEN GetOverrides is called
		// THEN empty list of Overrides is returned
		{
			"TestGetOverrides",
			&v1alpha1.Verrazzano{},
			nil,
			[]v1alpha1.Overrides{},
		},
		// GIVEN v1beta1 VZ CR with no install overrides
		// WHEN GetOverrides is called
		// THEN empty list of Overrides is returned
		{
			"TestGetOverridesWithV1BetaCR",
			&installv1beta1.Verrazzano{},
			nil,
			[]installv1beta1.Overrides{},
		},
		// GIVEN VZ CR with install overrides
		// WHEN GetOverrides is called
		// THEN list of overrides is returned
		{
			"TestGetOverridesWithOverrides",
			&v1alpha1.Verrazzano{},
			func(_ runtime.Object) interface{} { return overrides },
			overrides,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := &HelmComponent{}
			if tt.overrideFunc != nil {
				comp.GetInstallOverridesFunc = tt.overrideFunc
			}
			if got := comp.GetOverrides(tt.object); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetOverrides() = %v, want %v", got, tt.want)
			}
		})
	}
}

// helmFakeRunner overrides the helm run command
func (r helmFakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

func fakePreUpgrade(log vzlog.VerrazzanoLogger, client clipkg.Client, release string, namespace string, chartDir string) error {
	if release != "rancher" {
		return fmt.Errorf("Incorrect release name %s", release)
	}
	if chartDir != "ChartDir" {
		return fmt.Errorf("Incorrect chart directory %s", chartDir)
	}
	if namespace != "chartNS" {
		return fmt.Errorf("Incorrect namespace %s", namespace)
	}

	return nil
}

// TestIsAvailable tests IsAvailable to check whether a component is available for end users
func TestIsAvailable(t *testing.T) {
	deploymentName := "testDeployment"
	tests := []struct {
		name          string
		component     HelmComponent
		args          spi.ComponentContext
		wantReason    string
		wantAvailable bool
	}{
		// GIVEN Default Helm component
		// WHEN  IsAvailable is called
		// THEN true is returned if component is available
		{
			"TestIsAvailableWithNoAvailableObject",
			HelmComponent{},
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &v1alpha1.Verrazzano{}, nil, false),
			"",
			true,
		},
		// GIVEN Helm component with AvailabilityObjects
		// WHEN  IsAvailable is called
		// THEN true is returned if component is available
		{
			"TestIsAvailableWithAvailableObject",
			HelmComponent{
				AvailabilityObjects: &ready.AvailabilityObjects{DeploymentNames: []types.NamespacedName{}},
			},
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &v1alpha1.Verrazzano{}, nil, false),
			"",
			true,
		},
		// GIVEN Helm component with AvailabilityObjects
		// WHEN  IsAvailable is called
		// THEN false is returned if component is not available
		{
			"TestIsAvailableWithAvailableObject",
			HelmComponent{
				AvailabilityObjects: &ready.AvailabilityObjects{DeploymentNames: []types.NamespacedName{{Namespace: testNs, Name: deploymentName}}},
			},
			spi.NewFakeContext(fake.NewClientBuilder().Build(), &v1alpha1.Verrazzano{}, nil, false),
			fmt.Sprintf("waiting for deployment %s/%s to exist", testNs, deploymentName),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotReason, gotAvailable := tt.component.IsAvailable(tt.args)
			if gotReason != tt.wantReason {
				t.Errorf("IsAvailable() gotReason = %v, want %v", gotReason, tt.wantReason)
			}
			if gotAvailable != tt.wantAvailable {
				t.Errorf("IsAvailable() gotAvailable = %v, want %v", gotAvailable, tt.wantAvailable)
			}
		})
	}
}

// TestValidateMethods tests ValidateInstall, ValidateUpdate, ValidateInstallV1Beta1 and ValidateUpdateV1Beta1
func TestValidateMethods(t *testing.T) {
	c := HelmComponent{}
	validOverrideFunc := func(_ runtime.Object) interface{} { return overrides }
	invalidOverrideFunc := func(_ runtime.Object) interface{} { return []v1alpha1.Overrides{{}} }
	validBetaOverrideFunc := func(_ runtime.Object) interface{} { return betaOverrides }
	invalidBetaOverrideFunc := func(_ runtime.Object) interface{} { return []installv1beta1.Overrides{{}} }

	tests := []struct {
		name             string
		vz               *v1alpha1.Verrazzano
		overrideFunc     func()
		betaOverrideFunc func()
		wantErr          bool
	}{
		// GIVEN default Verrazzano CR
		// WHEN ValidateInstall, ValidateUpdate, ValidateInstallV1Beta1 and ValidateUpdateV1Beta1 are called
		// THEN no error is returned if all overrides are valid
		{
			"TestValidateMethods with valid overrides",
			&v1alpha1.Verrazzano{},
			func() {
				c.GetInstallOverridesFunc = validOverrideFunc
			},
			func() {
				c.GetInstallOverridesFunc = validBetaOverrideFunc
			},

			false,
		},
		// GIVEN default Verrazzano CR
		// WHEN ValidateInstall, ValidateUpdate, ValidateInstallV1Beta1 and ValidateUpdateV1Beta1 are called
		// THEN error is returned if overrides are invalid
		{
			"TestValidateMethods when invalid overrides",
			&v1alpha1.Verrazzano{},
			func() {
				c.GetInstallOverridesFunc = invalidOverrideFunc
			},
			func() {
				c.GetInstallOverridesFunc = invalidBetaOverrideFunc
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() { c.GetInstallOverridesFunc = nil }()
			tt.overrideFunc()
			if err := c.ValidateInstall(tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstall() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := c.ValidateUpdate(&v1alpha1.Verrazzano{}, tt.vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
			v1beta1Vz := &installv1beta1.Verrazzano{}
			err := tt.vz.ConvertTo(v1beta1Vz)
			assert.NoError(t, err)
			tt.betaOverrideFunc()
			if err := c.ValidateInstallV1Beta1(v1beta1Vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateInstallV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := c.ValidateUpdateV1Beta1(v1beta1Vz, v1beta1Vz); (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdateV1Beta1() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPreInstall test PreInstall to verify the Pre-Install process.
func TestPreInstall(t *testing.T) {
	defer helm.SetDefaultChartStatusFunction()
	tests := []struct {
		name          string
		helmComponent HelmComponent
		chartStatusFn helm.ChartStatusFnType
		ctx           spi.ComponentContext
		expectSuccess bool
	}{
		// GIVEN Helm component
		// WHEN PreInstall is called
		// THEN no error is returned if there is no error during PreInstall process
		{
			name: "TestPreInstall when no error",
			helmComponent: HelmComponent{
				ReleaseName: releaseName,
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
				PreInstallFunc: func(context spi.ComponentContext, releaseName string, namespace string, chartDir string) error {
					return nil
				},
			},
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusPendingInstall, nil
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: true,
		},
		// GIVEN Helm component
		// WHEN PreInstall is called
		// THEN no error is returned if there is no error fetching release status
		{
			name: "TestPreInstall when  error getting release status",
			helmComponent: HelmComponent{
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
				PreInstallFunc: func(context spi.ComponentContext, releaseName string, namespace string, chartDir string) error {
					return nil
				},
			},
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return "", fmt.Errorf(unexpectedError)
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: true,
		},
		// GIVEN Helm component
		// WHEN PreInstall is called
		// THEN error is returned if there is any error during PreInstallation process
		{
			name: "TestPreInstall when pre-install fails",
			helmComponent: HelmComponent{
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
				PreInstallFunc: func(context spi.ComponentContext, releaseName string, namespace string, chartDir string) error {
					return fmt.Errorf(unexpectedError)
				},
			},
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusPendingInstall, nil
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: false,
		},
	}
	a := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helm.SetChartStatusFunction(tt.chartStatusFn)
			if tt.expectSuccess {
				a.NoError(tt.helmComponent.PreInstall(tt.ctx))
			} else {
				a.Error(tt.helmComponent.PreInstall(tt.ctx))
			}
		})
	}
}

// TestPostInstall tests PostInstall to verify the PostInstall process.
func TestPostInstall(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).WithObjects().Build()
	fakeCtx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false)
	tests := []struct {
		name          string
		helmComponent HelmComponent
		ctx           spi.ComponentContext
		expectSuccess bool
	}{
		// GIVEN Helm component
		// WHEN PostInstall is called
		// THEN no error is returned if there is no error during PostInstallation process
		{
			name: "TestPostInstall when no error",
			helmComponent: HelmComponent{
				PostInstallFunc: func(context spi.ComponentContext, releaseName string, namespace string) error {
					return nil
				},
			},
			ctx:           fakeCtx,
			expectSuccess: true,
		},
		// GIVEN Helm component
		// WHEN PostInstall is called
		// THEN error is returned if there is any error during PostInstallation process
		{
			name: "TestPostInstall when postInstall error",
			helmComponent: HelmComponent{
				PostInstallFunc: func(context spi.ComponentContext, releaseName string, namespace string) error {
					return fmt.Errorf(unexpectedError)
				},
			},
			ctx:           fakeCtx,
			expectSuccess: false,
		},
		// GIVEN Helm component
		// WHEN PostInstall is called
		// THEN error is returned if associated ingresses are not present
		{
			name: "TestPostInstall when associated ingresses are not present",
			helmComponent: HelmComponent{
				PostInstallFunc: func(context spi.ComponentContext, releaseName string, namespace string) error {
					return nil
				},
				IngressNames: []types.NamespacedName{{Namespace: "ingressNs", Name: "testIngressName"}},
			},
			ctx:           fakeCtx,
			expectSuccess: false,
		},
		// GIVEN Helm component
		// WHEN PostInstall is called
		// THEN error is returned if certificates are not ready
		{
			name: "TestPostInstall when certificates not ready ",
			helmComponent: HelmComponent{
				PostInstallFunc: func(context spi.ComponentContext, releaseName string, namespace string) error {
					return nil
				},
				Certificates: []types.NamespacedName{{Namespace: "certificateNs", Name: "testCertificateName"}},
			},
			ctx:           fakeCtx,
			expectSuccess: false,
		},
	}
	a := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectSuccess {
				a.NoError(tt.helmComponent.PostInstall(tt.ctx))
			} else {
				a.Error(tt.helmComponent.PostInstall(tt.ctx))
			}
		})
	}
}

// TestPreUninstall tests PreUninstall function
func TestPreUninstall(t *testing.T) {
	defer helm.SetDefaultChartStatusFunction()
	tests := []struct {
		name          string
		helmComponent HelmComponent
		chartStatusFn helm.ChartStatusFnType
		ctx           spi.ComponentContext
		expectSuccess bool
	}{
		// GIVEN Helm component
		// WHEN PreUninstall is called and chart status is deployed
		// THEN no error is returned
		{
			name: "TestPreUninstall when no error",
			helmComponent: HelmComponent{
				ReleaseName: releaseName,
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
			},
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusDeployed, nil
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: true,
		},
		// GIVEN Helm component
		// WHEN PreUninstall is called and there is error while checking chart status
		// THEN error is logged but no error is returned
		{
			name: "TestPreUninstall when  error getting release status",
			helmComponent: HelmComponent{
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
			},
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return "", fmt.Errorf(unexpectedError)
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: true,
		},
		// GIVEN Helm component
		// WHEN PreUninstall is called and chart status is pending install
		// THEN PreUninstall process is skipped
		{
			name: "TestPreUninstall when chart status is pending install",
			helmComponent: HelmComponent{
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
			},
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusPendingInstall, nil
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: true,
		},
	}
	a := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helm.SetChartStatusFunction(tt.chartStatusFn)
			if tt.expectSuccess {
				a.NoError(tt.helmComponent.PreUninstall(tt.ctx))
			} else {
				a.Error(tt.helmComponent.PreUninstall(tt.ctx))
			}
		})
	}
}

// TestUninstall validates the uninstallation of Helm component
func TestUninstall(t *testing.T) {
	defer helm.SetDefaultChartStatusFunction()
	defer helm.SetDefaultRunner()
	dryRunCtx := spi.NewFakeContext(clientWithSecret, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, true)
	tests := []struct {
		name          string
		helmComponent HelmComponent
		helmOverride  func()
		ctx           spi.ComponentContext
		expectSuccess bool
	}{
		// GIVEN Helm component
		// WHEN Uninstall is called
		// THEN no error is returned if there is no error during process
		{
			name: "TestUninstall when no error",
			helmComponent: HelmComponent{
				ReleaseName: releaseName,
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
			},
			helmOverride: func() {
				helm.SetCmdRunner(genericHelmTestRunner{
					stdOut: []byte("namespace installed"),
					stdErr: []byte(""),
					err:    nil,
				})
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: true,
		},
		// GIVEN Helm component
		// WHEN Uninstall is called
		// THEN uninstallation is skipped if specified namespace is not found
		{
			name: "TestUninstall when namespace is not found",
			helmComponent: HelmComponent{
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
			},
			helmOverride: func() {
				helm.SetCmdRunner(genericHelmTestRunner{
					stdOut: []byte(""),
					stdErr: []byte(notFoundErrorString),
					err:    fmt.Errorf(notFoundErrorString),
				})
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: true,
		},
		// GIVEN Helm component
		// WHEN Uninstall is called
		// THEN uninstallation is skipped if specified namespace is not found
		{
			name: "TestUninstall when helm uninstall fails",
			helmComponent: HelmComponent{
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
			},
			helmOverride: func() {
				helm.SetCmdRunner(genericHelmTestRunner{
					stdOut: []byte(""),
					stdErr: []byte("failed to uninstall namespace"),
					err:    fmt.Errorf("error uninstalling"),
				})
			},
			ctx:           dryRunCtx,
			expectSuccess: false,
		},
	}
	a := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.helmOverride()
			if tt.expectSuccess {
				a.NoError(tt.helmComponent.Uninstall(tt.ctx))
			} else {
				a.Error(tt.helmComponent.Uninstall(tt.ctx))
			}
		})
	}
}

// TestPostUninstall to test PostUninstall function
// GIVEN default Helm component
// WHEN PostUninstall is called
// THEN always nil is returned
func TestPostUninstall(t *testing.T) {
	tests := []struct {
		name    string
		context spi.ComponentContext
		wantErr bool
	}{
		{
			"TestPostUninstall",
			fakeContext,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{}
			if err := h.PostUninstall(tt.context); (err != nil) != tt.wantErr {
				t.Errorf("PostUninstall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPreUpgrade tests the PreUpgrade process
func TestPreUpgrade(t *testing.T) {
	tests := []struct {
		name          string
		helmComponent HelmComponent
		chartStatusFn helm.ChartStatusFnType
		ctx           spi.ComponentContext
		expectSuccess bool
	}{
		// GIVEN Helm component
		// WHEN PreUpgrade is called and chart status is not Deployed
		// THEN upgrade process get success after cleaning up latestSecrets
		{
			name: "TestPreUpgrade when no error",
			helmComponent: HelmComponent{
				ReleaseName: releaseName,
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
			},
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusPendingInstall, nil
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: true,
		},
		// GIVEN Helm component
		// WHEN PreUpgrade is called and there is error while fetching chart status
		// THEN error is returned during PreUpgrade process.
		{
			name: "TestPreUpgrade when  error getting release status",
			helmComponent: HelmComponent{
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
			},
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return "", fmt.Errorf(unexpectedError)
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: false,
		},
		// GIVEN Helm component
		// WHEN PreUpgrade is called and chart status is deployed
		// THEN error is returned during PreUpgrade process.
		{
			name: "TestPreUpgrade when chart status is deployed",
			helmComponent: HelmComponent{
				ResolveNamespaceFunc: func(ns string) string {
					return testNs
				},
			},
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusDeployed, nil
			},
			ctx:           fakeContextWithSecret,
			expectSuccess: true,
		},
	}
	a := assert.New(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helm.SetChartStatusFunction(tt.chartStatusFn)
			if tt.expectSuccess {
				a.NoError(tt.helmComponent.PreUpgrade(tt.ctx))
			} else {
				a.Error(tt.helmComponent.PreUpgrade(tt.ctx))
			}
		})
	}
}

// TestReconcile to test PostUpgrade function
// GIVEN default Helm component
// WHEN Reconcile is called
// THEN always nil is returned
func TestPostUpgrade(t *testing.T) {
	a := assert.New(t)
	a.NoError(HelmComponent{}.PostUpgrade(nil))
}

// TestReconcile to test Reconcile function
// GIVEN default Helm component
// WHEN Reconcile is called
// THEN always nil is returned
func TestReconcile(t *testing.T) {
	a := assert.New(t)
	a.NoError(HelmComponent{}.Reconcile(nil))
}

// TestGetInstallArgs tests GetInstallArgs to check list of install args are returned as Helm value pairs
// GIVEN v1alpha1 install arguments
// WHEN GetInstallArgs is called
// THEN list of install args are returned as Helm value pairs
func TestGetInstallArgs(t *testing.T) {
	installArgName := "testInstallName"
	installArgName2 := "testInstallName2"
	installArgValue := "testValue"
	installArgValue1 := "value1"
	expectedInstallArgs := []bom.KeyValue{
		{
			Key:       installArgName,
			SetString: true,
			Value:     installArgValue,
		},
		{
			Key:       fmt.Sprintf("%s[%d]", installArgName2, 0),
			SetString: false,
			Value:     installArgValue1,
		},
	}

	tests := []struct {
		name        string
		installArgs []v1alpha1.InstallArgs
		want        []bom.KeyValue
	}{
		{
			name: "TestGetInstallArgs",
			installArgs: []v1alpha1.InstallArgs{
				{
					Name:      installArgName,
					SetString: true,
					Value:     installArgValue,
					ValueList: []string{},
				},
				{
					Name:      installArgName2,
					SetString: true,
					Value:     "",
					ValueList: []string{installArgValue1},
				},
			},
			want: expectedInstallArgs,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetInstallArgs(tt.installArgs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInstallArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHelmComponentUpgrade tests Upgrade to check upgrade of the helm component
func TestHelmComponentUpgrade(t *testing.T) {
	mock := gomock.NewController(t)
	mockClient := mocks.NewMockClient(mock)
	mockClient.EXPECT().
		Get(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil())).Return(fmt.Errorf(unexpectedError))

	tests := []struct {
		name          string
		helmComponent HelmComponent
		context       spi.ComponentContext
		helmOverride  func()
		wantErr       bool
	}{
		// GIVEN Helm component with SkipUpgrade set to true
		// WHEN Upgrade is called
		// THEN upgrade process is skipped
		{
			"TestHelmComponentUpgrade when skipUpgrade is true",
			HelmComponent{
				SkipUpgrade: true,
			},
			fakeContext,
			nil,
			false,
		},
		// GIVEN default Helm component
		// WHEN Upgrade is called with no existing release
		// THEN upgrade process is skipped
		{
			"TestHelmComponentUpgrade when release is not found",
			HelmComponent{},
			fakeContext,
			func() {
				helm.SetCmdRunner(genericHelmTestRunner{
					stdOut: []byte(""),
					stdErr: []byte(notFoundErrorString),
					err:    fmt.Errorf(notFoundErrorString),
				})
			},
			false,
		},
		// GIVEN default Helm component
		// WHEN Upgrade is called with failed PreUpgrade function
		// THEN upgrade process throws error
		{
			"TestHelmComponentUpgrade when preUpgrade fails",
			HelmComponent{
				PreUpgradeFunc: func(log vzlog.VerrazzanoLogger, client clipkg.Client, releaseName string, namespace string, chartDir string) error {
					return fmt.Errorf(unexpectedError)
				},
			},
			fakeContext,
			func() {
				helm.SetCmdRunner(genericHelmTestRunner{
					stdOut: []byte(""),
					stdErr: []byte(""),
					err:    nil,
				})
			},
			true,
		},
		// GIVEN default Helm component with PreUpgrade function
		// WHEN Upgrade is called
		// THEN upgrade process throws error if helm command fails
		{
			"TestHelmComponentUpgrade when global image pull secret fails",
			HelmComponent{
				PreUpgradeFunc: func(log vzlog.VerrazzanoLogger, client clipkg.Client, releaseName string, namespace string, chartDir string) error {
					return nil
				},
			},
			spi.NewFakeContext(mockClient, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false),
			func() {
				helm.SetCmdRunner(genericHelmTestRunner{
					stdOut: []byte(""),
					stdErr: []byte(""),
					err:    nil,
				})
			},
			true,
		},
	}
	for _, tt := range tests {
		if tt.helmOverride != nil {
			tt.helmOverride()
		}
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.helmComponent.Upgrade(tt.context); (err != nil) != tt.wantErr {
				t.Errorf("Upgrade() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
