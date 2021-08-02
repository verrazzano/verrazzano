// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"os/exec"
	"strings"

	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/util/os"
	"go.uber.org/zap"
)

// cmdRunner needed for unit tests
var runner vzos.CmdRunner = vzos.DefaultRunner{}

// GetValues will run 'helm get values' command and return the output from the command.
func GetValues(log *zap.SugaredLogger, releaseName string, namespace string) ([]byte, error) {
	// Helm get values command will get the current set values for the installed chart.
	// The output will be used as input to the helm upgrade command.
	args := []string{"get", "values", releaseName}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
	}

	cmd := exec.Command("helm", args...)
	log.Infof("Running command: %s", cmd.String())
	stdout, stderr, err := runner.Run(cmd)
	if err != nil {
		log.Errorf("helm get values for %s failed with stderr: %s", releaseName, string(stderr))
		return nil, err
	}

	//  Log get values output
	log.Infof("helm get values succeeded for %s", releaseName)

	return stdout, nil
}

// Upgrade will upgrade a Helm release with the specified charts.  The overrideFiles array
// are in order with the first files in the array have lower precedence than latter files.
func Upgrade(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, overrideFile string, overrides string, existingValuesFile string) (stdout []byte, stderr []byte, err error) {
	// Helm upgrade command will apply the new chart, but use all the existing
	// overrides that we used during the install.
	args := []string{"upgrade", releaseName, chartDir}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
	}

	// Do not pass the --reuse-values arg to 'helm upgrade'.  Instead, pass the
	// values retrieved from 'helm get values' with the -f arg to 'helm upgrade'. This is a workaround to avoid
	// a failed helm upgrade that results from a nil reference.  The nil reference occurs when a default value
	// is added to a new chart and new chart references the new value.
	args = append(args, "-f")
	args = append(args, existingValuesFile)

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
	log.Infof("Running command: %s", cmd.String())
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
func SetCmdRunner(r vzos.CmdRunner) {
	runner = r
}

// SetDefaultRunner sets the command runner to default
func SetDefaultRunner() {
	runner = vzos.DefaultRunner{}
}
