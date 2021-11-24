// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"os/exec"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
	"testing"

	clipkg "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"go.uber.org/zap"
)

// Needed for unit tests
var fakeOverrides string

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

// Run genericHelmTestRunner executor
func (r genericHelmTestRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return r.stdOut, r.stdErr, r.err
}

// TestGetName tests the component name
// GIVEN a Verrazzano component
//  WHEN I call Name
//  THEN the correct Verrazzano name is returned
func TestGetName(t *testing.T) {
	comp := HelmComponent{
		ReleaseName: "release1",
	}

	assert := assert.New(t)
	assert.Equal("release1", comp.Name(), "Wrong component name")
}

// TestUpgrade tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade
//  THEN the upgrade returns success and passes the correct values to the upgrade function
func TestUpgrade(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "rancher",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
	}

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function
	fakeOverrides = "rancherImageTag=v2.5.7-20210407205410-1c7b39d0c,rancherImage=ghcr.io/verrazzano/rancher"

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	err := comp.Upgrade(spi.NewFakeContext(nil, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false))
	assert.NoError(err, "Upgrade returned an error")
}

// TestUpgradeIsInstalledUnexpectedError tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade and the chart status function returns an error
//  THEN the upgrade returns an error
func TestUpgradeIsInstalledUnexpectedError(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{}

	setUpgradeFunc(func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides helm.HelmOverrides) (stdout []byte, stderr []byte, err error) {
		return nil, nil, nil
	})
	defer setDefaultUpgradeFunc()

	helm.SetCmdRunner(genericHelmTestRunner{
		stdOut: []byte(""),
		stdErr: []byte("What happened?"),
		err:    fmt.Errorf("Unexpected error"),
	})
	defer helm.SetDefaultRunner()

	err := comp.Upgrade(spi.NewFakeContext(nil, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false))
	assert.Error(err)
}

// TestUpgradeReleaseNotInstalled tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade and the chart is not installed
//  THEN the upgrade returns no error
func TestUpgradeReleaseNotInstalled(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{}

	setUpgradeFunc(func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides helm.HelmOverrides) (stdout []byte, stderr []byte, err error) {
		return nil, nil, nil
	})
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	config.SetDefaultBomFilePath(testBomFilePath)
	defer config.SetDefaultBomFilePath("")

	err := comp.Upgrade(spi.NewFakeContext(nil, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false))
	assert.NoError(err)
}

// TestUpgradeWithEnvOverrides tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade when the registry and repo overrides are set
//  THEN the upgrade returns success and passes the correct values to the upgrade function
func TestUpgradeWithEnvOverrides(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "rancher",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
		AppendOverridesFunc:     istio.AppendIstioOverrides,
	}

	os.Setenv(constants.RegistryOverrideEnvVar, "myreg.io")
	defer os.Unsetenv(constants.RegistryOverrideEnvVar)

	os.Setenv(constants.ImageRepoOverrideEnvVar, "myrepo")
	defer os.Unsetenv(constants.ImageRepoOverrideEnvVar)

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function
	fakeOverrides = "rancherImageTag=v2.5.7-20210407205410-1c7b39d0c,rancherImage=myreg.io/myrepo/verrazzano/rancher,global.hub=myreg.io/myrepo/verrazzano"

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	err := comp.Upgrade(spi.NewFakeContext(nil, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false))
	assert.NoError(err, "Upgrade returned an error")
}

// TestInstall tests the component install
// GIVEN a component
//  WHEN I call Install and the chart is not installed
//  THEN the install runs and returns no error
func TestInstall(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "rancher",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function
	fakeOverrides = "rancherImageTag=v2.5.7-20210407205410-1c7b39d0c,rancherImage=ghcr.io/verrazzano/rancher"

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	helm.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStateFunction()
	err := comp.Install(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false))
	assert.NoError(err, "Upgrade returned an error")
}

