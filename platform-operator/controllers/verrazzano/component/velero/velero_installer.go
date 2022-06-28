// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package velero

import (
	"bytes"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"time"
)

type bashCommand struct {
	Timeout      time.Duration `json:"timeout"`
	CmdDirectory string        `json:"directory"`
	CommandArgs  []string      `json:"cmdArgs"`
}

type runnerResponse struct {
	StandardOut  bytes.Buffer `json:"stdout"`
	StandardErr  bytes.Buffer `json:"stderr"`
	CommandError error        `json:"error"`
}

type VeleroImage struct {
	VeleroImage                    string `json:"velero"`
	VeleroPluginForAwsImage        string `json:"velero-plugin-for-aws"`
	VeleroResticRestoreHelperImage string `json:"velero-restic-restore-helper"`
}

// Generic method to execute shell commands
func genericRunner(bcmd *bashCommand, log vzlog.VerrazzanoLogger) *runnerResponse {
	var stdoutBuf, stderrBuf bytes.Buffer
	var bashCommandResponse runnerResponse
	shellCommand := exec.Command(bcmd.CommandArgs[0], bcmd.CommandArgs[1:]...) //nolint:gosec
	shellCommand.Dir = bcmd.CmdDirectory
	shellCommand.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	shellCommand.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	log.Debugf("Executing command '%v'", shellCommand.String())
	err := shellCommand.Start()
	if err != nil {
		log.Errorf("Cmd '%v' execution failed due to '%v'", shellCommand.String(), zap.Error(err))
		bashCommandResponse.CommandError = err
		return &bashCommandResponse
	}
	done := make(chan error, 1)
	go func() {
		done <- shellCommand.Wait()
	}()
	select {
	case <-time.After(bcmd.Timeout):
		if err = shellCommand.Process.Kill(); err != nil {
			log.Errorf("Failed to kill cmd '%v' due to '%v'", shellCommand.String(), zap.Error(err))
			bashCommandResponse.CommandError = err
			return &bashCommandResponse
		}
		log.Errorf("Cmd '%v' timeout expired", shellCommand.String())
		bashCommandResponse.CommandError = err
		return &bashCommandResponse
	case err = <-done:
		if err != nil {
			log.Errorf("Cmd '%v' execution failed due to '%v'", shellCommand.String(), zap.Error(err))
			bashCommandResponse.StandardErr = stderrBuf
			bashCommandResponse.CommandError = err
			return &bashCommandResponse
		}
		log.Debugf("Command '%s' execution successful", shellCommand.String())
		bashCommandResponse.StandardOut = stdoutBuf
		bashCommandResponse.CommandError = err
		return &bashCommandResponse
	}
}
