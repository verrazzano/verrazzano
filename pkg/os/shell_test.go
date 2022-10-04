// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package os

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"os/exec"
	"testing"
)

// TestRunBash tests the exec RunBash command
// GIVEN a command
//
//	WHEN I call Run
//	THEN the value returned will have the correct stdout and stderr
func TestRunBash(t *testing.T) {
	assert := assert.New(t)

	// override cmd.Run function
	cmdRunFunc = bashGoodCmdRunner
	stdout, stderr, err := RunBash("fakecmd", "arg1", "arg2", "arg3")
	assert.NoError(err, "Error should not be returned from exec")
	assert.Len(stderr, 0, "stderr is incorrect")
	assert.Equal("success", string(stdout), "stdout is incorrect")
	assert.NoError(err, "Error should not be returned from exec")
}

// TestRunBashErr tests the exec Run error condition
// GIVEN a command and a fake runner that returns an error
//
//	WHEN I call Run
//	THEN the value returned will have and error status and the correct stdout and stderr
func TestRunBashErr(t *testing.T) {
	assert := assert.New(t)

	// override cmd.Run function
	cmdRunFunc = bashBadCmdRunner
	cmd := exec.Command("fakecmd", "arg1", "arg2", "arg3")
	stdout, stderr, err := DefaultRunner{}.Run(cmd)
	assert.Error(err, "Error should be returned from exec")
	assert.Len(stdout, 0, "stdout is incorrect")
	assert.Equal("err", string(stderr), "stderr is incorrect")
}

func bashGoodCmdRunner(cmd *exec.Cmd) error {
	cmd.Stdout.Write([]byte("success"))
	cmd.Stderr.Write([]byte(""))
	return nil
}

func bashBadCmdRunner(cmd *exec.Cmd) error {
	cmd.Stdout.Write([]byte(""))
	cmd.Stderr.Write([]byte("err"))
	return errors.New("error")
}
