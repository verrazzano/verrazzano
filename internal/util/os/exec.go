// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package os

import (
	"bytes"
	"fmt"
	"os/exec"
)

type CmdRunner interface {
	Run(cmd *exec.Cmd) (stdout []byte, stderr []byte, err error)
}
// Runner is used to run an external command
type DefaultRunner struct{
}

// Verify that Verrazzano implements Component
var _ CmdRunner = DefaultRunner{}

// needed for unit test
var cmdRunFunc func(cmd *exec.Cmd) error

func (r DefaultRunner) Run (cmd *exec.Cmd) (stdout []byte, stderr []byte, err error) {
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
		return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), fmt.Errorf("failed to run '%s :  Error %s", cmd, err)
	}
	return stdoutBuffer.Bytes(), stderrBuffer.Bytes(), nil
}

