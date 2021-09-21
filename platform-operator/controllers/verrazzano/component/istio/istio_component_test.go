// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"github.com/stretchr/testify/assert"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/istio"
	"go.uber.org/zap"
	"gopkg.in/errgo.v2/fmt/errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// fakeRunner is used to test istio without actually running an OS exec command
type fakeRunner struct {
}

var vz = &installv1alpha1.Verrazzano{
	Spec: installv1alpha1.VerrazzanoSpec{
		Components: installv1alpha1.ComponentSpec{
			Istio: &installv1alpha1.IstioComponent{
				IstioInstallArgs: []installv1alpha1.InstallArgs{{
					Name:  "arg1",
					Value: "val1",
				}},
			},
		},
	},
}

var comp = IstioComponent{}

const testBomFilePath = "../../testdata/test_bom.json"

// TestGetName tests the component name
// GIVEN a Verrazzano component
//  WHEN I call Name
//  THEN the correct verrazzano name is returned
func TestGetName(t *testing.T) {
	assert := assert.New(t)
	assert.Equal("istio", comp.Name(), "Wrong component name")
}

// TestUpgrade tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade
//  THEN the upgrade returns success and passes the correct values to the upgrade function
func TestUpgrade(t *testing.T) {
	assert := assert.New(t)

	comp := IstioComponent{
		ValuesFile: "test-values-file.yaml",
		Revision:   "1-1-1",
	}

	config.SetDefaultBomFilePath(testBomFilePath)
	istio.SetCmdRunner(fakeRunner{})
	defer istio.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	err := comp.Upgrade(zap.S(), vz, nil, "", false)
	assert.NoError(err, "Upgrade returned an error")
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeUpgrade(log *zap.SugaredLogger, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
	if len(overridesFiles) != 2 {
		return []byte("error"), []byte(""), errors.Newf("Incorrect number of override files: expected 2, received %v", len(overridesFiles))
	}
	if overridesFiles[0] != "test-values-file.yaml" {
		return []byte("error"), []byte(""), errors.New("Invalid values file")
	}
	if !strings.Contains(overridesFiles[1], "values-") || !strings.Contains(overridesFiles[1], ".yaml") {
		return []byte("error"), []byte(""), errors.New("Incorrect install args overrides file")
	}
	installArgsFromFile, err := os.ReadFile(overridesFiles[1])
	if err != nil {
		return []byte("error"), []byte(""), errors.New("Unable to read install args overrides file")
	}
	if !strings.Contains(string(installArgsFromFile), "val1") {
		return []byte("error"), []byte(""), errors.New("install args overrides file does not contain install args")
	}
	return []byte("success"), []byte(""), nil
}

// fakeRunner overrides the istio run command
func (r fakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}
