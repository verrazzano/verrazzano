// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helm

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	yaml2 "github.com/verrazzano/verrazzano/pkg/yaml"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/strvals"
	"sigs.k8s.io/yaml"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"go.uber.org/zap"
)

// Debug is set from a platform-operator arg and sets the helm --debug flag
var Debug bool

// Helm chart status values: unknown, deployed, uninstalled, superseded, failed, uninstalling, pending-install, pending-upgrade or pending-rollback
const ChartNotFound = "NotFound"
const ChartStatusDeployed = "deployed"
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

type ActionConfigFnType func(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error)

var actionConfigFn ActionConfigFnType = getActionConfig

func SetActionConfigFunction(f ActionConfigFnType) {
	actionConfigFn = f
}

// SetDefaultActionConfigFunction Resets the action config function
func SetDefaultActionConfigFunction() {
	actionConfigFn = getActionConfig
}

type LoadChartFnType func(chartDir string) (*chart.Chart, error)

var loadChartFn LoadChartFnType = loadChart

func SetLoadChartFunction(f LoadChartFnType) {
	loadChartFn = f
}

func SetDefaultLoadChartFunction() {
	loadChartFn = loadChart
}

// GetValuesMap will run 'helm get values' command and return the output from the command.
func GetValuesMap(log vzlog.VerrazzanoLogger, releaseName string, namespace string) (map[string]interface{}, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig, err := actionConfigFn(log, settings, namespace)
	if err != nil {
		return nil, err
	}

	client := action.NewGetValues(actionConfig)
	vals, err := client.Run(releaseName)
	if err != nil {
		return nil, err
	}

	return vals, nil
}

// GetValues will run 'helm get values' command and return the output from the command.
func GetValues(log vzlog.VerrazzanoLogger, releaseName string, namespace string) ([]byte, error) {
	vals, err := GetValuesMap(log, releaseName, namespace)
	if err != nil {
		return nil, err
	}

	yamlValues, err := yaml.Marshal(vals)
	if err != nil {
		return nil, err
	}
	return yamlValues, nil
}

// Upgrade will upgrade a Helm helmRelease with the specified charts.  The override files array
// are in order with the first files in the array have lower precedence than latter files.
func Upgrade(log vzlog.VerrazzanoLogger, releaseName string, namespace string, chartDir string, wait bool, dryRun bool, overrides []HelmOverrides) (*release.Release, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig, err := actionConfigFn(log, settings, namespace)
	if err != nil {
		return nil, err
	}

	p := getter.All(settings)
	vals, err := mergeValues(overrides, p)
	if err != nil {
		return nil, err
	}
	// load chart from the path
	chart, err := loadChartFn(chartDir)
	if err != nil {
		return nil, err
	}
	installed, err := IsReleaseInstalled(releaseName, namespace)
	if err != nil {
		return nil, err
	}

	var rel *release.Release
	if installed {
		// upgrade it
		log.Infof("Starting Helm upgrade of release %s in namespace %s with overrides: %v", releaseName, namespace, overrides)
		client := action.NewUpgrade(actionConfig)
		client.Namespace = namespace
		client.DryRun = dryRun
		client.Wait = wait
		client.MaxHistory = 1

		rel, err = client.Run(releaseName, chart, vals)
		if err != nil {
			log.Errorf("Failed running Helm command for release %s",
				releaseName)
			return nil, err
		}
	} else {
		log.Infof("Starting Helm installation of release %s in namespace %s with overrides: %v", releaseName, namespace, overrides)
		client := action.NewInstall(actionConfig)
		client.Namespace = namespace
		client.ReleaseName = releaseName
		client.DryRun = dryRun
		client.Replace = true
		client.Wait = wait

		rel, err = client.Run(chart, vals)
		if err != nil {
			log.Errorf("Failed running Helm command for release %s: %v",
				releaseName, err.Error())
			return nil, err
		}
	}

	log.Infof("Helm upgraded/installed %s in namespace %s", rel.Name, rel.Namespace)

	return rel, nil
}

// Uninstall will uninstall the helmRelease in the specified namespace  using helm uninstall
func Uninstall(log vzlog.VerrazzanoLogger, releaseName string, namespace string, dryRun bool) (err error) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig, err := actionConfigFn(log, settings, namespace)
	if err != nil {
		return err
	}

	client := action.NewUninstall(actionConfig)
	client.DryRun = dryRun

	_, err = client.Run(releaseName)
	if err != nil {
		log.Errorf("Error uninstalling release %s: %s", releaseName, err.Error())
		return err
	}

	return nil
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

// IsReleaseFailed Returns true if the chart helmRelease state is marked 'failed'
func IsReleaseFailed(releaseName string, namespace string) (bool, error) {
	log := zap.S()
	releaseStatus, err := getReleaseState(releaseName, namespace)
	if err != nil {
		log.Errorf("Getting status for chart %s/%s failed", namespace, releaseName)
		return false, err
	}
	return releaseStatus == ChartStatusFailed, nil
}

// IsReleaseDeployed returns true if the helmRelease is deployed
func IsReleaseDeployed(releaseName string, namespace string) (found bool, err error) {
	log := zap.S()
	releaseStatus, err := getChartStatus(releaseName, namespace)
	if err != nil {
		log.Errorf("Getting status for chart %s/%s failed with error: %v\n", namespace, releaseName, err)
		return false, err
	}
	switch releaseStatus {
	case ChartNotFound:
		log.Debugf("releasename=%s/%s; status= %s", namespace, releaseName, releaseStatus)
	case ChartStatusDeployed:
		return true, nil
	}
	return false, nil
}

