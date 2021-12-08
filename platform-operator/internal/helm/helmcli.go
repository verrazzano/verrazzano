// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	vzos "github.com/verrazzano/verrazzano/platform-operator/internal/os"
	"go.uber.org/zap"
)

// cmdRunner needed for unit tests
var runner vzos.CmdRunner = vzos.DefaultRunner{}

// Helm chart status values: unknown, deployed, uninstalled, superseded, failed, uninstalling, pending-install, pending-upgrade or pending-rollback
const ChartNotFound = "NotFound"
const ChartStatusDeployed = "deployed"
const ChartStatusPendingInstall = "pending-install"
const ChartStatusFailed = "failed"

// Package-level var and functions to allow overriding GetChartStatus for unit test purposes
type ChartStatusFnType func(releaseName string, namespace string) (string, error)

// HelmOverrides contains all of the overrides that gets passed to the helm cli runner
type HelmOverrides struct {
	SetOverrides       string   // for --set
	SetStringOverrides string   // for --set-string
	SetFileOverrides   string   // for --set-file
	FileOverrides      []string // for -f
}

var chartStatusFn ChartStatusFnType = getChartStatus

// SetChartStatusFunction Override the chart status function for unit testing
func SetChartStatusFunction(f ChartStatusFnType) {
	chartStatusFn = f
}

// SetDefaultChartStatusFunction Reset the chart status function
func SetDefaultChartStatusFunction() {
	chartStatusFn = getChartStatus
}

// Package-level var and functions to allow overriding getReleaseState for unit test purposes
type releaseStateFnType func(releaseName string, namespace string) (string, error)

var releaseStateFn releaseStateFnType = getReleaseState

// SetChartStateFunction Override the chart state function for unit testing
func SetChartStateFunction(f releaseStateFnType) {
	releaseStateFn = f
}

// SetDefaultChartStateFunction Reset the chart state function
func SetDefaultChartStateFunction() {
	releaseStateFn = getChartStatus
}

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
	log.Debugf("helm get values succeeded for %s", releaseName)

	return stdout, nil
}

