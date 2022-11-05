// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"fmt"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strings"
)

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
func (m Manager) StartScenario(scman *ScenarioManifest) (string, error) {
	helmReleases := []types.NamespacedName{}

	// Make sure the scenario is not running already
	running, err := m.FindRunningScenarios()
	if err != nil {
		return "", err
	}
	for _, sc := range running {
		if strings.EqualFold(sc.ID, scman.ID) {
			return "", fmt.Errorf("Scenario %s already running", sc.ID)
		}
	}

	// Helm install each use case
	var i int
	for _, uc := range scman.Usecases {
		helmOverrides := []helmcli.HelmOverrides{}

		// This is the usecase path, E.G. manifests/usecases/opensearch/getlogs/getlogs.yaml
		ucOverride := filepath.Join(m.Manifest.UseCasesAbsDir, uc.UsecasePath)
		helmOverrides = append(helmOverrides, helmcli.HelmOverrides{FileOverride: ucOverride})

		// This is the scenario override path for the use case
		scOverride := filepath.Join(scman.ScenarioUsecaseOverridesDir, uc.OverrideFile)
		helmOverrides = append(helmOverrides, helmcli.HelmOverrides{FileOverride: scOverride})

		wType, err := readWorkerType(ucOverride)
		if err != nil {
			return "", err
		}

		// Build release name psr-<scenarioID>-workertype-<index>
		relname := fmt.Sprintf("psr-%s-%s-%v", scman.ID, wType, i)
		_, stderr, err := helmcli.Upgrade(m.Log, relname, m.Namespace, m.Manifest.WorkerChartAbsDir, true, m.DryRun, helmOverrides)
		if err != nil {
			return string(stderr), err
		}
		helmReleases = append(helmReleases, types.NamespacedName{
			Namespace: m.Namespace,
			Name:      relname,
		})
		i = i + 1
	}

	// Save the scenario in a ConfigMap
	sc := Scenario{
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
