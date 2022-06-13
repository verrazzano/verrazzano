// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
)

// TestMergeYAMLFilesSingle
// GIVEN a single YAML file
//  WHEN I call MergeYAMLFiles
//  THEN a vz resource is returned representing the single YAML file
func TestMergeYAMLFilesSingle(t *testing.T) {
	vz, err := MergeYAMLFiles([]string{"../../test/testdata/dev-profile.yaml"})
	assert.Nil(t, err)
	assert.Equal(t, "my-verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, vzapi.Dev, vz.Spec.Profile)
}

// TestMergeYAMLFilesComponents
// GIVEN a base yaml file and components yaml file
//  WHEN I call MergeYAMLFiles
//  THEN a vz resource is returned representing the merged YAML files
func TestMergeYAMLFilesComponents(t *testing.T) {
	vz, err := MergeYAMLFiles([]string{
		"../../test/testdata/dev-profile.yaml",
		"../../test/testdata/components.yaml",
	})
	assert.Nil(t, err)
	assert.Equal(t, "my-verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, vzapi.Dev, vz.Spec.Profile)
	assert.Equal(t, false, *vz.Spec.Components.Console.Enabled)
	assert.Equal(t, false, *vz.Spec.Components.Fluentd.Enabled)
	assert.Equal(t, true, *vz.Spec.Components.Rancher.Enabled)
	assert.Nil(t, vz.Spec.Components.Verrazzano)
}

// TestMergeYAMLFilesOverrideComponents
// GIVEN a component yaml file and components override yaml file
//  WHEN I call MergeYAMLFiles
//  THEN a vz resource is returned representing the merged YAML files
func TestMergeYAMLFilesOverrideComponents(t *testing.T) {
	vz, err := MergeYAMLFiles([]string{
		"../../test/testdata/components.yaml",
		"../../test/testdata/override-components.yaml",
	})
	assert.Nil(t, err)
	assert.Equal(t, "verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, true, *vz.Spec.Components.Console.Enabled)
	assert.Equal(t, true, *vz.Spec.Components.Fluentd.Enabled)
	assert.Equal(t, false, *vz.Spec.Components.Rancher.Enabled)
	assert.Nil(t, vz.Spec.Components.Verrazzano)
}

// TestMergeYAMLFilesEmpty
// GIVEN a base yaml file and a empty yaml file
//  WHEN I call MergeYAMLFiles
//  THEN a vz resource is returned representing the base yaml file
func TestMergeYAMLFilesEmpty(t *testing.T) {
	vz, err := MergeYAMLFiles([]string{
		"../../test/testdata/dev-profile.yaml",
		"../../test/testdata/empty.yaml",
	})
	assert.Nil(t, err)
	assert.Equal(t, "my-verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, vzapi.Dev, vz.Spec.Profile)
}

// TestMergeYAMLFilesNone
// GIVEN no YAML files
//  WHEN I call MergeYAMLFiles
//  THEN the call returns an error
func TestMergeYAMLFilesNone(t *testing.T) {
	_, err := MergeYAMLFiles([]string{})
	assert.Error(t, err)
	assert.EqualError(t, err, "Failed to merge files - no files specified")
}

// TestMergeYAMLFilesNotFound
// GIVEN a YAML file that does not exist
//  WHEN I call MergeYAMLFiles
//  THEN the call returns an error
func TestMergeYAMLFilesNotFound(t *testing.T) {
	_, err := MergeYAMLFiles([]string{"../../test/testdate/file-does-not-exist.yaml"})
	assert.Error(t, err)
	assert.EqualError(t, err, "open ../../test/testdate/file-does-not-exist.yaml: no such file or directory")
}
