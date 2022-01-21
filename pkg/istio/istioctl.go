// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"os/exec"
	"strings"

	vzos "github.com/verrazzano/verrazzano/pkg/os"

	"go.uber.org/zap"
)

// cmdRunner needed for unit tests
var runner vzos.CmdRunner = vzos.DefaultRunner{}

// fakeIstioInstalledRunner is used to test if Istio is installed
type fakeIstioInstalledRunner struct {
}

// Upgrade function gets called from istio_component to perform istio upgrade
func Upgrade(log *zap.SugaredLogger, imageOverrideString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
	args := []string{"install", "-y"}

	// Add override files to arg array
	for _, overridesFileName := range overridesFiles {
		args = append(args, "-f")
		args = append(args, overridesFileName)
	}

	// Add the image override strings
	if len(imageOverrideString) > 0 {
		segs := strings.Split(imageOverrideString, ",")
		for i := range segs {
			args = append(args, "--set")
			args = append(args, segs[i])
		}
	}

	// Perform istioctl call of type upgrade
	stdout, stderr, err = runIstioctl(log, args, "upgrade")
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// Install does and Istio installation using or or more IstioOperator YAML files
func Install(log *zap.SugaredLogger, overrideStrings string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
	args := []string{"install", "-y"}

	for _, overridesFileName := range overridesFiles {
		args = append(args, "-f")
		args = append(args, overridesFileName)
	}

	// Add the override strings
	if len(overrideStrings) > 0 {
		segs := strings.Split(overrideStrings, ",")
		for i := range segs {
			args = append(args, "--set")
			args = append(args, segs[i])
		}
	}

	// Perform istioctl call of type upgrade
	stdout, stderr, err = runIstioctl(log, args, "install")
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// IsInstalled returns true if Istio is installed
func IsInstalled(log *zap.SugaredLogger) (bool, error) {

	// Perform istioctl call of type upgrade
	stdout, _, err := VerifyInstall(log)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(stdout), "Istio is installed and verified successfully") {
		return true, nil
	}
	return false, nil
}

// VerifyInstall verifies the Istio installation
func VerifyInstall(log *zap.SugaredLogger) (stdout []byte, stderr []byte, err error) {
	args := []string{}

	// Perform istioctl call of type upgrade
	stdout, stderr, err = runIstioctl(log, args, "verify-install")
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
		log.Errorf("istioctl %s failed: %s", operationName, stderr)
		return stdout, stderr, err
	}

	log.Debugf("istioctl %s succeeded: %s", operationName, stdout)

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
