// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"errors"
	"os/exec"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
)

const ns = "my_namespace"
const chartdir = "my_charts"
const release = "my_release"
const missingRelease = "no_release"
const overrideYaml = "my-override.yaml"

// goodRunner is used to test Helm without actually running an OS exec command
type goodRunner struct {
	t *testing.T
}

// badRunner is used to test Helm errors without actually running an OS exec command
type badRunner struct {
	t *testing.T
}

// foundRunner is used to test helm status command
type foundRunner struct {
	t *testing.T
}

// TestUpgrade tests the Helm upgrade command
// GIVEN a set of upgrade parameters
//  WHEN I call Upgrade
//  THEN the Helm upgrade returns success and the cmd object has correct values
func TestUpgrade(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(goodRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Upgrade(zap.S(), release, ns, chartdir, overrideYaml)
	assert.NoError(err, "Upgrade returned an error")
	assert.Len(stderr, 0, "Upgrade stderr should be empty")
	assert.NotZero(stdout, "Upgrade stdout should not be empty")
}

// TestUpgradeFail tests the Helm upgrade command failure condition
// GIVEN a set of upgrade parameters and a fake runner that fails
//  WHEN I call Upgrade
//  THEN the Helm upgrade returns an error
func TestUpgradeFail(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(badRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Upgrade(zap.S(), release, ns, "", "")
	assert.Error(err, "Upgrade should have returned an error")
	assert.Len(stdout, 0, "Upgrade stdout should be empty")
	assert.NotZero(stderr, "Upgrade stderr should not be empty")
}

// TestIsReleaseInstalled tests checking if a Helm release is installed
// GIVEN a release name and namespace
//  WHEN I call IsReleaseInstalled
//  THEN the function returns success and found equal true
func TestIsReleaseInstalled(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(foundRunner{t: t})
	defer SetDefaultRunner()

	found, err := IsReleaseInstalled(release, ns)
	assert.NoError(err, "IsReleaseInstalled returned an error")
	assert.True(found, "Release not found")
}

// TestIsReleaseNotInstalled tests checking if a Helm release is not installed
// GIVEN a release name and namespace
//  WHEN I call IsReleaseInstalled
//  THEN the function returns success and the correct found status
func TestIsReleaseNotInstalled(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(foundRunner{t: t})
	defer SetDefaultRunner()

	found, err := IsReleaseInstalled(missingRelease, ns)
	assert.NoError(err, "IsReleaseInstalled returned an error")
	assert.False(found, "Release should not be found")
}

// TestIsReleaseInstalledFailed tests failure when checking if a Helm release is installed
// GIVEN a bad release name and namespace
//  WHEN I call IsReleaseInstalled
//  THEN the function returns a failure
func TestIsReleaseInstalledFailed(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(foundRunner{t: t})
	defer SetDefaultRunner()

	found, err := IsReleaseInstalled("", ns)
	assert.Error(err, "IsReleaseInstalled should have returned an error")
	assert.False(found, "Release should not be found")
}

// Run should assert the command parameters are correct then return a success with stdout contents
func (r goodRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	assert := assert.New(r.t)
	assert.Contains(cmd.Path, "helm", "command should contain helm")
	assert.Contains(cmd.Args[0], "helm", "args should contain helm")
	assert.Contains(cmd.Args[1], "upgrade", "args should contain upgrade")
	assert.Contains(cmd.Args[2], release, "args should contain release name")
	assert.Contains(cmd.Args[3], chartdir, "args should contain chart dir ")

	return []byte("success"), []byte(""), nil
}

// Run should return an error with stderr contents
func (r badRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte(""), []byte("error"), errors.New("error")
}

// Run should assert the command parameters are correct then return a success or error depending on release name
func (r foundRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	assert := assert.New(r.t)
	assert.Contains(cmd.Path, "helm", "command should contain helm")
	assert.Contains(cmd.Args[0], "helm", "args should contain helm")
	assert.Contains(cmd.Args[1], "status", "args should contain status")

	if cmd.Args[2] == release {
		return []byte(""), []byte(""), nil
	}
	if cmd.Args[2] == missingRelease {
		return []byte(""), []byte("not found"), errors.New("not found error")
	}
	// simulate a Helm error
	return []byte(""), []byte("error"), errors.New("helm error")
}
