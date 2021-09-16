// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cr

import (
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/equality"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzyaml "github.com/verrazzano/verrazzano/platform-operator/internal/yaml"
)

// TestMergeSpec tests the MergeFiles function for a list of VerrazzanoSpecs
// GIVEN an array of tests, where each tests specifies files to merge
// WHEN MergeFiles is called, with some contents being a list that should be merged
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
			merged, err := vzyaml.MergeFiles(vzapi.VerrazzanoSpec{}, test.base, test.overlay)
			assert.NoError(err, merged, "error merging profiles")
			expected, err := ioutil.ReadFile(filepath.Join(test.expected))
			assert.NoError(err, merged, "error reading profiles")
			assert.YAMLEq(merged, string(expected), "merged profile is incorrect ")
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
		base     string
		profiles []string
		expected string
	}{
		{
			name: "1",
			base: "./testdata/dev.yaml",
			profiles: []string{
				"./testdata/managed.yaml",
			},
			expected: "./testdata/managed.yaml",
		},
		{
			name: "2",
			base: "./testdata/managed.yaml",
			profiles: []string{
				"./testdata/console.yaml",
				"./testdata/keycloak.yaml",
			},
			expected: "./testdata/managed_merged.yaml",
		},
		{
			name: "3",
			base: "./testdata/cert_base.yaml",
			profiles: []string{
				"./testdata/cert_overlay.yaml",
			},
			expected: "./testdata/cert_merged.yaml",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			// Create VerrazzanoSpec from base profile
			spec, err := readProfile(test.base)
			assert.NoError(err, "error reading profiles")

			// Merge the profiles
			mergedSpec, err := MergeProfiles(spec, test.profiles...)
			assert.NoError(err, "error merging profiles")

			// Create VerrazzanoSpec from expected profile
			expectedSpec, err := readProfile(test.expected)
			assert.NoError(err, "error reading profiles")

			assert.True(equality.Semantic.DeepEqual(mergedSpec, expectedSpec), "merged profile is incorrect ")
		})
	}
}

// Create VerrazzanoSpec from profile
func readProfile(filename string) (*vzapi.VerrazzanoSpec, error) {
	specYaml, err := ioutil.ReadFile(filepath.Join(filename))
	if err != nil {
		return nil, err
	}
	var spec vzapi.VerrazzanoSpec
	err = yaml.Unmarshal([]byte(specYaml), &spec)
	if err != nil {
		return nil, err
	}
	return &spec, nil
}
