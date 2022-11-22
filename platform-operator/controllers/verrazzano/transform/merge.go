// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	string2 "github.com/verrazzano/verrazzano/pkg/string"
	"strings"

	"github.com/verrazzano/verrazzano/pkg/constants"
	vzprofiles "github.com/verrazzano/verrazzano/pkg/profiles"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const (
	// implicit base profile (defaults)
	baseProfile = "base"
)

// GetEffectiveCR Creates an "effective" v1alpha1.Verrazzano CR based on the user defined resource merged with the profile definitions
// - Effective CR == base profile + declared profiles + ActualCR (in order)
// - last definition wins
func GetEffectiveCR(actualCR *v1alpha1.Verrazzano) (*v1alpha1.Verrazzano, error) {
	if actualCR == nil {
		return nil, nil
	}
	// Identify the set of profiles, base + declared
	profiles := buildProfilesList(string(actualCR.Spec.Profile), string(v1alpha1.Prod), string(v1alpha1.None))
	var profileFiles []string
	for _, profile := range profiles {
		profileFiles = append(profileFiles, config.GetProfile(v1alpha1.SchemeGroupVersion, profile))
	}
	// Merge the profile files into an effective profile YAML string
	effectiveCR, err := vzprofiles.MergeProfiles(actualCR, profileFiles...)
	if err != nil {
		return nil, err
	}
	effectiveCR.Status = v1alpha1.VerrazzanoStatus{} // Don't replicate the CR status in the effective config
	// if Certificate in CertManager is empty, set it to default CA
	var emptyCertConfig = v1alpha1.Certificate{}
	if effectiveCR.Spec.Components.CertManager.Certificate == emptyCertConfig {
		effectiveCR.Spec.Components.CertManager.Certificate.CA = v1alpha1.CA{
			SecretName:               constants.DefaultVerrazzanoCASecretName,
			ClusterResourceNamespace: constants.CertManagerNamespace,
		}
	}
	return effectiveCR, nil
}

// GetEffectiveV1beta1CR Creates an "effective" v1beta1.Verrazzano CR based on the user defined resource merged with the profile definitions
// - Effective CR == base profile + declared profiles + ActualCR (in order)
// - last definition wins
func GetEffectiveV1beta1CR(actualCR *v1beta1.Verrazzano) (*v1beta1.Verrazzano, error) {
	if actualCR == nil {
		return nil, nil
	}
	profiles := buildProfilesList(string(actualCR.Spec.Profile), string(v1beta1.Prod), string(v1beta1.None))
	var profileFiles []string
	for _, profile := range profiles {
		profileFiles = append(profileFiles, config.GetProfile(v1beta1.SchemeGroupVersion, profile))
	}
	// Merge the profile files into an effective profile YAML string
	effectiveCR, err := vzprofiles.MergeProfilesForV1beta1(actualCR, profileFiles...)
	if err != nil {
		return nil, err
	}
	effectiveCR.Status = v1beta1.VerrazzanoStatus{} // Don't replicate the CR status in the effective config
	// if Certificate in CertManager is empty, set it to default CA
	var emptyCertConfig = v1beta1.Certificate{}
	if effectiveCR.Spec.Components.CertManager.Certificate == emptyCertConfig {
		effectiveCR.Spec.Components.CertManager.Certificate.CA = v1beta1.CA{
			SecretName:               constants.DefaultVerrazzanoCASecretName,
			ClusterResourceNamespace: constants.CertManagerNamespace,
		}
	}
	return effectiveCR, nil
}

func buildProfilesList(profilesVal string, defaultVal string, noneVal string) []string {
	if len(profilesVal) == 0 {
		return []string{baseProfile, defaultVal}
	}
	if profilesVal == noneVal {
		return []string{noneVal}
	}
	profiles := []string{}
	explicitProfilesList := strings.Split(profilesVal, ",")
	for _, explicitProfile := range explicitProfilesList {
		if explicitProfile != noneVal && !string2.SliceContainsString(explicitProfilesList, baseProfile) {
			// if the profile is not None and we haven't already included the baseProfile, include it
			profiles = append(profiles, baseProfile)
		}
		// append the explicitly declared profile to the list
		profiles = append(profiles, explicitProfile)
	}
	return profiles
}