// TestInstallWithFileOverride tests the component install
// GIVEN a component
//  WHEN I call Install and the chart is not installed and has a custom overrides
//  THEN the overrides struct is populated correctly and there are no errors
func TestInstallWithAllOverride(t *testing.T) {
	assert := assert.New(t)

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

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function
	fakeOverrides = "rancherImageTag=v2.5.7-20210407205410-1c7b39d0c,rancherImage=ghcr.io/verrazzano/rancher"

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()

	setUpgradeFunc(func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides helm.HelmOverrides) (stdout []byte, stderr []byte, err error) {
		assert.Contains(overrides.FileOverrides, "my-overrides.yaml", "Overrides file not found")
		assert.Contains(overrides.SetOverrides, "setKey=setValue", "Incorrect --set overrides")
		assert.Contains(overrides.SetStringOverrides, "setStringKey=setStringValue", "Incorrect --set overrides")
		assert.Contains(overrides.SetFileOverrides, "setFileKey=setFileValue", "Incorrect --set overrides")
		return fakeUpgrade(log, releaseName, namespace, chartDir, wait, dryRun, overrides)
	})
	defer setDefaultUpgradeFunc()

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()

	helm.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStateFunction()

	err := comp.Install(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false))
	assert.NoError(err, "Install returned an error")
}

// TestInstallPreviousFailure tests the component install
// GIVEN a component
//  WHEN I call Install and the chart release is in a failed status
//  THEN the chart is uninstalled and then re-installed
func TestInstallPreviousFailure(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "rancher",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function
	fakeOverrides = "rancherImageTag=v2.5.7-20210407205410-1c7b39d0c,rancherImage=ghcr.io/verrazzano/rancher"

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	helm.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusFailed, nil
	})
	defer helm.SetDefaultChartStateFunction()
	err := comp.Install(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false))
	assert.NoError(err, "Upgrade returned an error")
}

// TestInstallWithPreInstallFunc tests the component install
// GIVEN a component
//  WHEN I call Install and the component returns KVs from a preinstall func hook
//  THEN the chart is installed with the additional preInstall helm values
func TestInstallWithPreInstallFunc(t *testing.T) {
	assert := assert.New(t)

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

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function,
	// plus values returned from the preInstall function if present
	var buffer bytes.Buffer
	buffer.WriteString("rancherImageTag=v2.5.7-20210407205410-1c7b39d0c,rancherImage=ghcr.io/verrazzano/rancher,")
	for i, kv := range preInstallKVPairs {
		buffer.WriteString(kv.Key)
		buffer.WriteString("=")
		buffer.WriteString(kv.Value)
		if i != len(preInstallKVPairs)-1 {
			buffer.WriteString(",")
		}
	}
	expectedOverridesString := buffer.String()

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides helm.HelmOverrides) (stdout []byte, stderr []byte, err error) {
		if overrides.SetOverrides != expectedOverridesString {
			return nil, nil, fmt.Errorf("Unexpected overrides string %s, expected %s", overrides, expectedOverridesString)
		}
		return []byte{}, []byte{}, nil
	})
	defer setDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	helm.SetChartStateFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStateFunction()
	err := comp.Install(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false))
	assert.NoError(err, "Upgrade returned an error")
}

// TestOperatorInstallSupported tests IsOperatorInstallSupported
// GIVEN a component
//  WHEN I call IsOperatorInstallSupported
//  THEN the correct Value based on the component definition is returned
func TestOperatorInstallSupported(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{
		SupportsOperatorInstall: true,
	}
	assert.True(comp.IsOperatorInstallSupported())
	assert.False(HelmComponent{}.IsOperatorInstallSupported())
}

// TestGetDependencies tests GetDependencies
// GIVEN a component
//  WHEN I call GetDependencies
//  THEN the correct Value based on the component definition is returned
func TestGetDependencies(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{
		Dependencies: []string{"comp1", "comp2"},
	}
	assert.Equal([]string{"comp1", "comp2"}, comp.GetDependencies())
	assert.Nil(HelmComponent{}.GetDependencies())
}

// TestGetDependencies tests IsInstalled
// GIVEN a component
//  WHEN I call GetDependencies
//  THEN true is returned if it the helm release is deployed, false otherwise
func TestIsInstalled(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{}
	defer helm.SetDefaultChartStatusFunction()
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	helm.SetCmdRunner(genericHelmTestRunner{
		stdOut: []byte(""),
		stdErr: []byte(""),
		err:    nil,
	})
	defer helm.SetDefaultRunner()
	config.SetDefaultBomFilePath(testBomFilePath)
	defer config.SetDefaultBomFilePath("")
	assert.True(comp.IsInstalled(spi.NewFakeContext(nil, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false)))
	helm.SetCmdRunner(genericHelmTestRunner{
		stdOut: []byte(""),
		stdErr: []byte(""),
		err:    fmt.Errorf("Not installed"),
	})
	assert.False(comp.IsInstalled(spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false)))
}

