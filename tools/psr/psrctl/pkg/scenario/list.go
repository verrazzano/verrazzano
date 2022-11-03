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
		sc.UsecaseOverridesDir = filepath.Join(filepath.Dir(f), usecaseOverrideDir)
		scenarios = append(scenarios, sc)
	}
	return scenarios, nil
}
