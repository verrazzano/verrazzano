// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	k8scheme "k8s.io/client-go/kubernetes/scheme"
	"os"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	clipkg "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/helm"
	"go.uber.org/zap"
)

// Needed for unit tests
var fakeOverrides string

var vz = &installv1alpha1.Verrazzano{
	Spec: installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Istio: &installv1alpha1.IstioComponent{
				IstioInstallArgs: []installv1alpha1.InstallArgs{},
			},
		},
	},
}

// helmFakeRunner is used to test helm without actually running an OS exec command
type helmFakeRunner struct {
}

const testBomFilePath = "../../testdata/test_bom.json"

// TestGetName tests the component name
// GIVEN a Verrazzano component
//  WHEN I call Name
//  THEN the correct verrazzano name is returned
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
		ReleaseName:             "istiod",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
	}

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function
	fakeOverrides = "pilot.image=ghcr.io/verrazzano/pilot:1.7.3,global.proxy.image=proxyv2,global.tag=1.7.3"

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	err := comp.Upgrade(zap.S(), vz, nil, "", false)
	assert.NoError(err, "Upgrade returned an error")
}

// TestUpgradeIsInstalledUnexpectedError tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade and the chart status function returns an error
//  THEN the upgrade returns an error
func TestUpgradeIsInstalledUnexpectedError(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{}

	setUpgradeFunc(func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides string, overrideFiles ...string) (stdout []byte, stderr []byte, err error) {
		return nil, nil, nil
	})
	defer setDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return "", fmt.Errorf("Unexpected error")
	})
	defer helm.SetDefaultChartStatusFunction()
	err := comp.Upgrade(zap.S(), vz, nil, "", false)
	assert.Error(err)
}

// TestUpgradeReleaseNotInstalled tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade and the chart is not installed
//  THEN the upgrade returns no error
func TestUpgradeReleaseNotInstalled(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{}

	setUpgradeFunc(func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides string, overrideFiles ...string) (stdout []byte, stderr []byte, err error) {
		return nil, nil, nil
	})
	defer setDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	err := comp.Upgrade(zap.S(), vz, nil, "", false)
	assert.NoError(err)
}

// TestUpgradeWithEnvOverrides tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade when the registry and repo overrides are set
//  THEN the upgrade returns success and passes the correct values to the upgrade function
func TestUpgradeWithEnvOverrides(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "istiod",
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
	fakeOverrides = "pilot.image=myreg.io/myrepo/verrazzano/pilot:1.7.3,global.proxy.image=proxyv2,global.tag=1.7.3,global.hub=myreg.io/myrepo/verrazzano"

	config.SetDefaultBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	defer helm.SetDefaultChartStatusFunction()
	err := comp.Upgrade(zap.S(), vz, nil, "", false)
	assert.NoError(err, "Upgrade returned an error")
}

// TestInstall tests the component install
// GIVEN a component
//  WHEN I call Install and the chart is not installed
//  THEN the install runs and returns no error
func TestInstall(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "istiod",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function
	fakeOverrides = "pilot.image=ghcr.io/verrazzano/pilot:1.7.3,global.proxy.image=proxyv2,global.tag=1.7.3"

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
	err := comp.Install(zap.S(), vz, client, "default", false)
	assert.NoError(err, "Upgrade returned an error")
}

// TestInstallPreviousFailure tests the component install
// GIVEN a component
//  WHEN I call Install and the chart release is in a failed status
//  THEN the chart is uninstalled and then re-installed
func TestInstallPreviousFailure(t *testing.T) {
	assert := assert.New(t)

	comp := HelmComponent{
		ReleaseName:             "istiod",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreUpgradeFunc:          fakePreUpgrade,
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function
	fakeOverrides = "pilot.image=ghcr.io/verrazzano/pilot:1.7.3,global.proxy.image=proxyv2,global.tag=1.7.3"

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
	err := comp.Install(zap.S(), vz, client, "default", false)
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
		ReleaseName:             "istiod",
		ChartDir:                "ChartDir",
		ChartNamespace:          "chartNS",
		IgnoreNamespaceOverride: true,
		ValuesFile:              "ValuesFile",
		PreInstallFunc: func(log *zap.SugaredLogger, client clipkg.Client, releaseName string, namespace string, chartDir string) ([]bom.KeyValue, error) {
			return preInstallKVPairs, nil
		},
	}

	client := fake.NewFakeClientWithScheme(k8scheme.Scheme)

	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function,
	// plus values returned from the preInstall function if present
	var buffer bytes.Buffer
	buffer.WriteString("pilot.image=ghcr.io/verrazzano/pilot:1.7.3,global.proxy.image=proxyv2,global.tag=1.7.3,")
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
	setUpgradeFunc(func(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides string, overrideFiles ...string) (stdout []byte, stderr []byte, err error) {
		if overrides != expectedOverridesString {
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
	err := comp.Install(zap.S(), vz, client, "default", false)
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

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	assert.True(comp.IsInstalled(zap.S(), client, "default"))
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	assert.False(comp.IsInstalled(zap.S(), client, "default"))
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
	assert.True(comp.IsReady(zap.S(), client, "default"))

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartNotFound, nil
	})
	assert.False(comp.IsReady(zap.S(), client, "default"))

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusFailed, nil
	})
	assert.False(comp.IsReady(zap.S(), client, "default"))

	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return "", fmt.Errorf("Unexpected error")
	})
	assert.False(comp.IsReady(zap.S(), client, "default"))

	compInstalledWithNotReadyStatus := HelmComponent{
		ReadyStatusFunc: func(log *zap.SugaredLogger, client clipkg.Client, releaseName string, namespace string) bool {
			return false
		},
	}
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	assert.False(compInstalledWithNotReadyStatus.IsReady(zap.S(), client, "default"))

	compInstalledWithReadyStatus := HelmComponent{
		ReadyStatusFunc: func(log *zap.SugaredLogger, client clipkg.Client, releaseName string, namespace string) bool {
			return true
		},
	}
	helm.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return helm.ChartStatusDeployed, nil
	})
	assert.True(compInstalledWithReadyStatus.IsReady(zap.S(), client, "default"))
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeUpgrade(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
	if releaseName != "istiod" {
		return []byte("error"), []byte(""), errors.New("Invalid release name")
	}
	if chartDir != "ChartDir" {
		return []byte("error"), []byte(""), errors.New("Invalid chart directory name")
	}
	if namespace != "chartNS" {
		return []byte("error"), []byte(""), errors.New("Invalid chart namespace")
	}
	for _, file := range overridesFiles {
		if file != "ValuesFile" && file == "" {
			return []byte("error"), []byte(""), errors.New("Invalid values file")
		}
	}
	// This string is built from the Key:Value arrary returned by the bom.buildImageOverrides() function
	if overrides != fakeOverrides {
		return []byte("error"), []byte(""), errors.New("Invalid overrides")
	}
	return []byte("success"), []byte(""), nil
}

// helmFakeRunner overrides the helm run command
func (r helmFakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

func fakePreUpgrade(log *zap.SugaredLogger, client clipkg.Client, release string, namespace string, chartDir string) error {
	if release != "istiod" {
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