// Upgrade will upgrade a Helm release with the specified charts.  The override files array
// are in order with the first files in the array have lower precedence than latter files.
func Upgrade(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides HelmOverrides) (stdout []byte, stderr []byte, err error) {
	// Helm upgrade command will apply the new chart, but use all the existing
	// overrides that we used during the install.
	args := []string{"--install"}

	// Do not pass the --reuse-values arg to 'helm upgrade'.  Instead, pass the
	// values retrieved from 'helm get values' with the -f arg to 'helm upgrade'. This is a workaround to avoid
	// a failed helm upgrade that results from a nil reference.  The nil reference occurs when a default value
	// is added to a new chart and new chart references the new value.
	for _, overridesFileName := range overrides.FileOverrides {
		if len(overridesFileName) == 0 {
			log.Debugf("Empty overrides file name for release %s", releaseName)
			continue
		}
		args = append(args, "-f")
		args = append(args, overridesFileName)
	}

	// Add the override strings
	if len(overrides.SetOverrides) > 0 {
		args = append(args, "--set")
		args = append(args, overrides.SetOverrides)
	}
	// Add the set-string override strings
	if len(overrides.SetStringOverrides) > 0 {
		args = append(args, "--set-string")
		args = append(args, overrides.SetStringOverrides)
	}
	if len(overrides.SetFileOverrides) > 0 {
		args = append(args, "--set-file")
		args = append(args, overrides.SetFileOverrides)
	}
	stdout, stderr, err = runHelm(log, releaseName, namespace, chartDir, "upgrade", wait, args, dryRun)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// Uninstall will uninstall the release in the specified namespace  using helm uninstall
func Uninstall(log *zap.SugaredLogger, releaseName string, namespace string, dryRun bool) (stdout []byte, stderr []byte, err error) {
	// Helm upgrade command will apply the new chart, but use all the existing
	// overrides that we used during the install.
	args := []string{}

	stdout, stderr, err = runHelm(log, releaseName, namespace, "", "uninstall", false, args, dryRun)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// runHelm is a helper function to execute the helm CLI and return a result
func runHelm(log *zap.SugaredLogger, releaseName string, namespace string, chartDir string, operation string, wait bool, args []string, dryRun bool) (stdout []byte, stderr []byte, err error) {
	cmdArgs := []string{operation, releaseName}
	if len(chartDir) > 0 {
		cmdArgs = append(cmdArgs, chartDir)
	}
	if dryRun {
		cmdArgs = append(cmdArgs, "--dry-run")
	}
	if wait {
		cmdArgs = append(cmdArgs, "--wait")
	}
	if namespace != "" {
		cmdArgs = append(cmdArgs, "--namespace")
		cmdArgs = append(cmdArgs, namespace)
	}
	cmdArgs = append(cmdArgs, args...)

	// Try to upgrade several times.  Sometimes upgrade fails with "already exists" or "no deployed release".
	// We have seen from tests that doing a retry will eventually succeed if these 2 errors occur.
	const maxRetry = 5
	for i := 1; i <= maxRetry; i++ {
		cmd := exec.Command("helm", cmdArgs...)

		// mask sensitive data before logging
		cmdStr := maskSensitiveData(cmd.String())
		log.Infof("Running command: %s", cmdStr)

		stdout, stderr, err = runner.Run(cmd)
		if err == nil {
			log.Debugf("helm %s for %s succeeded: %s", operation, releaseName, stdout)
			break
		}
		log.Errorf("helm %s for %s failed with stderr: %s", operation, releaseName, string(stderr))
		if i == maxRetry {
			return stdout, stderr, err
		}
		log.Warnf("Retrying %s for %s, attempt %v", operation, releaseName, i+1)
	}

	//  Log upgrade output
	log.Debugf("helm upgrade succeeded for %s", releaseName)
	return stdout, stderr, nil
}

// maskSensitiveData replaces sensitive data in a string with mask characters.
func maskSensitiveData(str string) string {
	const maskString = "*****"
	re := regexp.MustCompile(`[Pp]assword=(.+?)(?:,|\z)`)

	matches := re.FindAllStringSubmatch(str, -1)
	for _, match := range matches {
		if len(match) == 2 {
			str = strings.Replace(str, match[1], maskString, 1)
		}
	}

	return str
}

// IsReleaseFailed Returns true if the chart release state is marked 'failed'
func IsReleaseFailed(releaseName string, namespace string) (bool, error) {
	log := zap.S()
	releaseStatus, err := releaseStateFn(releaseName, namespace)
	if err != nil {
		log.Errorf("Getting status for chart %s/%s failed with stderr: %v\n", namespace, releaseName, err)
		return false, err
	}
	return releaseStatus == ChartStatusFailed, nil
}

// IsReleaseDeployed returns true if the release is deployed
func IsReleaseDeployed(releaseName string, namespace string) (found bool, err error) {
	log := zap.S()
	releaseStatus, err := chartStatusFn(releaseName, namespace)
	if err != nil {
		log.Errorf("Getting status for chart %s/%s failed with stderr: %v\n", namespace, releaseName, err)
		return false, err
	}
	switch releaseStatus {
	case ChartNotFound:
		log.Debugf("Chart %s/%s not found", namespace, releaseName)
	case ChartStatusDeployed:
		return true, nil
	}
	return false, nil
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
	stdout, stderr, err := runner.Run(cmd)

	if err == nil {
		log.Debugf("helm status stdout: %s", string(stdout))
		return true, nil
	}

	if strings.Contains(string(stderr), "not found") {
		return false, nil
	}
	log.Errorf("helm status for release %s failed with stderr: %s\n", releaseName, string(stderr))
	return false, err
}

// getChartStatus extracts the Helm deployment status of the specified chart from the JSON output as a string
func getChartStatus(releaseName string, namespace string) (string, error) {
	args := []string{"status", releaseName}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
		args = append(args, "-o")
		args = append(args, "json")
	}
	cmd := exec.Command("helm", args...)
	stdout, stderr, err := runner.Run(cmd)
	if err != nil {
		if strings.Contains(string(stderr), "not found") {
			return ChartNotFound, nil
		}
		return "", fmt.Errorf("helm status for release %s failed with stderr: %s", releaseName, string(stderr))
	}

	var statusInfo map[string]interface{}
	if err := json.Unmarshal(stdout, &statusInfo); err != nil {
		return "", err
	}

	if info, infoFound := statusInfo["info"].(map[string]interface{}); infoFound {
		if status, statusFound := info["status"].(string); statusFound {
			return strings.TrimSpace(status), nil
		}
	}
	return "", fmt.Errorf("No chart status found for %s/%s", namespace, releaseName)
}

// getReleaseState extracts the release state from an "ls -o json" command for a specific release/namespace
func getReleaseState(releaseName string, namespace string) (string, error) {
	args := []string{"ls"}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
		args = append(args, "-o")
		args = append(args, "json")
	}
	cmd := exec.Command("helm", args...)
	stdout, stderr, err := runner.Run(cmd)
	if err != nil {
		if strings.Contains(string(stderr), "not found") {
			return ChartNotFound, nil
		}
		return "", fmt.Errorf("helm status for release %s failed with stderr: %s", releaseName, string(stderr))
	}

	var statusInfo []map[string]interface{}
	if err := json.Unmarshal(stdout, &statusInfo); err != nil {
		return "", err
	}

	var status string
	for _, info := range statusInfo {
		release := info["name"].(string)
		if release == releaseName {
			status = info["status"].(string)
			break
		}
	}
	return strings.TrimSpace(status), nil
}

// SetCmdRunner sets the command runner as needed by unit tests
func SetCmdRunner(r vzos.CmdRunner) {
	runner = r
}

// SetDefaultRunner sets the command runner to default
func SetDefaultRunner() {
	runner = vzos.DefaultRunner{}
}
