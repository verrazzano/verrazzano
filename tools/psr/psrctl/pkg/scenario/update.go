// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"fmt"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
)

// UpdateScenario updates a Scenario
func (m Manager) UpdateScenario(ID string) (string, error) {
	// Make sure the scenario is running
	scenario, err := m.FindRunningScenarioByID(ID)
	if err != nil {
		return "", err
	}

	// Helm upgrade each use case
	for _, hr := range scenario.HelmReleases {
		// Create the set of HelmOverrides, initialized from the manager settings
		helmOverrides := m.HelmOverrides

		if m.Verbose {
			fmt.Printf("Updating Helm release %s/%s\n", m.Namespace, hr.Name)
		}
		_, stderr, err := helmcli.Upgrade(m.Log, hr.Name, m.Namespace, m.Manifest.WorkerChartAbsDir, true, m.DryRun, helmOverrides)
		if err != nil {
			return string(stderr), err
		}
	}
	return "", nil
}
