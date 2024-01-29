// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package sanitize

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
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

// TestTwoInputArgsIntoSanitize
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an input directory and an input tar file specified
// THEN expect the command to return an error
func TestTwoInputArgsIntoSanitize(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputDirectoryFlagName, "input-directory")
	cmd.PersistentFlags().Set(constants.InputTarFileFlagName, "input.tar")
	assert.NotNil(t, cmd)
	err := cmd.Execute()
	assert.NotNil(t, err)
}

// TestTwoOutputArgsIntoSanitize
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an output directory and an output tar.gz file specified
// THEN expect the command to return an error
func TestTwoOutputArgsIntoSanitize(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.OutputDirectoryFlagName, "output-directory")
	cmd.PersistentFlags().Set(constants.OutputTarGZFileFlagName, "output.tar.gz")
	assert.NotNil(t, cmd)
	err := cmd.Execute()
	assert.NotNil(t, err)
}

// TestSanitizeCommandWithInputDirectoryAndOutputTarGZFil
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an input directory and an output tar.gz file specified
// THEN expect the command to not return an error and to create the specified tar.gz file
func TestSanitizeCommandWithInputDirectoryAndOutputTarGZFile(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputDirectoryFlagName, "../../pkg/analysis/test/cluster/testCattleSystempods")
	cmd.PersistentFlags().Set(constants.OutputTarGZFileFlagName, "output-tar-file.tar.gz")
	defer os.Remove("output-tar-file.tar.gz")
	assert.NotNil(t, cmd)
	err := cmd.Execute()
	assert.Nil(t, err)
}

// TestSanitizeCommandWithInputDirectoryAndOutputDirectoryFile
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an input directory and an output directory file specified
// THEN expect the command to not return an error and to create the specified directory
func TestSanitizeCommandWithInputDirectoryAndOutputDirectoryFile(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputDirectoryFlagName, "../../pkg/analysis/test/cluster/testCattleSystempods")
	cmd.PersistentFlags().Set(constants.OutputDirectoryFlagName, "test-directory")
	defer os.RemoveAll("test-directory")
	assert.NotNil(t, cmd)
	err := cmd.Execute()
	assert.Nil(t, err)
}

// TestSanitizeCommandWithInputTarAndOutputDirectoryFile
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an input tar file and an output directory file specified
// THEN expect the command to not return an error and to create the specified directory
func TestSanitizeCommandWithInputTarAndOutputTarGZFile(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputTarFileFlagName, "../../pkg/analysis/test/cluster/istio-ingress-ip-not-found-test.tar")
	cmd.PersistentFlags().Set(constants.OutputTarGZFileFlagName, "output-tar-file.tar.gz")
	defer os.Remove("output-tar-file.tar.gz")
	assert.NotNil(t, cmd)
	err := cmd.Execute()
	assert.Nil(t, err)
}

// TestSanitizeCommandWithInputTarAndOutputTarFile
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an input tar file and an output tar.gz file specified
// THEN expect the command to not return an error and to create the specified directory
func TestSanitizeCommandWithInputTarAndOutputDirectory(t *testing.T) {
	stdoutFile, stderrFile := createStdTempFiles(t)
	defer func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}()
	rc := helpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile})
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputTarFileFlagName, "../../pkg/analysis/test/cluster/istio-ingress-ip-not-found-test.tar")
	cmd.PersistentFlags().Set(constants.OutputDirectoryFlagName, "test-directory")
	defer os.RemoveAll("test-directory")
	assert.NotNil(t, cmd)
	err := cmd.Execute()
	assert.Nil(t, err)
}

// createStdTempFiles creates temporary files for stdout and stderr.
func createStdTempFiles(t *testing.T) (*os.File, *os.File) {
	stdoutFile, err := os.CreateTemp("", "tmpstdout")
	assert.NoError(t, err)

	stderrFile, err := os.CreateTemp("", "tmpstderr")
	assert.NoError(t, err)

	return stdoutFile, stderrFile
}
