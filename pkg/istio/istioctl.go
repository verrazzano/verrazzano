// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"os/exec"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/namespace"
	vzos "github.com/verrazzano/verrazzano/pkg/os"

	"github.com/pkg/errors"
)

// cmdRunner needed for unit tests
var runner vzos.CmdRunner = vzos.DefaultRunner{}

// fakeIstioInstalledRunner is used to test if Istio is installed
type fakeIstioInstalledRunner struct {
}

// Upgrade function gets called from istio_component to perform istio upgrade
func Upgrade(log vzlog.VerrazzanoLogger, imageOverrideString string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
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
	stdout, stderr, err = runIstioctl(log, args, "upgrade", true)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// Install does an Istio installation using one or more IstioOperator YAML files
func Install(log vzlog.VerrazzanoLogger, overrideStrings string, overridesFiles ...string) (stdout []byte, stderr []byte, err error) {
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
	stdout, stderr, err = runIstioctl(log, args, "install", true)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// Uninstall does an Istio uninstall removing the default revision installation. Istio CRDs are not removed.
func Uninstall(log vzlog.VerrazzanoLogger) (stdout []byte, stderr []byte, err error) {
	args := []string{"x", "uninstall", "--revision", "default", "-y"}

	// Perform istioctl call of type uninstall
	stdout, stderr, err = runIstioctl(log, args, "uninstall", true)
	if err != nil {
		return stdout, stderr, errors.Wrapf(err, "uninstall failed, stderr: %s", stderr)
	}

	return stdout, stderr, nil
}

// IsInstalled returns true if Istio is installed
func IsInstalled(log vzlog.VerrazzanoLogger) (bool, error) {
	// check to make sure we own the namespace first
	vzManaged, err := namespace.CheckIfVerrazzanoManagedNamespaceExists(constants.IstioSystemNamespace)
	if err != nil {
		return false, err
	}
	if !vzManaged {
		return false, nil
	}

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
func VerifyInstall(log vzlog.VerrazzanoLogger) (stdout []byte, stderr []byte, err error) {
	args := []string{"verify-install"}

	// Perform istioctl call of type upgrade
	stdout, stderr, err = runIstioctl(log, args, "verify-install", false)
	if err != nil {
		return stdout, stderr, errors.Wrapf(err, "verify-install failed, stderr: %s", stderr)
	}

	return stdout, stderr, nil
}

// runIstioctl will perform istioctl calls with specified arguments  for operations
// Note that operation name as of now does not affect the istioctl call (both upgrade and install call istioctl install)
// The operationName field is just used for visibility of operation in logging at the moment
func runIstioctl(log vzlog.VerrazzanoLogger, cmdArgs []string, operationName string, verbose bool) (stdout []byte, stderr []byte, err error) {
	cmd := exec.Command("istioctl", cmdArgs...)
	if verbose {
		log.Progressf("Running istioctl command: %s", cmd.String())
	}
	stdout, stderr, err = runner.Run(cmd)
	if err != nil {
		if verbose {
			log.Progressf("Failed running istioctl command %s: %s", cmd.String(), stderr)
		}
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
