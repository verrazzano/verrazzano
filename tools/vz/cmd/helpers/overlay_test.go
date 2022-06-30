// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestMergeYAMLFilesSingle
// GIVEN a single YAML file
//  WHEN I call MergeYAMLFiles
//  THEN a vz resource is returned representing the single YAML file
func TestMergeYAMLFilesSingle(t *testing.T) {
	vz, err := MergeYAMLFiles([]string{"../../test/testdata/dev-profile.yaml"}, os.Stdin)
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
	}, os.Stdin)
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
	}, os.Stdin)
	assert.Nil(t, err)
	assert.Equal(t, "verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, true, *vz.Spec.Components.Console.Enabled)
	assert.Equal(t, true, *vz.Spec.Components.Fluentd.Enabled)
	assert.Equal(t, false, *vz.Spec.Components.Rancher.Enabled)
	assert.Nil(t, vz.Spec.Components.Verrazzano)
}

// TestMergeYAMLFilesStdin
// GIVEN a yaml file from stdin
//  WHEN I call MergeYAMLFiles
//  THEN a vz resource is returned representing the yaml specified via stdin
func TestMergeYAMLFilesStdin(t *testing.T) {
	var filenames []string
	stdinReader := &bytes.Buffer{}
	b, err := os.ReadFile("../../test/testdata/quick-start.yaml")
	assert.Nil(t, err)
	_, err = stdinReader.Write(b)
	assert.Nil(t, err)
	filenames = append(filenames, "-")
	vz, err := MergeYAMLFiles(filenames, stdinReader)
	assert.Nil(t, err)
	assert.Equal(t, "example-verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, vzapi.Dev, vz.Spec.Profile)
	assert.Equal(t, "verrazzano-storage", vz.Spec.DefaultVolumeSource.PersistentVolumeClaim.ClaimName)
	assert.Equal(t, "verrazzano-storage", vz.Spec.VolumeClaimSpecTemplates[0].Name)
	storage := vz.Spec.VolumeClaimSpecTemplates[0].Spec.Resources.Requests.Storage()
	assert.Contains(t, storage.String(), "2Gi")
}

// TestMergeYAMLFilesStdinOverride
// GIVEN a yaml file from a file and a yaml file from stdin
//  WHEN I call MergeYAMLFiles
//  THEN a vz resource is returned representing the merged YAML files
func TestMergeYAMLFilesStdinOverride(t *testing.T) {
	var filenames []string
	filenames = append(filenames, "../../test/testdata/components.yaml")
	stdinReader := &bytes.Buffer{}
	b, err := os.ReadFile("../../test/testdata/override-components.yaml")
	assert.Nil(t, err)
	_, err = stdinReader.Write(b)
	assert.Nil(t, err)
	filenames = append(filenames, "-")
	vz, err := MergeYAMLFiles(filenames, stdinReader)
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
	}, os.Stdin)
	assert.Nil(t, err)
	assert.Equal(t, "my-verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, vzapi.Dev, vz.Spec.Profile)
}

// TestMergeYAMLFilesNotFound
// GIVEN a YAML file that does not exist
//  WHEN I call MergeYAMLFiles
//  THEN the call returns an error
func TestMergeYAMLFilesNotFound(t *testing.T) {
	_, err := MergeYAMLFiles([]string{"../../test/testdate/file-does-not-exist.yaml"}, os.Stdin)
	assert.Error(t, err)
	assert.EqualError(t, err, "open ../../test/testdate/file-does-not-exist.yaml: no such file or directory")
}

// TestMergeSetFlags
// GIVEN a YAML file and a YAML string
// WHEN I call MergeSetFlags
// THEN the call returns a vz resource with the two source merged
func TestMergeSetFlags(t *testing.T) {
	yamlString := "spec:\n  environmentName: test"
	vz := &vzapi.Verrazzano{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "verrazzano",
		},
	}
	c := fake.NewClientBuilder().WithScheme(helpers.NewScheme()).WithObjects(vz).Build()

	_, err := MergeSetFlags(vz, yamlString)
	assert.NoError(t, err)

	// Verify the vz resource is as expected
	mergedvz := vzapi.Verrazzano{}
	err = c.Get(context.TODO(), types.NamespacedName{Namespace: "default", Name: "verrazzano"}, &mergedvz)
	assert.NoError(t, err)
	assert.Equal(t, "test", vz.Spec.EnvironmentName)
}
