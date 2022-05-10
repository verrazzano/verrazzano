package operator

import (
	"bytes"
	"go.uber.org/zap"
	"io"
	"os"
	"os/exec"
	"time"
)

type BashCommand struct {
	Timeout      time.Duration `json:"timeout"`
	CmdDirectory string        `json:"directory"`
	CommandArgs  []string      `json:"cmdArgs"`
}

type RunnerResponse struct {
	Stdout bytes.Buffer `json:"stdout"`
	Stderr bytes.Buffer `json:"stderr"`
	Error  error        `json:"error"`
}

func VeleroRunner(bcmd *BashCommand, log *zap.SugaredLogger) *RunnerResponse {
	var stdoutBuf, stderrBuf bytes.Buffer
	var response RunnerResponse
	execCmd := exec.Command(bcmd.CommandArgs[0], bcmd.CommandArgs[1:]...)
	execCmd.Dir = bcmd.CmdDirectory
	execCmd.Stdout = io.MultiWriter(os.Stdout, &stdoutBuf)
	execCmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)

	log.Debugf("Executing command '%v'", execCmd.String())
	err := execCmd.Start()
	if err != nil {
		log.Errorf("Cmd '%v' execution failed due to '%v'", execCmd.String(), zap.Error(err))
		response.Error = err
		return &response
	}
	done := make(chan error, 1)
	go func() {
		done <- execCmd.Wait()
	}()
	select {
	case <-time.After(bcmd.Timeout):
		if err = execCmd.Process.Kill(); err != nil {
			log.Errorf("Failed to kill cmd '%v' due to '%v'", execCmd.String(), zap.Error(err))
			response.Error = err
			return &response
		}
		log.Errorf("Cmd '%v' timeout expired", execCmd.String())
		response.Error = err
		return &response
	case err = <-done:
		if err != nil {
			log.Errorf("Cmd '%v' execution failed due to '%v'", execCmd.String(), zap.Error(err))
			//log.Error(stderrBuf.String())
			response.Stderr = stderrBuf
			response.Error = err
			return &response
		} else {
			log.Debugf("Command '%s' execution successfull", execCmd.String())
			//log.Info(stdoutBuf.String())
			response.Stdout = stdoutBuf
			response.Error = err
			return &response
		}
	}
}
