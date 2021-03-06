// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/helm"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// Needed for unit tests
var fakeOverrides string

// helmFakeRunner is used to test helm without actually running an OS exec command
type helmFakeRunner struct {
}

// TestGetName tests the component name
// GIVEN a Verrazzano component
//  WHEN I call Name
//  THEN the correct verrazzano name is returned
func TestGetName(t *testing.T) {
	comp := helmComponent{
		releaseName: "release1",
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

	comp := helmComponent{
		releaseName:             "istiod",
		chartDir:                "chartDir",
		chartNamespace:          "chartNS",
		ignoreNamespaceOverride: true,
		valuesFile:              "valuesFile",
		preUpgradeFunc:          fakePreUpgrade,
	}

	// This string is built from the key:value arrary returned by the bom.buildImageOverrides() function
	fakeOverrides = "pilot.image=ghcr.io/verrazzano/pilot:1.7.3,global.proxy.image=proxyv2,global.tag=1.7.3"

	SetUnitTestBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	err := comp.Upgrade(zap.S(), nil, "")
	assert.NoError(err, "Upgrade returned an error")
}

// TestUpgradeWithEnvOverrides tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade when the registry and repo overrides are set
//  THEN the upgrade returns success and passes the correct values to the upgrade function
func TestUpgradeWithEnvOverrides(t *testing.T) {
	assert := assert.New(t)

	comp := helmComponent{
		releaseName:             "istiod",
		chartDir:                "chartDir",
		chartNamespace:          "chartNS",
		ignoreNamespaceOverride: true,
		valuesFile:              "valuesFile",
		preUpgradeFunc:          fakePreUpgrade,
		appendOverridesFunc:     appendIstioOverrides,
	}

	os.Setenv(constants.RegistryOverrideEnvVar, "myreg.io")
	defer os.Unsetenv(constants.RegistryOverrideEnvVar)

	os.Setenv(constants.ImageRepoOverrideEnvVar, "myrepo")
	defer os.Unsetenv(constants.ImageRepoOverrideEnvVar)

	// This string is built from the key:value arrary returned by the bom.buildImageOverrides() function
	fakeOverrides = "pilot.image=myreg.io/myrepo/verrazzano/pilot:1.7.3,global.proxy.image=proxyv2,global.tag=1.7.3,global.hub=myreg.io/myrepo/verrazzano"

	SetUnitTestBomFilePath(testBomFilePath)
	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	err := comp.Upgrade(zap.S(), nil, "")
	assert.NoError(err, "Upgrade returned an error")
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeUpgrade(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, overrideFile string, overrides string) (stdout []byte, stderr []byte, err error) {
	if releaseName != "istiod" {
		return []byte("error"), []byte(""), errors.New("Invalid release name")
	}
	if chartDir != "chartDir" {
		return []byte("error"), []byte(""), errors.New("Invalid chart directory name")
	}
	if namespace != "chartNS" {
		return []byte("error"), []byte(""), errors.New("Invalid chart namespace")
	}
	if overrideFile != "valuesFile" {
		return []byte("error"), []byte(""), errors.New("Invalid values file")
	}
	// This string is built from the key:value arrary returned by the bom.buildImageOverrides() function
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
	if chartDir != "chartDir" {
		return fmt.Errorf("Incorrect chart directory %s", chartDir)
	}
	if namespace != "chartNS" {
		return fmt.Errorf("Incorrect namespace %s", namespace)
	}

	return nil
}
