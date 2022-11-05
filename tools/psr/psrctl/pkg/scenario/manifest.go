// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"github.com/verrazzano/verrazzano/pkg/files"
	"os"
	"path/filepath"
	"regexp"
	"sigs.k8s.io/yaml"
	"strings"
)

// The required use case overrides directory
const usecaseOverrideDir = "usecase-overrides"

// ListScenarioManifests returns the list of ScenarioManifests. Scenario manifests
// are located in psr/manifests/scenarios.  By convention, a scenario directory must have
// a scenario.yaml file which describes the scenario. It must also have
// a subdirectory named usecase-overrides containing the override parameters for
// each use case. The name of the parent directory, for example s1, is irrelevant.
func (m Manager) ListScenarioManifests() ([]ScenarioManifest, error) {
	scenarios := []ScenarioManifest{}
	sfiles, err := files.GetMatchingFiles(m.Manifest.ScenarioAbsDir, regexp.MustCompile("scenario.yaml"))
	if err != nil {
		return nil, err
	}
	for _, f := range sfiles {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var sc ScenarioManifest
		if err := yaml.Unmarshal(data, &sc); err != nil {
			return nil, m.Log.ErrorfNewErr("Failed to unmarshal ScenarioManifest from file %s: %v", f, err)
		}

		// Build the parent directory name that has the scenario.yaml.
		sc.ScenarioUsecaseOverridesDir = filepath.Join(filepath.Dir(f), usecaseOverrideDir)
		scenarios = append(scenarios, sc)
	}
	return scenarios, nil
}

// FindScenarioManifestByID finds a ScenarioManifest by ID
func (m Manager) FindScenarioManifestByID(ID string) (*ScenarioManifest, error) {
	return m.findScenarioManifest(func(scenario ScenarioManifest) bool {
		return strings.EqualFold(scenario.ID, ID)
	})
}

// FindScenarioManifestByName finds a ScenarioManifest by mame
func (m Manager) FindScenarioManifestByName(name string) (*ScenarioManifest, error) {
	return m.findScenarioManifest(func(scenario ScenarioManifest) bool {
		return strings.EqualFold(scenario.Name, name)
	})
}

// findScenarioManifest finds a ScenarioManifest
func (m Manager) findScenarioManifest(f func(ScenarioManifest) bool) (*ScenarioManifest, error) {
	scList, err := m.ListScenarioManifests()
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
