// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package os

import (
	osexec "os/exec"
)

// cmdRunner needed for unit tests
var runner CmdRunner = DefaultRunner{}

// RunBash runs a bach script
func RunBash(inArgs ...string, ) (string, string, error) {
	args := []string{}
	for i := range inArgs {
		args = append(args, inArgs[i])
	}
	cmd := osexec.Command("bash", args...)
	stdout, stderr, err := runner.Run(cmd)
	if err != nil {
		return string(stdout), string(stderr), err
	}
	return "", "", err
}
