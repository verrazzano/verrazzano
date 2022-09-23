// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bytes"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"sigs.k8s.io/yaml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

// TestMergeYAMLFilesSingle
// GIVEN a single YAML file
//  WHEN I call MergeYAMLFiles
//  THEN a vz resource is returned representing the single YAML file
func TestMergeYAMLFilesSingle(t *testing.T) {
	obj, err := MergeYAMLFiles([]string{"../../test/testdata/dev-profile.yaml"}, os.Stdin)
	assert.Nil(t, err)
	vz, err := toV1Alpha1VZ(obj)
	assert.Nil(t, err)
	assert.Equal(t, "my-verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, v1alpha1.Dev, vz.Spec.Profile)
}

// TestMergeYAMLFilesComponents
// GIVEN a base yaml file and components yaml file
//  WHEN I call MergeYAMLFiles
//  THEN a vz resource is returned representing the merged YAML files
func TestMergeYAMLFilesComponents(t *testing.T) {
	obj, err := MergeYAMLFiles([]string{
		"../../test/testdata/dev-profile.yaml",
		"../../test/testdata/components.yaml",
	}, os.Stdin)
	assert.Nil(t, err)
	vz, err := toV1Alpha1VZ(obj)
	assert.Nil(t, err)
	assert.Equal(t, "my-verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, v1alpha1.Dev, vz.Spec.Profile)
	assert.Equal(t, false, *vz.Spec.Components.Console.Enabled)
	assert.Equal(t, false, *vz.Spec.Components.Fluentd.Enabled)
	assert.Equal(t, true, *vz.Spec.Components.Rancher.Enabled)
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
	obj, err := MergeYAMLFiles(filenames, stdinReader)
	assert.Nil(t, err)
	vz, err := toV1Alpha1VZ(obj)
	assert.Nil(t, err)
	assert.Equal(t, "example-verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, v1alpha1.Dev, vz.Spec.Profile)
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
	filenames = append(filenames, "../../test/testdata/dev-profile.yaml", "../../test/testdata/components.yaml")
	stdinReader := &bytes.Buffer{}
	b, err := os.ReadFile("../../test/testdata/override-components.yaml")
	assert.Nil(t, err)
	_, err = stdinReader.Write(b)
	assert.Nil(t, err)
	filenames = append(filenames, "-")
	obj, err := MergeYAMLFiles(filenames, stdinReader)
	assert.Nil(t, err)
	vz, err := toV1Alpha1VZ(obj)
	assert.Nil(t, err)
	assert.Equal(t, "my-verrazzano", vz.Name)
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
	obj, err := MergeYAMLFiles([]string{
		"../../test/testdata/dev-profile.yaml",
		"../../test/testdata/empty.yaml",
	}, os.Stdin)
	assert.Nil(t, err)
	vz, err := toV1Alpha1VZ(obj)
	assert.Nil(t, err)
	assert.Equal(t, "my-verrazzano", vz.Name)
	assert.Equal(t, "default", vz.Namespace)
	assert.Equal(t, v1alpha1.Dev, vz.Spec.Profile)
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
	_, vz, err := helpers.NewVerrazzanoForVZVersion("1.4.0")
	assert.NoError(t, err)
	obj, err := MergeSetFlags(v1beta1.SchemeGroupVersion, vz, yamlString)
	assert.NoError(t, err)
	assert.Equal(t, "test", obj.(*unstructured.Unstructured).Object["spec"].(map[string]interface{})["environmentName"])
}

func toV1Alpha1VZ(u *unstructured.Unstructured) (*v1alpha1.Verrazzano, error) {
	dat, err := yaml.Marshal(u)
	if err != nil {
		return nil, err
	}
	vz := &v1alpha1.Verrazzano{}
	err = yaml.Unmarshal(dat, vz)
	return vz, err
}
