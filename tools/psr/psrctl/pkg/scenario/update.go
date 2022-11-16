// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"fmt"
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	"os"
	"path/filepath"
)

// UpdateScenario updates a running Scenario
// The scenario manifest directory can be different that the one used to start the
// scenario.  However, the scenario.yaml must be identical.  In fact, the scenario.yaml
// is ignored during update, the code uses the scenario YAML information stored in the ConfigMap.
func (m Manager) UpdateScenario(scman *ScenarioManifest) (string, error) {
	// Make sure the scenario is running
	scenario, err := m.FindRunningScenarioByID(scman.ID)
	if err != nil {
		return "", err
	}

	// Helm upgrade each use case
	for _, hr := range scenario.HelmReleases {
		stderr, err := m.doHelmUpgrade(scman, hr)
		if err != nil {
			return stderr, err
		}
	}
	return "", nil
}

// doHelmUpgrade runs the Helm upgrade command, applying helm overrides.
func (m Manager) doHelmUpgrade(scman *ScenarioManifest, hr HelmRelease) (string, error) {
	// Create the set of HelmOverrides, initialized from the manager settings
	helmOverrides := m.HelmOverrides

	// Get existing Helm values for the release.  These need to be passed since --reuse-values is not used.
	stdout, err := helmcli.GetValues(m.Log, hr.Name, hr.Namespace)
	if err != nil {
		return "", err
	}

	// Create a temp file with the existing values and add to helm overrides
	tmpPath := filepath.Join(m.Manifest.RootTmpDir, fmt.Sprintf("upgrade-%s-%s", hr.Namespace, hr.Name))
	// delete any existing update tmp file, shouldn't exist but just in case
	os.RemoveAll(tmpPath)
	err = os.WriteFile(tmpPath, stdout, 0600)
	if err != nil {
		return "", fmt.Errorf("Failed to create temporary file %s", tmpPath)
	}
	defer os.RemoveAll(tmpPath)
	helmOverrides = append(helmOverrides, helmcli.HelmOverrides{FileOverride: tmpPath})

	// Build scenario override absolute path for the use case, E.G manifests/scenarios/opensearch/s1/usecase-overrides/getlogs-fast.yaml
	scOverride := filepath.Join(scman.ScenarioUsecaseOverridesAbsDir, hr.OverrideFile)
	helmOverrides = append(helmOverrides, helmcli.HelmOverrides{FileOverride: scOverride})

	if m.Verbose {
		fmt.Printf("Updating use case %s for Helm release %s/%s\n", hr.Usecase.UsecasePath, hr.Namespace, hr.Name)
	}
	_, stderr, err := helmcli.Upgrade(m.Log, hr.Name, m.Namespace, m.Manifest.WorkerChartAbsDir, true, m.DryRun, helmOverrides)
	if err != nil {
		return string(stderr), err
	}
	return "", nil
}
