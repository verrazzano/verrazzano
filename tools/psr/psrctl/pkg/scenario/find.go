// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

// FindRunningScenarios returns the list of Scenarios that are running in the cluster.
func (m Manager) FindRunningScenarios() ([]Scenario, error) {
	scenarios := []Scenario{}

	cms, err := m.getAllConfigMaps()
	if err != nil {
		return nil, err
	}

	for _, cm := range cms {
		sc, err := m.getScenarioFromConfigmap(&cm)
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, *sc)
	}

	return scenarios, nil
}

// FindRunningScenarioByID returns the Scenario with the specified Scenario ID
func (m Manager) FindRunningScenarioByID(ID string) (*Scenario, error) {
	cm, err := m.getConfigMapByID(ID)
	if err != nil {
		return nil, err
	}
	sc, err := m.getScenarioFromConfigmap(cm)
	if err != nil {
		return nil, err
	}
	return sc, nil
}
