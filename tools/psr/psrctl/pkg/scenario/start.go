// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"os"
	"path/filepath"
	"strings"

	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/manifest"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"
)

var StartUpgradeFunc = helmcli.Upgrade

// WorkerType is used to get the worker type from the worker use case YAML file.
// Note: struct and fields must be public for YAML unmarshal to work.
type WorkerType struct {
	Global struct {
		EnvVars struct {
			WorkerType string `json:"PSR_WORKER_TYPE"`
		}
	}
}

// StartScenario starts a Scenario
func (m ScenarioMananger) StartScenario(manifestMan manifest.ManifestManager, scman *manifest.ScenarioManifest, vzHelper helpers.VZHelper) (string, error) {
	helmReleases := []HelmRelease{}

	// Make sure the scenario is not running already
	running, err := m.FindRunningScenarios()
	if err != nil {
		return "", err
	}
	for _, sc := range running {
		if strings.EqualFold(sc.ID, scman.ID) {
			return "", fmt.Errorf("Scenario %s already running in namespace %s", sc.ID, m.Namespace)
		}
	}

	// Helm install each use case
	var i int
	for _, uc := range scman.Usecases {
		// Create the set of HelmOverrides, initialized from the manager settings
		helmOverrides := m.HelmOverrides

		// Build the usecase path, E.G. manifests/usecases/opensearch/getlogs/getlogs.yaml
		ucOverride := filepath.Join(manifestMan.Manifest.UseCasesAbsDir, uc.UsecasePath)
		helmOverrides = append(helmOverrides, helmcli.HelmOverrides{FileOverride: ucOverride})

		// Build scenario override path for the use case, E.G manifests/scenarios/opensearch/s1/usecase-overrides/getlogs-fast.yaml
		scOverride := filepath.Join(scman.ScenarioUsecaseOverridesAbsDir, uc.OverrideFile)
		helmOverrides = append(helmOverrides, helmcli.HelmOverrides{FileOverride: scOverride})

		wType, err := readWorkerType(ucOverride)
		if err != nil {
			return "", err
		}

		// Build release name psr-<scenarioID>-workertype-<index>
		relname := fmt.Sprintf("psr-%s-%s-%v", scman.ID, wType, i)

		if m.Verbose {
			fmt.Fprintf(vzHelper.GetOutputStream(), "Installing use case %s as Helm release %s/%s\n", uc.UsecasePath, m.Namespace, relname)
		}
		_, stderr, err := StartUpgradeFunc(m.Log, relname, m.Namespace, manifestMan.Manifest.WorkerChartAbsDir, true, m.DryRun, helmOverrides)
		if err != nil {
			return string(stderr), err
		}

		// Save the HelmRelease info
		helmRelease := HelmRelease{
			NamespacedName: types.NamespacedName{
				Namespace: m.Namespace,
				Name:      relname,
			},
			Usecase: uc,
		}
		helmReleases = append(helmReleases, helmRelease)
		i++
	}

	// Save the scenario in a ConfigMap
	sc := Scenario{
		Namespace:        m.Namespace,
		HelmReleases:     helmReleases,
		ScenarioManifest: scman,
	}
	_, err = m.createConfigMap(sc)
	if err != nil {
		return "", err
	}
	return "", nil
}

// readWorkerType reads the worker type from the use case worker YAML file at psr/manifests/usecases/...
func readWorkerType(ucOverride string) (string, error) {
	// Read in the manifests/usecases/.. YAML file to get the worker type
	var wt WorkerType
	data, err := os.ReadFile(ucOverride)
	if err != nil {
		return "nil", fmt.Errorf("Failed to read use case override file %s: %v", ucOverride, err)
	}
	if err := yaml.Unmarshal(data, &wt); err != nil {
		return "nil", fmt.Errorf("Failed to parse use case override file %s: %v", ucOverride, err)
	}
	if len(wt.Global.EnvVars.WorkerType) == 0 {
		return "nil", fmt.Errorf("Failed to find global.envVars.PSR_WORKER_TYPE in %s", ucOverride)
	}
	return wt.Global.EnvVars.WorkerType, nil
}
