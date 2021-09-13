// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cr

import (
	"sigs.k8s.io/yaml"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzyaml "github.com/verrazzano/verrazzano/platform-operator/internal/yaml"
)

// MergeProfiles merges a list of Verrazzano profile files with an existing VerrazzanpSpec.
// The profiles must be in the VerrazzanoSpec format
func MergeProfiles(cr *vzapi.VerrazzanoSpec, profileFiles ...string) (*vzapi.VerrazzanoSpec, error) {
	// First merge the profiles
	merged, err := vzyaml.MergeFiles(vzapi.VerrazzanoSpec{}, profileFiles...)
	if err != nil {
		return nil, err
	}

	// Now merge the the profiles on top of the Verrazzano CR
	bYAML, err := yaml.Marshal(cr)
	if err != nil {
		return nil, err
	}

	merged, err = vzyaml.MergeString(vzapi.VerrazzanoSpec{}, string(bYAML), merged)
	if err != nil {
		return nil, err
	}

	// Return a new CR
	var newCR vzapi.VerrazzanoSpec
	yaml.Unmarshal([]byte(merged), &newCR)

	return &newCR, nil
}
