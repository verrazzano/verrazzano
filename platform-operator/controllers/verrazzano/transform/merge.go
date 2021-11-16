// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"sigs.k8s.io/yaml"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzyaml "github.com/verrazzano/verrazzano/platform-operator/internal/yaml"
)

// MergeProfiles merges a list of Verrazzano profile files with an existing Verrazzano CR.
// The profiles must be in the Verrazzano CR format
func MergeProfiles(cr *vzapi.Verrazzano, profileFiles ...string) (*vzapi.Verrazzano, error) {
	// First merge the profiles
	merged, err := vzyaml.StrategicMergeFiles(vzapi.Verrazzano{}, profileFiles...)
	if err != nil {
		return nil, err
	}

	// Now merge the the profiles on top of the Verrazzano CR
	crYAML, err := yaml.Marshal(cr)
	if err != nil {
		return nil, err
	}

	merged, err = vzyaml.StrategicMerge(vzapi.Verrazzano{}, merged, string(crYAML))
	if err != nil {
		return nil, err
	}

	// Return a new CR
	var newCR vzapi.Verrazzano
	yaml.Unmarshal([]byte(merged), &newCR)

	return &newCR, nil
}
