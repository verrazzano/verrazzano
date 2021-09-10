// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	"os/exec"
	"testing"
)

// istioFakeRunner is used to test istio without actually running an OS exec command
type istioFakeRunner struct {
}

var comp istioComponent = istioComponent{
	componentName: "istio",
}

// IstioTestGetName tests the component name
// GIVEN a Verrazzano component
//  WHEN I call Name
//  THEN the correct verrazzano name is returned
func IstioTestGetName(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("istio", comp.Name(), "Wrong component name")
}

// IstioTestUpgrade tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade
//  THEN the upgrade returns success and passes the correct values to the upgrade function
func IstioTestUpgrade(t *testing.T) {
	assert := assert.New(t)

	SetUnitTestBomFilePath(testBomFilePath)
	istio.SetCmdRunner(istioFakeRunner{})
	defer istio.SetDefaultRunner()
	setIstioUpgradeFunc(fakeUpgrade)
	defer setistioDefaultUpgradeFunc()
	istio.SetChartStatusFunction(func(releaseName string, namespace string) (string, error) {
		return istio.ChartStatusDeployed, nil
	})
	defer istio.SetDefaultChartStatusFunction()
	err := comp.Upgrade(zap.S(), nil, "", false)
	assert.NoError(err, "Upgrade returned an error")
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeUpgrade(log *zap.SugaredLogger, componentName string) (stdout []byte, stderr []byte, err error) {
	if releaseName != "istio" {
		return []byte("error"), []byte(""), errors.New("Invalid release name")
	}
	if chartDir != "chartDir" {
		return []byte("error"), []byte(""), errors.New("Invalid chart directory name")
	}
	if namespace != "chartNS" {
		return []byte("error"), []byte(""), errors.New("Invalid chart namespace")
	}
	for _, file := range overridesFiles {
		if file != "valuesFile" && file == "" {
			return []byte("error"), []byte(""), errors.New("Invalid values file")
		}
	}
	// This string is built from the key:value arrary returned by the bom.buildImageOverrides() function
	if overrides != fakeOverrides {
		return []byte("error"), []byte(""), errors.New("Invalid overrides")
	}
	return []byte("success"), []byte(""), nil
}

// istioFakeRunner overrides the istio run command
func (r istioFakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}
