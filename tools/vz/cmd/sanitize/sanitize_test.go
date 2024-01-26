// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package sanitize

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"testing"
)

// TestNewCmdSanitize
// GIVEN a VZ Helper
// WHEN I call NewCmdSanitize
// THEN expect the command to successfully create a new Sanitize command
func TestNewCmdSanitize(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdSanitize(rc)
	assert.NotNil(t, cmd)
}

// TestNoArgsIntoSanitize
// GIVEN an Sanitize command
// WHEN I call cmd.Execute() with no flags passed in
// THEN expect the command to return an error
func TestNoArgsIntoSanitize(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdSanitize(rc)
	assert.NotNil(t, cmd)
	err := cmd.Execute()
	assert.NotNil(t, err)
}

// createStdTempFiles creates temporary files for stdout and stderr.
func createStdTempFiles(t *testing.T) (*os.File, *os.File) {
	stdoutFile, err := os.CreateTemp("", "tmpstdout")
	assert.NoError(t, err)

	stderrFile, err := os.CreateTemp("", "tmpstderr")
	assert.NoError(t, err)

	return stdoutFile, stderrFile
}
