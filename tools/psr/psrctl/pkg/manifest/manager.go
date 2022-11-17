package manifest

import (
	"github.com/verrazzano/verrazzano/pkg/helm"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"

	"github.com/verrazzano/verrazzano/pkg/files"
	"os"
	"path/filepath"
	"regexp"
	"sigs.k8s.io/yaml"
	"strings"
)

// The required use case overrides directory
const usecaseOverrideDir = "usecase-overrides"

// ManifestManager contains the information needed to manage a Scenario
type ManifestManager struct {
	Log                 vzlog.VerrazzanoLogger
	Manifest            PsrManifests
	ExternalScenarioDir string
}

// NewManager returns a manifest ManifestManager
func NewManager(externalScenarioDir string, helmOverrides ...helm.HelmOverrides) (ManifestManager, error) {
	m := ManifestManager{
		Log:                 vzlog.DefaultLogger(),
		Manifest:            *Manifests,
		ExternalScenarioDir: externalScenarioDir,
	}
	return m, nil
}

// ListScenarioManifests returns the list of ScenarioManifests. Scenario manifests
// are located in psr/manifests/scenarios.  By convention, a scenario directory must have
// a scenario.yaml file which describes the scenario. It must also have
// a subdirectory named usecase-overrides containing the override parameters for
// each use case. The name of the parent directory, for example s1, is irrelevant.
func (m ManifestManager) ListScenarioManifests() ([]ScenarioManifest, error) {
	scenarios := []ScenarioManifest{}

	// Default to the scenarios built into the image. However, the user can
	// override this dir for some operations, like start
	scenariosDir := m.Manifest.ScenarioAbsDir
	if len(m.ExternalScenarioDir) > 0 {
		scenariosDir = m.ExternalScenarioDir
	}

	// Find all the directories that contain scenario.yaml file
	sfiles, err := files.GetMatchingFiles(scenariosDir, regexp.MustCompile("scenario.yaml"))
	if err != nil {
		return nil, err
	}
	for _, f := range sfiles {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		var sman ScenarioManifest
		if err := yaml.Unmarshal(data, &sman); err != nil {
			return nil, m.Log.ErrorfNewErr("Failed to unmarshal ScenarioManifest from file %s: %v", f, err)
		}

		// Build the parent directory name that has the scenario.yaml.
		sman.ScenarioUsecaseOverridesAbsDir = filepath.Join(filepath.Dir(f), usecaseOverrideDir)
		
		sman.ManifestManager = &m
		scenarios = append(scenarios, sman)
	}
	return scenarios, nil
}

// FindScenarioManifestByID finds a ScenarioManifest by ID
func (m ManifestManager) FindScenarioManifestByID(ID string) (*ScenarioManifest, error) {
	return m.findScenarioManifest(func(scenario ScenarioManifest) bool {
		return strings.EqualFold(scenario.ID, ID)
	})
}

// FindScenarioManifestByName finds a ScenarioManifest by mame
func (m ManifestManager) FindScenarioManifestByName(name string) (*ScenarioManifest, error) {
	return m.findScenarioManifest(func(scenario ScenarioManifest) bool {
		return strings.EqualFold(scenario.Name, name)
	})
}

// findScenarioManifest finds a ScenarioManifest
func (m ManifestManager) findScenarioManifest(f func(ScenarioManifest) bool) (*ScenarioManifest, error) {
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
