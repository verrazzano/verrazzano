package scenario

// FindRunningScenarios returns the list of Scenarios that are running in the cluster.
func (m ScenarioMananger) FindRunningScenarios() ([]Scenario, error) {
	scenarios := []Scenario{}

	cms, err := m.getAllConfigMaps()
	if err != nil {
		return nil, err
	}

	for i := range cms {
		sc, err := m.getScenarioFromConfigmap(&cms[i])
		if err != nil {
			return nil, err
		}
		scenarios = append(scenarios, *sc)
	}

	return scenarios, nil
}

// FindRunningScenarioByID returns the Scenario with the specified Scenario ID
func (m ScenarioMananger) FindRunningScenarioByID(ID string) (*Scenario, error) {
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
