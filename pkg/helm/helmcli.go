// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"encoding/json"
	"fmt"
	"helm.sh/helm/v3/pkg/release"
	"os/exec"
	"regexp"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzos "github.com/verrazzano/verrazzano/pkg/os"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
)

// Debug is set from a platform-operator arg and sets the helm --debug flag
var Debug bool

// cmdRunner needed for unit tests
var runner vzos.CmdRunner = vzos.DefaultRunner{}

// Helm chart status values: unknown, deployed, uninstalled, superseded, failed, uninstalling, pending-install, pending-upgrade or pending-rollback
const ChartNotFound = "NotFound"
const ChartStatusDeployed = "deployed"
const ChartStatusUninstalled = "uninstalled"
const ChartStatusPendingInstall = "pending-install"
const ChartStatusFailed = "failed"

// ChartStatusFnType - Package-level var and functions to allow overriding GetChartStatus for unit test purposes
type ChartStatusFnType func(releaseName string, namespace string) (string, error)

// HelmOverrides contains all of the overrides that gets passed to the helm cli runner
type HelmOverrides struct {
	SetOverrides       string // for --set
	SetStringOverrides string // for --set-string
	SetFileOverrides   string // for --set-file
	FileOverride       string // for -f
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

// ReleaseAppVersionFnType - Package-level var and functions to allow overriding GetReleaseAppVersion for unit test purposes
type ReleaseAppVersionFnType func(releaseName string, namespace string) (string, error)

var releaseAppVersionFn ReleaseAppVersionFnType = getReleaseAppVersion

// SetReleaseAppVersionFunction Override the GetReleaseAppVersion for unit testing
func SetReleaseAppVersionFunction(f ReleaseAppVersionFnType) {
	releaseAppVersionFn = f
}

// SetDefaultReleaseAppVersionFunction Reset the GetReleaseAppVersion function
func SetDefaultReleaseAppVersionFunction() {
	releaseAppVersionFn = getReleaseAppVersion
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
func GetValues(log vzlog.VerrazzanoLogger, releaseName string, namespace string) ([]byte, error) {
	// Helm get values command will get the current set values for the installed chart.
	// The output will be used as input to the helm upgrade command.
	args := []string{"get", "values", releaseName}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
		args = append(args, "-o")
		args = append(args, "yaml")
	}

	cmd := exec.Command("helm", args...)
	log.Debugf("Running command to get Helm values: %s", cmd.String())
	stdout, stderr, err := runner.Run(cmd)
	if err != nil {
		log.Errorf("Failed to get Helm values for %s: stderr %s", releaseName, string(stderr))
		return nil, err
	}

	//  Log get values output
	log.Debugf("Successfully fetched Helm get values %s", releaseName)

	return stdout, nil
}

// GetValuesMap will run 'helm get values' command and return the output from the command as a map of Objects.
func GetValuesMap(log vzlog.VerrazzanoLogger, releaseName string, namespace string) (map[string]interface{}, error) {
	// Helm get values command will get the current set values for the installed chart.
	// The output will be used as input to the helm upgrade command.
	args := []string{"get", "values", releaseName}
	if namespace != "" {
		args = append(args, "--namespace")
		args = append(args, namespace)
		args = append(args, "-o")
		args = append(args, "json")
	}

	cmd := exec.Command("helm", args...)
	log.Debugf("Running command to get Helm values: %s", cmd.String())
	stdout, stderr, err := runner.Run(cmd)
	if err != nil {
		log.Errorf("Failed to get Helm values for %s: stderr %s", releaseName, string(stderr))
		return nil, err
	}

	var valuesMap map[string]interface{}
	if err := json.Unmarshal(stdout, &valuesMap); err != nil {
		return map[string]interface{}{}, err
	}
	//  Log get values output
	log.Debugf("Successfully fetched Helm get values %s", releaseName)
	return valuesMap, nil
}

// Upgrade will upgrade a Helm release with the specified charts.  The override files array
// are in order with the first files in the array have lower precedence than latter files.
func Upgrade(log vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides []HelmOverrides) (stdout []byte, stderr []byte, err error) {
	// Helm upgrade command will apply the new chart, but use all the existing
	// overrides that we used during the install.
	args := []string{"--install"}

	// Do not pass the --reuse-values arg to 'helm upgrade'.  Instead, pass the
	// values retrieved from 'helm get values' with the -f arg to 'helm upgrade'. This is a workaround to avoid
	// a failed helm upgrade that results from a nil reference.  The nil reference occurs when a default value
	// is added to a new chart and new chart references the new value.
	for _, override := range overrides {
		// Add file overrides
		if len(override.FileOverride) > 0 {
			args = append(args, "-f")
			args = append(args, override.FileOverride)
		}
		// Add the override strings
		if len(override.SetOverrides) > 0 {
			args = append(args, "--set")
			args = append(args, override.SetOverrides)
		}
		// Add the set-string override strings
		if len(override.SetStringOverrides) > 0 {
			args = append(args, "--set-string")
			args = append(args, override.SetStringOverrides)
		}
		// Add the set-file override strings
		if len(override.SetFileOverrides) > 0 {
			args = append(args, "--set-file")
			args = append(args, override.SetFileOverrides)
		}
	}

	stdout, stderr, err = runHelm(log, releaseName, namespace, chartDir, "upgrade", wait, args, dryRun)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// Uninstall will uninstall the release in the specified namespace  using helm uninstall
func Uninstall(log vzlog.VerrazzanoLogger, releaseName string, namespace string, dryRun bool) (stdout []byte, stderr []byte, err error) {
	// Helm upgrade command will apply the new chart, but use all the existing
	// overrides that we used during the install.
	var args []string

	stdout, stderr, err = runHelm(log, releaseName, namespace, "", "uninstall", false, args, dryRun)
	if err != nil {
		return stdout, stderr, err
	}

	return stdout, stderr, nil
}

// runHelm is a helper function to execute the helm CLI and return a result
func runHelm(log vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, operation string, wait bool, args []string, dryRun bool) (stdout []byte, stderr []byte, err error) {
	cmdArgs := []string{operation, releaseName}
	if len(chartDir) > 0 {
		cmdArgs = append(cmdArgs, chartDir)
	}
	if Debug {
		cmdArgs = append(cmdArgs, "--debug")
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
		if i == 1 {
			log.Progressf("Running Helm command %s for release %s", cmdStr, releaseName)
		} else {
			log.Progressf("Re-running Helm command for release %s", releaseName)
		}

		stdout, stderr, err = runner.Run(cmd)
		if err == nil {
			log.Debugf("Successfully ran Helm command for operation %s and release %s", operation, releaseName)
			break
		}
		if i == 1 || i == maxRetry {
			log.Errorf("Failed running Helm command for release %s: stderr %s",
				releaseName, string(stderr))
			return stdout, stderr, err
		}
		log.Infof("Failed running Helm command for operation %s and release %s. Retrying %s of %s", operation, releaseName, i+1, maxRetry)
	}

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

// GetReleaseStatus returns the release status
func GetReleaseStatus(releaseName string, namespace string) (status string, err error) {
	log := zap.S()
	releaseStatus, err := chartStatusFn(releaseName, namespace)
	if err != nil {
		log.Errorf("Getting status for chart %s/%s failed with stderr: %v\n", namespace, releaseName, err)
		return "", err
	}
	switch releaseStatus {
	case ChartNotFound:
		log.Debugf("Chart %s/%s not found", namespace, releaseName)
	case release.StatusSuperseded.String():
		return release.StatusSuperseded.String(), nil
	case release.StatusDeployed.String():
		return release.StatusDeployed.String(), nil
	case release.StatusFailed.String():
		return release.StatusFailed.String(), nil
	case release.StatusPendingInstall.String():
		return release.StatusPendingInstall.String(), nil
	case release.StatusPendingRollback.String():
		return release.StatusPendingRollback.String(), nil
	case release.StatusPendingUpgrade.String():
		return release.StatusPendingUpgrade.String(), nil
	case release.StatusUninstalled.String():
		return release.StatusUninstalled.String(), nil
	case release.StatusUninstalling.String():
		return release.StatusUninstalling.String(), nil
	case release.StatusUnknown.String():
		return release.StatusUnknown.String(), nil
	}
	return "", nil
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
	statusInfo, err := getReleases(namespace)
	if err != nil {
		if errors.IsNotFound(err) {
			return ChartNotFound, nil
		}
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

// GetReleaseAppVersion - public function to execute releaseAppVersionFn
func GetReleaseAppVersion(releaseName string, namespace string) (string, error) {
	return releaseAppVersionFn(releaseName, namespace)
}

// GetReleaseStringValues - Returns a subset of Helm release values as a map of strings
func GetReleaseStringValues(log vzlog.VerrazzanoLogger, valueKeys []string, releaseName string, namespace string) (map[string]string, error) {
	values, err := GetReleaseValues(log, valueKeys, releaseName, namespace)
	if err != nil {
		return map[string]string{}, err
	}
	returnVals := map[string]string{}
	for key, val := range values {
		returnVals[key] = fmt.Sprintf("%v", val)
	}
	return returnVals, err
}

// GetReleaseValues - Returns a subset of Helm release values as a map of objects
func GetReleaseValues(log vzlog.VerrazzanoLogger, valueKeys []string, releaseName string, namespace string) (map[string]interface{}, error) {
	isDeployed, err := IsReleaseDeployed(releaseName, namespace)
	if err != nil {
		return map[string]interface{}{}, err
	}
	var values = map[string]interface{}{}
	if isDeployed {
		valuesMap, err := GetValuesMap(log, releaseName, namespace)
		if err != nil {
			return map[string]interface{}{}, err
		}
		for _, valueKey := range valueKeys {
			if mapVal, ok := valuesMap[valueKey]; ok {
				log.Debugf("Found value for %s: %v", valueKey, mapVal)
				values[valueKey] = mapVal
			}
		}
	}
	return values, nil
}

// getReleaseAppVersion extracts the release app_version from a "ls -o json" command for a specific release/namespace
func getReleaseAppVersion(releaseName string, namespace string) (string, error) {
	statusInfo, err := getReleases(namespace)
	if err != nil {
		if err.Error() == ChartNotFound {
			return ChartNotFound, nil
		}
		return "", err
	}

	var status string
	for _, info := range statusInfo {
		release := info["name"].(string)
		if release == releaseName {
			status = info["app_version"].(string)
			break
		}
	}
	return strings.TrimSpace(status), nil
}

func getReleases(namespace string) ([]map[string]interface{}, error) {
	var statusInfo []map[string]interface{}

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
			return statusInfo, fmt.Errorf(ChartNotFound)
		}
		return statusInfo, fmt.Errorf("helm status for namespace %s failed with stderr: %s", namespace, string(stderr))
	}

	if err := json.Unmarshal(stdout, &statusInfo); err != nil {
		return statusInfo, err
	}

	return statusInfo, nil
}

// SetCmdRunner sets the command runner as needed by unit tests
func SetCmdRunner(r vzos.CmdRunner) {
	runner = r
}

// SetDefaultRunner sets the command runner to default
func SetDefaultRunner() {
	runner = vzos.DefaultRunner{}
}
