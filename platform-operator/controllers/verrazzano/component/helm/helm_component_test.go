// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

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

const testBomFilePath = "../../testdata/test_bom.json"

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

// TestGetName tests the component name
// GIVEN a Verrazzano component
//
//	WHEN I call Name
//	THEN the correct Verrazzano name is returned
func TestGetName(t *testing.T) {
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

	tests := []struct {
		name                string
		chartStatusFn       helm.ChartStatusFnType
		chartInfoFn         helm.ChartInfoFnType
		releaseAppVersionFn helm.ReleaseAppVersionFnType
		expectSuccess       bool
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
		},
		{
			name: "IsReady fail because chart not found",
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartNotFound, nil
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
		},
		{
			name: "IsReady fail because chart status is failure",
			chartStatusFn: func(releaseName string, namespace string) (string, error) {
				return helm.ChartStatusFailed, nil
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
		},
	}

	a := assert.New(t)
	client := fake.NewClientBuilder().WithScheme(k8scheme.Scheme).Build()
	ctx := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, nil, false)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := HelmComponent{}
			helm.SetChartStatusFunction(tt.chartStatusFn)
			helm.SetChartInfoFunction(tt.chartInfoFn)
			helm.SetReleaseAppVersionFunction(tt.releaseAppVersionFn)
			if tt.expectSuccess {
				a.True(comp.IsReady(ctx))
			} else {
				a.False(comp.IsReady(ctx))
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
