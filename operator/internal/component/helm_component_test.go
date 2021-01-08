// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/operator/internal/util/helm"
	"os/exec"
	"testing"
)

// fakeRunner is used to test helm without actually running an OS exec command
type helmFakeRunner struct {
}

type testComponent struct {
}


// TestGetName tests the component name
// GIVEN a Verrazzano component
//  WHEN I call Name
//  THEN the correct verrazzano name is returned
func TestGetName(t *testing.T) {
	comp:= helmComponent{
		releaseName:        "release1",
	}

	assert := assert.New(t)
	assert.Equal("release1", comp.Name(), "Wrong component name")
}

// TestUpgrade tests the component upgrade
// GIVEN a component
//  WHEN I call Upgrade
//  THEN the upgrade returns success and passes the correct values to the runner
func TestUpgrade(t *testing.T) {
	const defNs = "verrazzano-system"
	assert := assert.New(t)

	comp:= helmComponent{
		releaseName:        "release1",
		chartDir:           "chartDir",
		chartNamespace:     "chartNS",
		namespaceHardcoded: true,
		valuesFile:         "valuesFile",
	}

	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	err := comp.Upgrade("")
	assert.NoError(err, "Upgrade returned an error")
}

func fakeUpgrade(releaseName string, namespace string, chartDir string, overwriteYaml string) (stdout []byte, stderr []byte, err error) {
	if releaseName != "release1" {
		return  []byte("error"), []byte(""), errors.New("Invalid release name")
	}
	if chartDir != "chartDir" {
		return  []byte("error"), []byte(""), errors.New("Invalid chart directory name")
	}
	if namespace != "chartNS" {
		return  []byte("error"), []byte(""), errors.New("Invalid chart namespace")
	}
	if overwriteYaml != "valuesFile" {
		return  []byte("error"), []byte(""), errors.New("Invalid values file")
	}
	return []byte("success"), []byte(""), nil
}

func (r helmFakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}