// GetReleaseStatus returns the helmRelease status
func GetReleaseStatus(log vzlog.VerrazzanoLogger, releaseName string, namespace string) (status string, err error) {
	releaseStatus, err := getChartStatus(releaseName, namespace)
	if err != nil {
		log.ErrorfNewErr("Failed getting status for chart %s/%s with stderr: %v\n", namespace, releaseName, err)
		return "", err
	}
	if releaseStatus == ChartNotFound {
		log.Debugf("Chart %s/%s not found", namespace, releaseName)
	}
	return releaseStatus, nil
}

// IsReleaseInstalled returns true if the helmRelease is installed
func IsReleaseInstalled(releaseName string, namespace string) (found bool, err error) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig, err := actionConfigFn(vzlog.DefaultLogger(), settings, namespace)
	if err != nil {
		return false, err
	}

	client := action.NewStatus(actionConfig)
	helmRelease, err := client.Run(releaseName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, err
	}
	return release.StatusDeployed == helmRelease.Info.Status, nil
}

// getChartStatus extracts the Helm deployment status of the specified chart from the JSON output as a string
func getChartStatus(releaseName string, namespace string) (string, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig, err := actionConfigFn(vzlog.DefaultLogger(), settings, namespace)
	if err != nil {
		return "", err
	}

	client := action.NewStatus(actionConfig)
	helmRelease, err := client.Run(releaseName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ChartNotFound, nil
		}
		return "", err
	}

	return helmRelease.Info.Status.String(), nil
}

// getReleaseState extracts the helmRelease state from an "ls -o json" command for a specific helmRelease/namespace
func getReleaseState(releaseName string, namespace string) (string, error) {
	releases, err := getReleases(namespace)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return ChartNotFound, nil
		}
		return "", err
	}

	status := ""
	for _, info := range releases {
		release := info.Name
		if release == releaseName {
			status = info.Info.Status.String()
			break
		}
	}
	return strings.TrimSpace(status), nil
}

// GetReleaseAppVersion - public function to execute releaseAppVersionFn
func GetReleaseAppVersion(releaseName string, namespace string) (string, error) {
	return getReleaseAppVersion(releaseName, namespace)
}

// GetReleaseStringValues - Returns a subset of Helm helmRelease values as a map of strings
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

// GetReleaseValues - Returns a subset of Helm helmRelease values as a map of objects
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

// getReleaseAppVersion extracts the helmRelease app_version from a "ls -o json" command for a specific helmRelease/namespace
func getReleaseAppVersion(releaseName string, namespace string) (string, error) {
	releases, err := getReleases(namespace)
	if err != nil {
		if err.Error() == ChartNotFound {
			return ChartNotFound, nil
		}
		return "", err
	}

	var status string
	for _, info := range releases {
		release := info.Name
		if release == releaseName {
			status = info.Chart.AppVersion()
			break
		}
	}
	return strings.TrimSpace(status), nil
}

func getReleases(namespace string) ([]*release.Release, error) {
	settings := cli.New()
	settings.SetNamespace(namespace)
	actionConfig, err := actionConfigFn(vzlog.DefaultLogger(), settings, namespace)
	if err != nil {
		return nil, err
	}

	client := action.NewList(actionConfig)
	client.AllNamespaces = false
	client.All = true
	client.StateMask = action.ListAll

	releases, err := client.Run()
	if err != nil {
		return nil, err
	}

	return releases, nil
}

func getActionConfig(log vzlog.VerrazzanoLogger, settings *cli.EnvSettings, namespace string) (*action.Configuration, error) {
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(settings.RESTClientGetter(), namespace, os.Getenv("HELM_DRIVER"), log.Debugf); err != nil {
		return nil, err
	}
	actionConfig.Releases.MaxHistory = 1
	return actionConfig, nil
}

func loadChart(chartDir string) (*chart.Chart, error) {
	return loader.Load(chartDir)
}

// readFile load a file using a URI scheme provider
func readFile(filePath string, p getter.Providers) ([]byte, error) {
	if strings.TrimSpace(filePath) == "-" {
		return io.ReadAll(os.Stdin)
	}
	u, err := url.Parse(filePath)
	if err != nil {
		return nil, err
	}

	g, err := p.ByScheme(u.Scheme)
	if err != nil {
		return os.ReadFile(filePath)
	}
	data, err := g.Get(filePath, getter.WithURL(filePath))
	if err != nil {
		return nil, err
	}
	return data.Bytes(), err
}

// mergeValues merges values from the specified overrides
func mergeValues(overrides []HelmOverrides, p getter.Providers) (map[string]interface{}, error) {
	base := map[string]interface{}{}

	// User specified a values files via -f/--values
	for _, override := range overrides {
		if len(override.FileOverride) > 0 {
			currentMap := map[string]interface{}{}

			bytes, err := readFile(override.FileOverride, p)
			if err != nil {
				return nil, err
			}

			if err := yaml.Unmarshal(bytes, &currentMap); err != nil {
				return nil, err
			}
			// Merge with the previous map
			yaml2.MergeMaps(base, currentMap)
		}

		// User specified a value via --set
		if len(override.SetOverrides) > 0 {
			if err := strvals.ParseInto(override.SetOverrides, base); err != nil {
				return nil, err
			}
		}

		// User specified a value via --set-string
		if len(override.SetStringOverrides) > 0 {
			if err := strvals.ParseIntoString(override.SetStringOverrides, base); err != nil {
				return nil, err
			}
		}

		// User specified a value via --set-file
		if len(override.SetFileOverrides) > 0 {
			reader := func(rs []rune) (interface{}, error) {
				bytes, err := readFile(string(rs), p)
				if err != nil {
					return nil, err
				}
				return string(bytes), err
			}
			if err := strvals.ParseIntoFile(override.SetFileOverrides, base, reader); err != nil {
				return nil, err
			}
		}
	}

	return base, nil
}
