// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

const ns = "my_namespace"
const chartdir = "my_charts"
const release = "my_release"
const overrideYaml = "my-override.yaml"

// goodRunner is used to test helm without actually running an OS exec command
type goodRunner struct {
	t *testing.T
}

// badRunner is used to test helm errors without actually running an OS exec command
type badRunner struct {
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

	stdout, stderr, err := Upgrade(release, ns, chartdir, overrideYaml)
	assert.NoError(err, "Upgrade returned an error")
	assert.Len(stderr, 0, "Upgrade stderr should be empty")
	assert.NotZero(stdout, "Upgrade stdout should not be empty")
}

// TestUpgradeFail tests the Helm upgrade command
// GIVEN a set of upgrade parameters and a fake runner that fails
//  WHEN I call Upgrade
//  THEN the Helm upgrade returns an error
func TestUpgradeFail(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(badRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Upgrade(release, ns, "", "")
	assert.Error(err, "Upgrade should have returned an error")
	assert.Len(stdout, 0, "Upgrade stderr should be empty")
	assert.NotZero(stderr, "Upgrade stdout should not be empty")
}

// RunCommand should assert that the cmd contains the correct data
func (r goodRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	assert := assert.New(r.t)
	assert.Contains(cmd.Path, "helm", "exec command should contain helm")
	assert.Contains(cmd.Args[0], "helm", "exec args should contain helm")
	assert.Contains(cmd.Args[1], "upgrade", "exec args should contain upgrade")
	assert.Contains(cmd.Args[2], release, "exec args should contain release name")
	assert.Contains(cmd.Args[3], chartdir, "exec args should contain chart dir ")

	return []byte("success"), []byte(""), nil
}

// RunCommand should assert that the cmd contains the correct data
func (r badRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte(""), []byte("error"), errors.New("error")
}
