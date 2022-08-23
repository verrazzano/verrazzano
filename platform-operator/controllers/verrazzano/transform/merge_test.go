// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"sigs.k8s.io/yaml"

	"github.com/stretchr/testify/assert"
	vzyaml "github.com/verrazzano/verrazzano/pkg/yaml"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

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
			base:     "./testdata/dev.yaml",
			overlay:  "./testdata/managed.yaml",
			expected: "./testdata/managed.yaml",
		},
		{
			name:     "3",
			base:     "./testdata/cert_base.yaml",
			overlay:  "./testdata/cert_overlay.yaml",
			expected: "./testdata/cert_merged.yaml",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			merged, err := vzyaml.StrategicMergeFiles(vzapi.Verrazzano{}, test.base, test.overlay)
			assert.NoError(err, "error merging profiles")
			vzMerged := vzapi.Verrazzano{}
			err = yaml.Unmarshal([]byte(merged), &vzMerged)
			assert.NoError(err, "Error marshalling merged results into VZ struct")

			expected, err := ioutil.ReadFile(filepath.Join(test.expected))
			assert.NoError(err, "error reading mergedCR results file")
			vzExpected := vzapi.Verrazzano{}
			err = yaml.Unmarshal(expected, &vzExpected)
			assert.NoError(err, "Error marshalling mergedCR results into VZ struct")

			assert.Equal(vzExpected, vzMerged, "merged profile is incorrect ")
		})
	}
}

// TestMergeProfiles tests the MergeProfiles function for a list of profiles
// GIVEN an array of tests, where each tests specifies profiles to merge
// WHEN MergeProfiles is called, with some contents being a list that should be merged
// THEN ensure that the merged result is correct.
func TestMergeProfiles(t *testing.T) {
	tests := []struct {
		name     string
		actualCR string
		profiles []string
		mergedCR string
	}{
		{
			name:     "1",
			actualCR: "./testdata/dev.yaml",
			profiles: []string{
				"./testdata/managed.yaml",
			},
			mergedCR: "./testdata/managed.yaml",
		},
		{
			name:     "2",
			actualCR: "./testdata/managed.yaml",
			profiles: []string{
				"./testdata/console.yaml",
				"./testdata/keycloak.yaml",
			},
			mergedCR: "./testdata/managed_merged.yaml",
		},
		{
			name:     "3",
			actualCR: "./testdata/cert_base.yaml",
			profiles: []string{
				"./testdata/cert_overlay.yaml",
			},
			mergedCR: "./testdata/cert_base.yaml",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Create VerrazzanoSpec from actualCR profile
			actualCR, err := readProfile(test.actualCR)
			assert.NoError(err, "error reading profiles")

			// Merge the profiles
			mergedCR, err := MergeProfiles(actualCR, test.profiles...)
			assert.NoError(err, "error merging profiles")

			// Create VerrazzanoSpec from mergedCR profile
			expectedSpec, err := readProfile(test.mergedCR)
			assert.NoError(err, "error reading profiles")

			assert.Equal(expectedSpec, mergedCR, "merged profile is incorrect ")
		})
	}
}

// TestAppendComponentOverrides tests the appendComponentOverrides
// GIVEN  actual and profile CRs
// WHEN appendComponentOverrides is called
// THEN the compoent overrides from the profile CR should be appended to the component overrides of the actual CR.
func TestAppendComponentOverrides(t *testing.T) {
	actual, err := readProfile("./testdata/actual.yaml")
	assert.NoError(t, err)
	profile, err := readProfile("./testdata/profile.yaml")
	assert.NoError(t, err)
	appendComponentOverrides(actual, profile)
	merged, err := readProfile("./testdata/merged.yaml")
	assert.NoError(t, err)
	assert.Equal(t, merged, actual)
}

// TestMergeProfilesForV1beta1 tests the MergeProfiles function for a list of profiles
// GIVEN an array of tests, where each tests specifies profiles to merge
// WHEN MergeProfilesForV1beta1 is called, with some contents being a list that should be merged
// THEN ensure that the merged result is correct.
func TestMergeProfilesForV1beta1(t *testing.T) {
	tests := []struct {
		name     string
		actualCR string
		profiles []string
		mergedCR string
	}{
		{
			name:     "1",
			actualCR: "./testdata/dev.yaml",
			profiles: []string{
				"./testdata/managed.yaml",
			},
			mergedCR: "./testdata/managed.yaml",
		},
		{
			name:     "2",
			actualCR: "./testdata/managed.yaml",
			profiles: []string{
				"./testdata/console.yaml",
				"./testdata/keycloak.yaml",
			},
			mergedCR: "./testdata/managed_merged.yaml",
		},
		{
			name:     "3",
			actualCR: "./testdata/cert_base.yaml",
			profiles: []string{
				"./testdata/cert_overlay.yaml",
			},
			mergedCR: "./testdata/cert_base.yaml",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Create VerrazzanoSpec from actualCR profile
			actualCR, err := readProfileForV1beta1(test.actualCR)
			assert.NoError(err, "error reading profiles")

			// Merge the profiles
			mergedCR, err := MergeProfilesForV1beta1(actualCR, test.profiles...)
			assert.NoError(err, "error merging profiles")

			// Create VerrazzanoSpec from mergedCR profile
			expectedSpec, err := readProfileForV1beta1(test.mergedCR)
			assert.NoError(err, "error reading profiles")

			assert.Equal(expectedSpec, mergedCR, "merged profile is incorrect ")
		})
	}
}

// TestAppendComponentOverrides tests the appendComponentOverrides
// GIVEN  actual and profile CRs
// WHEN appendComponentOverrides is called
// THEN the compoent overrides from the profile CR should be appended to the component overrides of the actual CR.
func TestAppendComponentOverridesForV1beta1(t *testing.T) {
	actual, err := readProfileForV1beta1("./testdata/actual.yaml")
	assert.NoError(t, err)
	profile, err := readProfileForV1beta1("./testdata/profile.yaml")
	assert.NoError(t, err)
	appendComponentOverridesV1beta1(actual, profile)
	merged, err := readProfileForV1beta1("./testdata/merged.yaml")
	assert.NoError(t, err)
	assert.Equal(t, merged, actual)
}

// Create VerrazzanoSpec from profile
func readProfile(filename string) (*vzapi.Verrazzano, error) {
	specYaml, err := ioutil.ReadFile(filepath.Join(filename))
	if err != nil {
		return nil, err
	}
	var spec vzapi.Verrazzano
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
