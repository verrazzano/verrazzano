package operator

import (
	"bytes"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
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

type VeleroImage struct {
	VeleroImage                    string `json:"velero"`
	VeleroPluginForAwsImage        string `json:"velero-plugin-for-aws"`
	VeleroResticRestoreHelperImage string `json:"velero-restic-restore-helper"`
}

// Generic method to execute shell commands
func veleroRunner(bcmd *BashCommand, log vzlog.VerrazzanoLogger) *RunnerResponse {
	var stdoutBuf, stderrBuf bytes.Buffer
	var response RunnerResponse
	execCmd := exec.Command(bcmd.CommandArgs[0], bcmd.CommandArgs[1:]...) //nolint:gosec
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
			response.Stderr = stderrBuf
			response.Error = err
			return &response
		}
		log.Debugf("Command '%s' execution successful", execCmd.String())
		response.Stdout = stdoutBuf
		response.Error = err
		return &response
	}
}
