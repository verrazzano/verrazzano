// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"sigs.k8s.io/yaml"
)

const (
	actualSpecFilePath         = "./testdata/actual.yaml"
	certsBaseSpecFilePath      = "./testdata/cert_base.yaml"
	certsOverlaySpecFilePath   = "./testdata/cert_overlay.yaml"
	certsMergedSpecFilePath    = "./testdata/cert_merged.yaml"
	consoleSpecFilePath        = "./testdata/console.yaml"
	devProfileSpecFilePath     = "./testdata/dev.yaml"
	keycloakSpecFilePath       = "./testdata/keycloak.yaml"
	managedProfileSpecFilePath = "./testdata/managed.yaml"
	managedMergedSpecFilePath  = "./testdata/managed_merged.yaml"
	mergedSpecFilePath         = "./testdata/merged.yaml"
	profileCustomSpecFilePath  = "./testdata/profile.yaml"
)

const (
	mergeProfileErrorMsg      = "error merging profiles"
	readProfileErrorMsg       = "error reading profiles"
	incorrectMergedProfileMsg = "merged profile is incorrect"
)

type mergeProfileTestData struct {
	name     string
	actualCR string
	profiles []string
	mergedCR string
}

// TestMergeSpec tests the StrategicMergeFiles function for a list of VerrazzanoSpecs
// GIVEN an array of tests, where each tests specifies files to merge
// WHEN StrategicMergeFiles is called, with some contents being a list that should be merged
// THEN ensure that the merged result is correct.
func TestMergeSpec(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		overlay  string
		expected string
	}{
		{
			name:     "1",
			base:     devProfileSpecFilePath,
			overlay:  managedProfileSpecFilePath,
			expected: managedProfileSpecFilePath,
		},
		{
			name:     "3",
			base:     certsBaseSpecFilePath,
			overlay:  certsOverlaySpecFilePath,
			expected: certsMergedSpecFilePath,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			merged, err := vzyaml.StrategicMergeFiles(v1alpha1.Verrazzano{}, test.base, test.overlay)
			assert.NoError(t, err, mergeProfileErrorMsg)
			vzMerged := v1alpha1.Verrazzano{}
			err = yaml.Unmarshal([]byte(merged), &vzMerged)
			assert.NoError(t, err, "Error marshalling merged results into VZ struct")

			expected, err := ioutil.ReadFile(filepath.Join(test.expected))
			assert.NoError(t, err, "error reading mergedCR results file")
			vzExpected := v1alpha1.Verrazzano{}
			err = yaml.Unmarshal(expected, &vzExpected)
			assert.NoError(t, err, "Error marshalling mergedCR results into VZ struct")

			assert.Equal(t, vzExpected, vzMerged, incorrectMergedProfileMsg)
		})
	}
}

// TestMergeProfiles tests the MergeProfiles function for a list of profiles
// GIVEN an array of tests, where each tests specifies profiles to merge
// WHEN MergeProfiles is called, with some contents being a list that should be merged
// THEN ensure that the merged result is correct.
func TestMergeProfiles(t *testing.T) {
	for _, test := range getMergeProfileTestData() {
		t.Run(test.name, func(t *testing.T) {

			// Create VerrazzanoSpec from actualCR profile
			actualCR, err := readProfile(test.actualCR)
			assert.NoError(t, err, readProfileErrorMsg)

			// Merge the profiles
			mergedCR, err := MergeProfiles(actualCR, test.profiles...)
			assert.NoError(t, err, mergeProfileErrorMsg)

			// Create VerrazzanoSpec from mergedCR profile
			expectedSpec, err := readProfile(test.mergedCR)
			assert.NoError(t, err, readProfileErrorMsg)

			assert.Equal(t, expectedSpec, mergedCR, incorrectMergedProfileMsg)
		})
	}
}

