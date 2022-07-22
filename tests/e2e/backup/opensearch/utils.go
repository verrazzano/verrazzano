// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"bytes"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"time"
)

type BashCommand struct {
	Timeout     time.Duration `json:"timeout"`
	CommandArgs []string      `json:"cmdArgs"`
}

type RunnerResponse struct {
	StandardOut  bytes.Buffer `json:"stdout"`
	StandardErr  bytes.Buffer `json:"stderr"`
	CommandError error        `json:"error"`
}

func Runner(bcmd *BashCommand, log *zap.SugaredLogger) *RunnerResponse {
	var stdoutBuf, stderrBuf bytes.Buffer
	var bashCommandResponse RunnerResponse
	bashCommand := exec.Command(bcmd.CommandArgs[0], bcmd.CommandArgs[1:]...) //nolint:gosec
	bashCommand.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	bashCommand.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	log.Infof("Executing command '%v'", bashCommand.String())
	err := bashCommand.Start()
	if err != nil {
		log.Errorf("Cmd '%v' execution failed due to '%v'", bashCommand.String(), zap.Error(err))
		bashCommandResponse.CommandError = err
		return &bashCommandResponse
	}
	done := make(chan error, 1)
	go func() {
		done <- bashCommand.Wait()
	}()
	select {
	case <-time.After(bcmd.Timeout):
		if err = bashCommand.Process.Kill(); err != nil {
			log.Errorf("Failed to kill cmd '%v' due to '%v'", bashCommand.String(), zap.Error(err))
			bashCommandResponse.CommandError = err
			return &bashCommandResponse
		}
		log.Errorf("Cmd '%v' timeout expired", bashCommand.String())
		bashCommandResponse.CommandError = err
		return &bashCommandResponse
	case err = <-done:
		if err != nil {
			log.Errorf("Cmd '%v' execution failed due to '%v'", bashCommand.String(), zap.Error(err))
			bashCommandResponse.StandardErr = stderrBuf
			bashCommandResponse.CommandError = err
			return &bashCommandResponse
		}
		log.Debugf("Command '%s' execution successful", bashCommand.String())
		bashCommandResponse.StandardOut = stdoutBuf
		bashCommandResponse.CommandError = err
		return &bashCommandResponse
	}
}
