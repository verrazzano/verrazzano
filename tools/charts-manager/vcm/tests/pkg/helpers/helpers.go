// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"os"

	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

const (
	DummyError   = "dummy error"
	FlagNotFound = "%s flag not supported by command"
)

// ContextSetup creates a default RootCmdContext.
func ContextSetup() (*testhelpers.FakeRootCmdContext, func(), error) {
	stdoutFile, stderrFile, err := createStdTempFiles()
	if err != nil {
		return nil, nil, err
	}
	return testhelpers.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: stdoutFile, ErrOut: stderrFile}), func() {
		os.Remove(stdoutFile.Name())
		os.Remove(stderrFile.Name())
	}, nil
}

// createStdTempFiles creates temporary files for stdout and stderr.
func createStdTempFiles() (*os.File, *os.File, error) {
	stdoutFile, err := os.CreateTemp("", "tmpstdout")
	if err != nil {
		return nil, nil, err
	}

	stderrFile, err := os.CreateTemp("", "tmpstderr")
	if err != nil {
		return nil, nil, err
	}

	return stdoutFile, stderrFile, nil
}