// TestAppendComponentOverrides tests the appendComponentOverrides
// GIVEN  actual and profile CRs
// WHEN appendComponentOverrides is called
// THEN the compoent overrides from the profile CR should be appended to the component overrides of the actual CR.
func TestAppendComponentOverrides(t *testing.T) {
	actual, err := readProfile(actualSpecFilePath)
	assert.NoError(t, err)
	profile, err := readProfile(profileCustomSpecFilePath)
	assert.NoError(t, err)
	appendComponentOverrides(actual, profile)
	merged, err := readProfile(mergedSpecFilePath)
	assert.NoError(t, err)
	assert.Equal(t, merged, actual)
}

// TestMergeProfilesForV1beta1 tests the MergeProfiles function for a list of profiles
// GIVEN an array of tests, where each tests specifies profiles to merge
// WHEN MergeProfilesForV1beta1 is called, with some contents being a list that should be merged
// THEN ensure that the merged result is correct.
func TestMergeProfilesForV1beta1(t *testing.T) {
	for _, test := range getMergeProfileTestData() {
		t.Run(test.name, func(t *testing.T) {

			// Create VerrazzanoSpec from actualCR profile
			actualCR, err := readProfileForV1beta1(test.actualCR)
			assert.NoError(t, err, readProfileErrorMsg)

			// Merge the profiles
			mergedCR, err := MergeProfilesForV1beta1(actualCR, test.profiles...)
			assert.NoError(t, err, mergeProfileErrorMsg)

			// Create VerrazzanoSpec from mergedCR profile
			expectedSpec, err := readProfileForV1beta1(test.mergedCR)
			assert.NoError(t, err, readProfileErrorMsg)

			assert.Equal(t, expectedSpec, mergedCR, incorrectMergedProfileMsg)
		})
	}
}

// TestAppendComponentOverrides tests the appendComponentOverrides
// GIVEN  actual and profile CRs
// WHEN appendComponentOverrides is called
// THEN the compoent overrides from the profile CR should be appended to the component overrides of the actual CR.
func TestAppendComponentOverridesForV1beta1(t *testing.T) {
	actual, err := readProfileForV1beta1(actualSpecFilePath)
	assert.NoError(t, err)
	profile, err := readProfileForV1beta1(profileCustomSpecFilePath)
	assert.NoError(t, err)
	appendComponentOverridesV1beta1(actual, profile)
	merged, err := readProfileForV1beta1(mergedSpecFilePath)
	assert.NoError(t, err)
	assert.Equal(t, merged, actual)
}

// Create VerrazzanoSpec from profile
func readProfile(filename string) (*v1alpha1.Verrazzano, error) {
	specYaml, err := ioutil.ReadFile(filepath.Join(filename))
	if err != nil {
		return nil, err
	}
	var spec v1alpha1.Verrazzano
	err = yaml.Unmarshal(specYaml, &spec)
	if err != nil {
		return nil, err
	}
	return &spec, nil
}

// Create VerrazzanoSpec from profile
func readProfileForV1beta1(filename string) (*v1beta1.Verrazzano, error) {
	specYaml, err := ioutil.ReadFile(filepath.Join(filename))
	if err != nil {
		return nil, err
	}
	var spec v1beta1.Verrazzano
	err = yaml.Unmarshal(specYaml, &spec)
	if err != nil {
		return nil, err
	}
	return &spec, nil
}

func getMergeProfileTestData() []mergeProfileTestData {
	return []mergeProfileTestData{
		{
			name:     "1",
			actualCR: devProfileSpecFilePath,
			profiles: []string{
				managedProfileSpecFilePath,
			},
			mergedCR: managedProfileSpecFilePath,
		},
		{
			name:     "2",
			actualCR: managedProfileSpecFilePath,
			profiles: []string{
				consoleSpecFilePath,
				keycloakSpecFilePath,
			},
			mergedCR: managedMergedSpecFilePath,
		},
		{
			name:     "3",
			actualCR: certsBaseSpecFilePath,
			profiles: []string{
				certsOverlaySpecFilePath,
			},
			mergedCR: certsBaseSpecFilePath,
		},
	}
}
