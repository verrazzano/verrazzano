// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import "k8s.io/apimachinery/pkg/types"

// Usecase specifies a PSR usecase that does a single worker task running in a pod
type Usecase struct {
	UsecasePath  string
	OverrideFile string
	Description  string
}

// ScenarioManifest specifies a PSR scenario manifest which consists of multiple use cases
// The manifest represents files on disk, not a runtime scenario.
type ScenarioManifest struct {
	Name        string
	ID          string
	Description string
	Usecases    []Usecase

	// This is  the directory that contains scenario usecase-overrides.  It is not specified by the user,
	// but built at runtime
	ScenarioUsecaseOverridesDir string
}

// Scenario specifies a PSR scenario that was installed in the cluster
type Scenario struct {
	// The namespaced names of the helm releases that comprise the scenario
	HelmReleases []types.NamespacedName

	// The scenario manifests that was used to run the scenario
	*ScenarioManifest
}
