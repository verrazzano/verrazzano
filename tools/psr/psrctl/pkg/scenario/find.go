// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import "strings"

// FindScenarioById finds a Scenario by ID
func FindScenarioById(scenarioAbsDir string, ID string) (*Scenario, error) {
	return findScenario(scenarioAbsDir, func(scenario Scenario) bool {
		return strings.EqualFold(scenario.ID, ID)
	})
}

// FindScenarioById finds a Scenario by Name
func FindScenarioByName(scenarioAbsDir string, name string) (*Scenario, error) {
	return findScenario(scenarioAbsDir, func(scenario Scenario) bool {
		return strings.EqualFold(scenario.Name, name)
	})
}

func findScenario(scenarioAbsDir string, f func(Scenario) bool) (*Scenario, error) {
	scList, err := ListAvailableScenarios(scenarioAbsDir)
	if err != nil {
		return nil, err
	}
	for i, sc := range scList {
		if f(sc) {
			return &scList[i], nil
		}
	}
	return nil, nil
}