// TestReady tests IsReady
// GIVEN a component
//  WHEN I call IsReady
//  THEN true is returned based on chart status and the status check function if defined for the component
func TestReady(t *testing.T) {
	assert := assert.New(t)

	defer helm.SetDefaultChartStatusFunction()

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	comp := HelmComponent{}
	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)
	compContext := spi.NewFakeContext(client, &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}, false)
	//compContext := spi.ComponentContext{Log: zap.S(), Client: client, EffectiveCR: &v1alpha1.Verrazzano{ObjectMeta: v1.ObjectMeta{Namespace: "foo"}}}

	assert.True(comp.IsReady(compContext))

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	assert.False(comp.IsReady(compContext))

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusFailed, nil
	})
	assert.False(comp.IsReady(compContext))

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return "", fmt.Errorf("Unexpected error")
	})
	assert.False(comp.IsReady(compContext))

	compInstalledWithNotReadyStatus := HelmComponent{
		ReadyStatusFunc: func(ctx spi.ComponentContext, releaseName string, namespace string) bool {
			return false
		},
	}
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	assert.False(compInstalledWithNotReadyStatus.IsReady(compContext))

	compInstalledWithReadyStatus := HelmComponent{
		ReadyStatusFunc: func(ctx spi.ComponentContext, releaseName string, namespace string) bool {
			return true
		},
	}
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	assert.True(compInstalledWithReadyStatus.IsReady(compContext))
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeUpgrade(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides helm.HelmOverrides) (stdout []byte, stderr []byte, err error) {
	if releaseName != "rancher" {
		return []byte("error"), []byte(""), errors.New("Invalid release name")
	}
	if chartDir != "ChartDir" {
		return []byte("error"), []byte(""), errors.New("Invalid chart directory name")
	}
	if namespace != "chartNS" {
		return []byte("error"), []byte(""), errors.New("Invalid chart namespace")
	}

	foundChartOverridesFile := false
	for _, file := range overrides.FileOverrides {
		if file == "ValuesFile" {
			foundChartOverridesFile = true
			break
		}
	}
	if !foundChartOverridesFile {
		return []byte("error"), []byte(""), errors.New("Invalid values file")
	}

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function
	if !strings.Contains(overrides.SetOverrides, fakeOverrides) {
		return []byte("error"), []byte(""), errors.New("Invalid overrides")
	}
	return []byte("success"), []byte(""), nil
}

