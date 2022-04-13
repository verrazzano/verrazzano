// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"github.com/verrazzano/verrazzano/pkg/constants"
	"strings"

	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"sigs.k8s.io/yaml"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzyaml "github.com/verrazzano/verrazzano/platform-operator/internal/yaml"
)

const (
	// implicit base profile (defaults)
	baseProfile = "base"
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

// GetEffectiveCR Creates an "effective" Verrazzano CR based on the user defined resource merged with the profile definitions
// - Effective CR == base profile + declared profiles + ActualCR (in order)
// - last definition wins
func GetEffectiveCR(actualCR *vzapi.Verrazzano) (*vzapi.Verrazzano, error) {
	if actualCR == nil {
		return nil, nil
	}
	// Identify the set of profiles, base + declared
	profiles := []string{baseProfile, string(vzapi.Prod)}
	if len(actualCR.Spec.Profile) > 0 {
		profiles = append([]string{baseProfile}, strings.Split(string(actualCR.Spec.Profile), ",")...)
	}
	var profileFiles []string
	for _, profile := range profiles {
		profileFiles = append(profileFiles, config.GetProfile(profile))
	}
	// Merge the profile files into an effective profile YAML string
	effectiveCR, err := MergeProfiles(actualCR, profileFiles...)
	if err != nil {
		return nil, err
	}
	effectiveCR.Status = vzapi.VerrazzanoStatus{} // Don't replicate the CR status in the effective config
	// if Certificate in CertManager is empty, set it to default CA
	var emptyCertConfig = vzapi.Certificate{}
	if effectiveCR.Spec.Components.CertManager.Certificate == emptyCertConfig {
		effectiveCR.Spec.Components.CertManager.Certificate.CA = vzapi.CA{
			SecretName:               constants.DefaultVerrazzanoCASecretName,
			ClusterResourceNamespace: constants.CertManagerNamespace,
		}
	}
	return effectiveCR, nil
}
