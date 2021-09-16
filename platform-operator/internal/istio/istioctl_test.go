package istio

import (
	"errors"
	"os/exec"
	"testing"

	"go.uber.org/zap"

	"github.com/stretchr/testify/assert"
)

// upgradeRunner is used to test istioctl upgrade without actually running an OS exec command
type upgradeRunner struct {
	t *testing.T
}

// getValuesRunner is used to test istioctl get values without actually running an OS exec command
type getValuesRunner struct {
	t *testing.T
}

// badRunner is used to test istioctl errors without actually running an OS exec command
type badRunner struct {
	t *testing.T
}

// foundRunner is used to test istioctl status command
type foundRunner struct {
	t *testing.T
}

// genericTestRunner is used to run generic OS commands with expected results
type genericTestRunner struct {
	stdOut []byte
	stdErr []byte
	err    error
}

// Run genericTestRunner executor
func (r genericTestRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return r.stdOut, r.stdErr, r.err
}

// TestUpgrade tests the istioctl upgrade command
// GIVEN a set of upgrade parameters
//  WHEN I call Upgrade
//  THEN the istioctl upgrade returns success and the cmd object has correct values
func TestUpgrade(t *testing.T) {
	overrideYaml := "my-override.yaml"

	assert := assert.New(t)
	SetCmdRunner(upgradeRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Upgrade(zap.S(), overrideYaml)
	assert.NoError(err, "Upgrade returned an error")
	assert.Len(stderr, 0, "Upgrade stderr should be empty")
	assert.NotZero(stdout, "Upgrade stdout should not be empty")
}

// TestUpgradeFail tests the istioctl upgrade command failure condition
// GIVEN a set of upgrade parameters and a fake runner that fails
//  WHEN I call Upgrade
//  THEN the istioctl upgrade returns an error
func TestUpgradeFail(t *testing.T) {
	assert := assert.New(t)
	SetCmdRunner(badRunner{t: t})
	defer SetDefaultRunner()

	stdout, stderr, err := Upgrade(zap.S(), "", "")
	assert.Error(err, "Upgrade should have returned an error")
	assert.Len(stdout, 0, "Upgrade stdout should be empty")
	assert.NotZero(stderr, "Upgrade stderr should not be empty")
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

// Run should return an error with stderr contents
func (r badRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return []byte(""), []byte("error"), errors.New("error")
}
