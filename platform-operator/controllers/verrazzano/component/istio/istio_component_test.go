// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	"os/exec"
	"testing"
)

// istioFakeRunner is used to test istio without actually running an OS exec command
type istioFakeRunner struct {
}

var comp = IstioComponent{}

const testBomFilePath = "../../testdata/test_bom.json"

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

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(istioFakeRunner{})
	defer istio.SetDefaultRunner()
	setIstioUpgradeFunc(fakeIstioUpgrade)
	defer setIstioDefaultUpgradeFunc()
	err := comp.Upgrade(zap.S(), nil)
	assert.NoError(err, "Upgrade returned an error")
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeIstioUpgrade(log *zap.SugaredLogger, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
	//TODO: add testing
	return []byte("success"), []byte(""), nil
}

// istioFakeRunner overrides the istio run command
func (r istioFakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}
