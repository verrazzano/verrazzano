// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"errors"
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/internal/util/helm"
	"go.uber.org/zap"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

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
		releaseName:             "release1",
		chartDir:                "chartDir",
		chartNamespace:          "chartNS",
		ignoreNamespaceOverride: true,
		valuesFile:              "valuesFile",
		preUpgradeFunc:          fakePreUpgrade,
	}

	helm.SetCmdRunner(helmFakeRunner{})
	defer helm.SetDefaultRunner()
	setUpgradeFunc(fakeUpgrade)
	defer setDefaultUpgradeFunc()
	err := comp.Upgrade(zap.S(), nil, "")
	assert.NoError(err, "Upgrade returned an error")
}

// fakeUpgrade verifies that the correct parameter values are passed to upgrade
func fakeUpgrade(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, overwriteYaml string) (stdout []byte, stderr []byte, err error) {
	if releaseName != "release1" {
		return []byte("error"), []byte(""), errors.New("Invalid release name")
	}
	if chartDir != "chartDir" {
		return []byte("error"), []byte(""), errors.New("Invalid chart directory name")
	}
	if namespace != "chartNS" {
		return []byte("error"), []byte(""), errors.New("Invalid chart namespace")
	}
	if overwriteYaml != "valuesFile" {
		return []byte("error"), []byte(""), errors.New("Invalid values file")
	}
	return []byte("success"), []byte(""), nil
}

// helmFakeRunner overrides the helm run command
func (r helmFakeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("success"), []byte(""), nil
}

func fakePreUpgrade(log *zap.SugaredLogger, client clipkg.Client, release string, namespace string, chartDir string) error {
	if release != "release1" {
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
