// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"strings"

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
func GetEffectiveCR(actualCR *v1alpha1.Verrazzano, validate bool) (*v1alpha1.Verrazzano, error) {
	if actualCR == nil {
		return nil, nil
	}
	// Identify the set of profiles, base + declared
	profiles := []string{baseProfile, string(v1alpha1.Prod)}
	if len(actualCR.Spec.Profile) > 0 {
		profiles = append([]string{baseProfile}, strings.Split(string(actualCR.Spec.Profile), ",")...)
	}
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

	// Align the ClusterIssuer configurations between CertManager and ClusterIssuer components
	if err := convertCertificateToClusterIssuerV1Alpha1(effectiveCR); err != nil && validate {
		return nil, err
	}

	return effectiveCR, nil
}

// GetEffectiveV1beta1CR Creates an "effective" v1beta1.Verrazzano CR based on the user defined resource merged with the profile definitions
// - Effective CR == base profile + declared profiles + ActualCR (in order)
// - last definition wins
func GetEffectiveV1beta1CR(actualCR *v1beta1.Verrazzano, validate bool) (*v1beta1.Verrazzano, error) {
	if actualCR == nil {
		return nil, nil
	}
	// Identify the set of profiles, base + declared
	profiles := []string{baseProfile, string(v1beta1.Prod)}
	if len(actualCR.Spec.Profile) > 0 {
		profiles = append([]string{baseProfile}, strings.Split(string(actualCR.Spec.Profile), ",")...)
	}
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

	// Align the ClusterIssuer configurations between CertManager and ClusterIssuer components
	if err := convertCertificateToClusterIssuerV1Beta1(effectiveCR); err != nil && validate {
		return nil, err
	}

	return effectiveCR, nil
}
