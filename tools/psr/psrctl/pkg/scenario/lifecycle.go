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

// workerType is used to get the worker type from the worker use case YAML file.
type workerType struct {
	global struct {
		envVars struct {
			PSR_WORKER_TYPE string
		}
	}
}

func InstallScenario(man *embedded.PsrManifests, sc *Scenario) (string, error) {
	// Helm install each use case
	var i int
	for _, uc := range sc.Usecases {
		helmOverrides := []helmcli.HelmOverrides{}
		// This is the usecase path, E.G. manifests/usecases/opensearch/getlogs/getlogs.yaml
		ucOverride := filepath.Join(man.UseCasesAbsDir, uc.UsecasePath)
		helmOverrides = append(helmOverrides, helmcli.HelmOverrides{FileOverride: ucOverride})

		// This is the scenario override path for the use case
		scOverride := filepath.Join(sc.UsecaseOverridesDir, uc.UsecasePath)
		helmOverrides = append(helmOverrides, helmcli.HelmOverrides{FileOverride: scOverride})

		// Read in the manifests/usecases/.. YAML file to get the worker type
		var wt workerType
		data, err := os.ReadFile(ucOverride)
		if err != nil {
			return "nil", fmt.Errorf("Failed to read use case override file %s: %v", ucOverride, err)
		}
		if err := yaml.Unmarshal(data, &wt); err != nil {
			return "nil", fmt.Errorf("Failed to parse use case override file %s: %v", ucOverride, err)
		}
		if len(wt.global.envVars.PSR_WORKER_TYPE) == 0 {
			return "nil", fmt.Errorf("Failed to find global.envVars.PSR_WORKER_TYPE in %s", ucOverride)
		}
		// Build release name psr-<scenarioID>-workertype-<index>
		rname := fmt.Sprintf("psr-%s-%s-%v", sc.ID, wt.global.envVars.PSR_WORKER_TYPE, i)
		_, stderr, err := helmcli.Upgrade(vzlog.DefaultLogger(), rname, "default", man.WorkerChartAbsDir, true, false, helmOverrides)
		if err != nil {
			return string(stderr), err
		}
	}
	return "", nil
}
