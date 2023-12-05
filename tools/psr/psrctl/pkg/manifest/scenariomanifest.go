// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package manifest

// WorkerConfig specifies a PSR worker config used by a single worker task running in a pod
type WorkerConfig struct {
	// WorkerConfigPath specifies the manifest relative path of the worker config, e.g. opensearch/writelogs.yaml
	WorkerConfigPath string

	// WorkerOverrideFile is the use case override file in the scenario worker-overrides directory, e.g. writelogs-fast.yaml
	WorkerOverrideFile string

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

	// WorkerConfigs are the scenario worker configurations
	WorkerConfigs []WorkerConfig `json:"workers"`

	// This is the absolute directory that contains scenario.yaml and scenario worker config overrides.
	// It is not specified by the user, but built at runtime
	ScenarioWorkerConfigOverridesAbsDir string
}
