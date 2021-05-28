// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"os/exec"
	"strings"

	vz_os "github.com/verrazzano/verrazzano/platform-operator/internal/util/os"
	"go.uber.org/zap"
)

// cmdRunner needed for unit tests
var runner vz_os.CmdRunner = vz_os.DefaultRunner{}

// Upgrade will upgrade a Helm release with the specified charts.  The overrideFiles array
// are in order with the first files in the array have lower precedence than latter files.
func Upgrade(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, overrideFile string, overrides string) (stdout []byte, stderr []byte, err error) {
	// Helm upgrade command will apply the new chart, but use all the existing
	// overrides that we used during the install.
	args := []string{"upgrade", releaseName, chartDir}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
	}

	// If overrides are provided the specify --reuse-values
	if len(overrideFile) > 0 || len(overrides) > 0 {
		args = append(args, "--reuse-values")
	}
	// Add the override files
	if len(overrideFile) > 0 {
		args = append(args, "-f")
		args = append(args, overrideFile)
	}
	// Add the override strings
	if len(overrides) > 0 {
		args = append(args, "--set")
		args = append(args, overrides)
	}
	cmd := exec.Command("helm", args...)
	log.Infof("Running command: %s",cmd.String())
	stdout, stderr, err = runner.Run(cmd)
	if err != nil {
		log.Errorf("helm upgrade for %s failed with stderr: %s", releaseName, string(stderr))
		return stdout, stderr, err
	}

	//  Log upgrade output
	log.Infof("helm upgrade succeeded for %s", releaseName)
	return stdout, stderr, nil
}

// IsReleaseInstalled returns true if the release is installed
func IsReleaseInstalled(releaseName string, namespace string) (found bool, err error) {
	log := zap.S()

	args := []string{"status", releaseName}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
	}
	cmd := exec.Command("helm", args...)
	_, stderr, err := runner.Run(cmd)
	if err == nil {
		return true, nil
	}
	if strings.Contains(string(stderr), "not found") {
		return false, nil
	}
	log.Errorf("helm status for release %s failed with stderr: %s\n", releaseName, string(stderr))
	return false, err
}

// SetCmdRunner sets the command runner as needed by unit tests
func SetCmdRunner(r vz_os.CmdRunner) {
	runner = r
}

// SetDefaultRunner sets the command runner to default
func SetDefaultRunner() {
	runner = vz_os.DefaultRunner{}
}
