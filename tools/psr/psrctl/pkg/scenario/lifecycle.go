// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"fmt"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/embedded"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
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

// InstallScenario installs a Helm chart for each use case in the scenario
func InstallScenario(man *embedded.PsrManifests, sc *ScenarioManifest) (string, error) {
	// Helm install each use case
	var i int
	for _, uc := range sc.Usecases {
		helmOverrides := []helmcli.HelmOverrides{}

		// This is the usecase path, E.G. manifests/usecases/opensearch/getlogs/getlogs.yaml
		ucOverride := filepath.Join(man.UseCasesAbsDir, uc.UsecasePath)
		helmOverrides = append(helmOverrides, helmcli.HelmOverrides{FileOverride: ucOverride})

		// This is the scenario override path for the use case
		scOverride := filepath.Join(sc.ScenarioUsecaseOverridesDir, uc.OverrideFile)
		helmOverrides = append(helmOverrides, helmcli.HelmOverrides{FileOverride: scOverride})

		wType, err := readWorkerType(ucOverride)
		if err != nil {
			return "", err
		}

		// Build release name psr-<scenarioID>-workertype-<index>
		rname := fmt.Sprintf("psr-%s-%s-%v", sc.ID, wType, i)
		_, stderr, err := helmcli.Upgrade(vzlog.DefaultLogger(), rname, "default", man.WorkerChartAbsDir, true, false, helmOverrides)
		if err != nil {
			return string(stderr), err
		}
		i = i + 1
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
