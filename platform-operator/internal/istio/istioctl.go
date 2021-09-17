// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"fmt"
	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"go.uber.org/zap"
	"os/exec"
)

// cmdRunner needed for unit tests
var runner vzos.CmdRunner = vzos.DefaultRunner{}

// Upgrade function gets called from istio_component to perform istio upgrade
func Upgrade(log *zap.SugaredLogger, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
	args := []string{"install", "-y"}

	args = append(args, "--set")
	args = append(args, "revision=1-10-2")

	for _, overridesFileName := range overridesFiles {
		args = append(args, "-f")
		args = append(args, overridesFileName)
	}

	// Perform istioctl call of type upgrade
	stdout, stderr, err = runIstioctl(log, args, "upgrade")
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// runIstioctl will perform istioctl calls with specified arguments  for operations
// Note that operation name as of now does not affect the istioctl call (both upgrade and install call istioctl install)
// The operationName field is just used for visibility of operation in logging at the moment
func runIstioctl(log *zap.SugaredLogger, cmdArgs []string, operationName string) (stdout []byte, stderr []byte, err error) {
	cmd := exec.Command("istioctl", cmdArgs...)
	log.Infof("Running command: %s", cmd.String())
	stdout, stderr, err = runner.Run(cmd)
	if err != nil {
		fmt.Printf("HERE=============     %v", err)
		log.Errorf("istioctl %s failed: %s", operationName, stderr)
		return stdout, stderr, err
	}

	log.Infof("istioctl %s succeeded: %s", operationName, stdout)

	return stdout, stderr, nil
}

// SetCmdRunner sets the command runner as needed by unit tests
func SetCmdRunner(r vzos.CmdRunner) {
	runner = r
}

// SetDefaultRunner sets the command runner to default
func SetDefaultRunner() {
	runner = vzos.DefaultRunner{}
}
