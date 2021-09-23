
package util

import (
	"bytes"
	"fmt"
	"os/exec"
)

// neecmdRunFunc is needed ded for unit test
var cmdRunFunc func(cmd *exec.Cmd) error

func K(commandLine string) (string, string) {
	cmd := exec.Command("kubectl", "apply", "-f",  commandLine)
	stdoutBuffer := &bytes.Buffer{}
	stderrBuffer := &bytes.Buffer{}
	cmd.Stdout = stdoutBuffer
	cmd.Stderr = stderrBuffer
	fmt.Println(commandLine)

	err := cmd.Run()
	if err != nil {
		fmt.Println("exec err " + err.Error() + " " + stderrBuffer.String() + stdoutBuffer.String())
		return "", "error"
	}
	fmt.Print("exec returns " + stderrBuffer.String())

	return "", stderrBuffer.String()
}
