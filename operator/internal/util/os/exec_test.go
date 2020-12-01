// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package os

import (
	"github.com/stretchr/testify/assert"
	"os/exec"
	"testing"
)

// TestRun tests the exec Run command
// GIVEN a command
//  WHEN I call Run
//  THEN the value returned will have the correct stdout and stderr
func TestRun(t *testing.T) {
	assert := assert.New(t)

	// override cmd.Run function
	cmdRunFunc = fakeCmdRunner
	cmd := exec.Command("helm", "arg1", "arg2", "arg3")
	stdout, stderr, err := DefaultRunner{}.Run(cmd)
	assert.NoError(err, "Error should not be returned from exec")
	assert.Len(stderr, 0, "stderr is incorrect")
	assert.Equal("success", string(stdout), "stdout is incorrect")
	assert.NoError(err, "Error should not be returned from exec")
	assert.Contains(cmd.Path, "helm", "exec command should contain helm")
	assert.Contains(cmd.Args[0], "helm", "exec args should contain helm")
	assert.Equal(cmd.Args[1], "arg1", "exec arg should equal arg1")
	assert.Equal(cmd.Args[2], "arg2", "exec arg should equal arg2")
	assert.Contains(cmd.Args[3], "arg3", "exec arg should equal arg3 ")
}

func fakeCmdRunner(cmd *exec.Cmd) error {
	cmd.Stdout.Write([]byte("success"))
	cmd.Stderr.Write([]byte(""))
	return nil
}
