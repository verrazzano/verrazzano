// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package transform

import (
	"io/ioutil"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"testing"

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
