// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"github.com/verrazzano/verrazzano/pkg/files"
	"os"
	"path/filepath"
	"regexp"
	"sigs.k8s.io/yaml"
)

// The required use case overrides directory
const usecaseOverrideDir = "usecase-overrides"

// ListAvailableScenarios returns the list of scenarios located in
// psr/manifests/scenarios.  By convention, a scenario directory must have
// a scenario.yaml file which describes the scenario. It must also have
// a subdirectory named usecase-overrides containing the override parameters for
// each use case. The name of the parent directory, for example s1, is irrelevant.
func ListAvailableScenarios(scenarioAbsDir string) ([]Scenario, error) {
	scenarios := []Scenario{}
	sfiles, err := files.GetMatchingFiles(scenarioAbsDir, regexp.MustCompile("scenario.yaml"))
	if err != nil {
		return nil, err
	}
	for _, f := range sfiles {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var sc Scenario
		if err := yaml.Unmarshal(data, &sc); err != nil {
			return nil, err
		}

		// Build the parent directory name that has the scenario.yaml.
		sc.ScenarioUsecaseOverridesDir = filepath.Join(filepath.Dir(f), usecaseOverrideDir)
		scenarios = append(scenarios, sc)
	}
	return scenarios, nil
}