// helmFakeRunner overrides the helm run command
func (r helmFakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

func fakePreUpgrade(log *zap.SugaredLogger, client clipkg.Client, release string, namespace string, chartDir string) error {
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

func TestGetInstallArgs(t *testing.T) {
	type args struct {
		args []v1alpha1.InstallArgs
	}
	tests := []struct {
		name string
		args args
		want []bom.KeyValue
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetInstallArgs(tt.args.args); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInstallArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmComponent_GetDependencies(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	tests := []struct {
		name   string
		fields fields
		want   []string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if got := h.GetDependencies(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDependencies() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmComponent_GetMinVerrazzanoVersion(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if got := h.GetMinVerrazzanoVersion(); got != tt.want {
				t.Errorf("GetMinVerrazzanoVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmComponent_GetSkipUpgrade(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if got := h.GetSkipUpgrade(); got != tt.want {
				t.Errorf("GetSkipUpgrade() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmComponent_Install(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		context spi.ComponentContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if err := h.Install(tt.args.context); (err != nil) != tt.wantErr {
				t.Errorf("Install() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHelmComponent_IsEnabled(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		context spi.ComponentContext
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if got := h.IsEnabled(tt.args.context); got != tt.want {
				t.Errorf("IsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmComponent_IsInstalled(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		context spi.ComponentContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			got, err := h.IsInstalled(tt.args.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsInstalled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsInstalled() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmComponent_IsOperatorInstallSupported(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if got := h.IsOperatorInstallSupported(); got != tt.want {
				t.Errorf("IsOperatorInstallSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmComponent_IsReady(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		context spi.ComponentContext
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if got := h.IsReady(tt.args.context); got != tt.want {
				t.Errorf("IsReady() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmComponent_Name(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if got := h.Name(); got != tt.want {
				t.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmComponent_PostInstall(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		context spi.ComponentContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if err := h.PostInstall(tt.args.context); (err != nil) != tt.wantErr {
				t.Errorf("PostInstall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHelmComponent_PostUpgrade(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		context spi.ComponentContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if err := h.PostUpgrade(tt.args.context); (err != nil) != tt.wantErr {
				t.Errorf("PostUpgrade() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHelmComponent_PreInstall(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		context spi.ComponentContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if err := h.PreInstall(tt.args.context); (err != nil) != tt.wantErr {
				t.Errorf("PreInstall() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHelmComponent_PreUpgrade(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		context spi.ComponentContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if err := h.PreUpgrade(tt.args.context); (err != nil) != tt.wantErr {
				t.Errorf("PreUpgrade() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHelmComponent_Upgrade(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		context spi.ComponentContext
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if err := h.Upgrade(tt.args.context); (err != nil) != tt.wantErr {
				t.Errorf("Upgrade() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHelmComponent_buildCustomHelmOverrides(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		context          spi.ComponentContext
		namespace        string
		additionalValues []bom.KeyValue
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    helm.HelmOverrides
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			got, err := h.buildCustomHelmOverrides(tt.args.context, tt.args.namespace, tt.args.additionalValues...)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildCustomHelmOverrides() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildCustomHelmOverrides() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHelmComponent_resolveNamespace(t *testing.T) {
	type fields struct {
		ReleaseName             string
		ChartDir                string
		ChartNamespace          string
		IgnoreNamespaceOverride bool
		IgnoreImageOverrides    bool
		ValuesFile              string
		PreInstallFunc          preInstallFuncSig
		PostInstallFunc         postInstallFuncSig
		PreUpgradeFunc          preUpgradeFuncSig
		AppendOverridesFunc     appendOverridesSig
		ReadyStatusFunc         readyStatusFuncSig
		ResolveNamespaceFunc    resolveNamespaceSig
		IsEnabledFunc           isEnabledFuncSig
		SupportsOperatorInstall bool
		WaitForInstall          bool
		ImagePullSecretKeyname  string
		Dependencies            []string
		SkipUpgrade             bool
		MinVerrazzanoVersion    string
	}
	type args struct {
		ns string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := HelmComponent{
				ReleaseName:             tt.fields.ReleaseName,
				ChartDir:                tt.fields.ChartDir,
				ChartNamespace:          tt.fields.ChartNamespace,
				IgnoreNamespaceOverride: tt.fields.IgnoreNamespaceOverride,
				IgnoreImageOverrides:    tt.fields.IgnoreImageOverrides,
				ValuesFile:              tt.fields.ValuesFile,
				PreInstallFunc:          tt.fields.PreInstallFunc,
				PostInstallFunc:         tt.fields.PostInstallFunc,
				PreUpgradeFunc:          tt.fields.PreUpgradeFunc,
				AppendOverridesFunc:     tt.fields.AppendOverridesFunc,
				ReadyStatusFunc:         tt.fields.ReadyStatusFunc,
				ResolveNamespaceFunc:    tt.fields.ResolveNamespaceFunc,
				IsEnabledFunc:           tt.fields.IsEnabledFunc,
				SupportsOperatorInstall: tt.fields.SupportsOperatorInstall,
				WaitForInstall:          tt.fields.WaitForInstall,
				ImagePullSecretKeyname:  tt.fields.ImagePullSecretKeyname,
				Dependencies:            tt.fields.Dependencies,
				SkipUpgrade:             tt.fields.SkipUpgrade,
				MinVerrazzanoVersion:    tt.fields.MinVerrazzanoVersion,
			}
			if got := h.resolveNamespace(tt.args.ns); got != tt.want {
				t.Errorf("resolveNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getImageOverrides(t *testing.T) {
	type args struct {
		subcomponentName string
	}
	tests := []struct {
		name    string
		args    args
		want    []bom.KeyValue
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getImageOverrides(tt.args.subcomponentName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getImageOverrides() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getImageOverrides() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_setDefaultUpgradeFunc(t *testing.T) {
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func Test_setUpgradeFunc(t *testing.T) {
	type args struct {
		f upgradeFuncSig
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
