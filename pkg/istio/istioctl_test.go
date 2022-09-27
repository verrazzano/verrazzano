// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

// upgradeRunner is used to test istioctl upgrade without actually running an OS exec command
type upgradeRunner struct {
	t *testing.T
}

// installRunner is used to test istioctl install without actually running an OS exec command
type installRunner struct {
	t *testing.T
}

// uninstallRunner is used to test istioctl uninstall without actually running an OS exec command
type uninstallRunner struct {
	t *testing.T
}

// badRunner is used to test istioctl errors without actually running an OS exec command
type badRunner struct {
	t *testing.T
}

// TestUpgrade tests the istioctl upgrade command
// GIVEN a set of upgrade parameters
//
//	WHEN I call Upgrade
//	THEN the istioctl upgrade returns success and the cmd object has correct values
func TestUpgrade(t *testing.T) {
	overrideYaml := "my-override.yaml"

	assert := assert.New(t)
	SetCmdRunner(upgradeRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Upgrade(vzlog.DefaultLogger(), overrideYaml)
	assert.NoError(err, "Upgrade returned an error")
	assert.Len(stderr, 0, "Upgrade stderr should be empty")
	assert.NotZero(stdout, "Upgrade stdout should not be empty")
}

// TestUpgradeFail tests the istioctl upgrade command failure condition
// GIVEN a set of upgrade parameters and a fake runner that fails
//
//	WHEN I call Upgrade
//	THEN the istioctl upgrade returns an error
func TestUpgradeFail(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(badRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Upgrade(vzlog.DefaultLogger(), "", "")
	assert.Error(err, "Upgrade should have returned an error")
	assert.Len(stdout, 0, "Upgrade stdout should be empty")
	assert.NotZero(stderr, "Upgrade stderr should not be empty")
}

// TestInstall tests the istioctl install command
// GIVEN a set of upgrade parameters
//
//	WHEN I call Install
//	THEN the istioctl install returns success and the cmd object has correct values
func TestInstall(t *testing.T) {
	overrideYaml := "my-override.yaml"

	assert := assert.New(t)
	SetCmdRunner(installRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Install(vzlog.DefaultLogger(), overrideYaml)
	assert.NoError(err, "Install returned an error")
	assert.Len(stderr, 0, "Install stderr should be empty")
	assert.NotZero(stdout, "Install stdout should not be empty")
}

// TestIsInstalled tests if the component is installed
// GIVEN a component
//
//	WHEN I call IsInstalled
//	THEN true is returned
func TestIsInstalled(t *testing.T) {
	assert := assert.New(t)

	SetCmdRunner(fakeIstioInstalledRunner{})
	b, err := IsInstalled(vzlog.DefaultLogger())
	assert.NoError(err, "IsInstalled returned an error")
	assert.True(b, "IsInstalled returned false")
}

// TestUninstall tests the istioctl uninstall command
// GIVEN a set of uninstall parameters and a fake uninstall runner
//
//	WHEN I call Uninstall
//	THEN the istioctl uninstall returns success
func TestUninstall(t *testing.T) {

	assert := assert.New(t)
	SetCmdRunner(uninstallRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Uninstall(vzlog.DefaultLogger())
	assert.NoError(err, "Uninstall returned an error")
	assert.Len(stderr, 0, "Uninstall stderr should be empty")
	assert.NotZero(stdout, "Uninstall stdout should not be empty")
}

// TestUninstallFail tests the istioctl uninstall command failure condition
// GIVEN a set of uninstall parameters and a fake runner that fails
//
//	WHEN I call Uninstall
//	THEN the istioctl uninstall returns an error
func TestUninstallFail(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(badRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Uninstall(vzlog.DefaultLogger())
	assert.Error(err, "Uninstall should have returned an error")
	assert.Len(stdout, 0, "Uninstall stdout should be empty")
	assert.NotZero(stderr, "Uninstall stderr should not be empty")
}

// fakeIsInstalledRunner overrides the istio run command
func (r fakeIstioInstalledRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte("Istio is installed and verified successfully"), []byte(""), nil
}

// Run should assert the command parameters are correct then return a success with stdout contents
func (r upgradeRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	assert := assert.New(r.t)
	assert.Contains(cmd.Path, "istioctl", "command should contain istioctl")
	assert.Contains(cmd.Args[0], "istioctl", "args should contain istioctl")
	assert.Contains(cmd.Args[1], "install", "args should contain install")
	assert.Contains(cmd.Args[2], "-y", "args should contain -y")

	return []byte("success"), []byte(""), nil
}

// Run should assert the command parameters are correct then return a success with stdout contents
func (r installRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	assert := assert.New(r.t)
	assert.Contains(cmd.Path, "istioctl", "command should contain istioctl")
	assert.Contains(cmd.Args[0], "istioctl", "args should contain istioctl")
	assert.Contains(cmd.Args[1], "install", "args should contain install")
	assert.Contains(cmd.Args[2], "-y", "args should contain -y")

	return []byte("success"), []byte(""), nil
}

// Run should assert the command parameters are correct then return a success with stdout contents
func (r uninstallRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	assert := assert.New(r.t)
	assert.Contains(cmd.Path, "istioctl", "command should contain istioctl")
	assert.Contains(cmd.Args[0], "istioctl", "args should contain istioctl")
	assert.Contains(cmd.Args[1], "x", "args should contain x")
	assert.Contains(cmd.Args[2], "uninstall", "args should contain uninstall")
	assert.Contains(cmd.Args[3], "--revision", "args should contain --revision")
	assert.Contains(cmd.Args[4], "default", "args should contain default")
	assert.Contains(cmd.Args[5], "-y", "args should contain -y")

	return []byte("success"), []byte(""), nil
}

// Run should return an error with stderr contents
func (r badRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte(""), []byte("error"), errors.New("error")
}
