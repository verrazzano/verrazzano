// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package os

import (
	"bytes"
	"fmt"
	"os/exec"
)

// CmdRunner defines the interface to run an external command
type CmdRunner interface {
	Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error)
}

// DefaultRunner is used to run an external command
type DefaultRunner struct {
}

// Verify that Verrazzano implements Component
var _ CmdRunner = DefaultRunner{}

// neecmdRunFunc is needed for unit test
var cmdRunFunc func(cmd *exec.Cmd) error

// Run runs an external command.  The caller is expected to initialize the
// cmd name and args, for example using exec.Command(...)
func (r DefaultRunner) Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}
	cmd.Stdout = stdoutBuffer
	cmd.Stderr = stderrBuffer
	if cmdRunFunc != nil {
		err = cmdRunFunc(cmd)
	} else {
		err = cmd.Run()
	}
	if err != nil {
		return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), fmt.Errorf("Failed to run '%s :  Error %s", cmd, err)
	}
	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
}

// GenericTestRunner is used to run generic OS commands with expected results
type GenericTestRunner struct {
	StdOut []byte
	StdErr []byte
	Err    error
}

// Run GenericTestRunner executor
func (r GenericTestRunner) Run(_ *exec.Cmd) (stdout []byte, stderr []byte, err error) {
	return r.StdOut, r.StdErr, r.Err
}
