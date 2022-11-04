// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package scenario

const (
	// LabelScenario has a bool value to indicate a resource is part of a scenario
	LabelScenario = "psr.verrazzano.io/scenario"

	// LabelScenarioId has a string value with the scenario ID
	LabelScenarioId = "psr.verrazzano.io/scenario-id"

	// DataScenarioKey is the configmap key for the data field which
	// has a base64 encoded Scenario
	DataScenarioKey = "scenario"
)
