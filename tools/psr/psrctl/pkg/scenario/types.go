// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

import (
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/pkg/manifest"
	"k8s.io/apimachinery/pkg/types"
)

// Scenario specifies a PSR scenario that was installed in the cluster
type Scenario struct {
	// The namespace where the scenario is installed
	Namespace string

	// The helm releases that are installed by the scenario
	HelmReleases []HelmRelease

	// The scenario manifests that was used to run the scenario
	*manifest.ScenarioManifest
}

// HelmRelease specifies a HelmRelease for a use case within a scenario
type HelmRelease struct {
	// The namespaced name of the Helm release
	types.NamespacedName

	// The scenario use case for this HelmRelase
	manifest.Usecase
}
