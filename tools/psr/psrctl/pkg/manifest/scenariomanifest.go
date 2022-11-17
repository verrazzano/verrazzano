// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package manifest

// Usecase specifies a PSR usecase that does a single worker task running in a pod
type Usecase struct {
	// UsecasePath specifies the manifest relative path of the use case, e.g. opensearch/writelogs.yaml
	UsecasePath string

	// OverrideFile is the use case override file in the scenario usecase-overrides directory, e.g. writelogs-fast.yaml
	OverrideFile string

	// Description is a description of the use case in the context of the scenario
	Description string
}

// ScenarioManifest specifies a PSR scenario manifest which consists of multiple use cases.
// The manifest represents files on disk, not a runtime scenario.
type ScenarioManifest struct {
	// Name is the scenario name
	Name string

	// ID is the scenario ID
	ID string

	// Description is the scenario description
	Description string

	// Usecases are the scenario use cases
	Usecases []Usecase

	// This is the absolute directory that contains scenario.yaml and scenario usecase-overrides.  It is not specified by the user,
	// but built at runtime
	ScenarioUsecaseOverridesAbsDir string

	// ManifestManager is the manifest manager
	*ManifestManager
}
