// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"sigs.k8s.io/yaml"
)

var vzDir = "../.."

// TestRun tests the following scenario
// GIVEN a call to run
// WHEN all the specified args and env vars are set
// THEN a generated profile file is found in the output directory
func TestRun(t *testing.T) {
	assert := assert.New(t)
	os.Setenv(VzRootDir, vzDir)
	dir := t.TempDir()
	err := run("prod", dir)
	assert.NoError(err)
	_, err = os.Stat(dir + "/" + "prod.yaml")
	assert.NoError(err)
}

// TestVerrazzanoRootNotSpecified tests the following scenario
// GIVEN a call to run func from main func
// WHEN VERRAZZANO_ROOT env var is not specified
// THEN an error is returned
func TestVerrazzanoRootNotSpecified(t *testing.T) {
	assert := assert.New(t)
	os.Unsetenv(VzRootDir)
	err := run("prod", "foo")
	assert.Error(err)
	assert.Equal(err, fmt.Errorf("VERRAZZANO_ROOT environment variable not specified"))
}

// TestInvalidOutputLocation tests the following scenario
// GIVEN a call to run func
// WHEN the outputLocation is invalid
// THEN an error is returned
func TestInvalidOutputLocation(t *testing.T) {
	assert := assert.New(t)
	os.Setenv(VzRootDir, vzDir)
	err := run("prod", "${HOME}/foo")
	assert.Error(err)
	assert.ErrorContains(err, "foo: no such file or directory")
}

// TestInvalidProfileType tests the following scenario
// GIVEN a call to generateProfile
// WHEN profile is found to be invalid
// THEN an error is returned containing the message that the profile file was not found
func TestInvalidProfileType(t *testing.T) {
	assert := assert.New(t)
	_, err := generateProfile("foo", vzDir)
	assert.Error(err)
	assert.ErrorContains(err, "foo.yaml: no such file or directory")
}

// TestValidProfileType tests the following scenario
// GIVEN a call to generate cr of a profileType
// WHEN the profileType is found to be valid
// THEN no error is returned
func TestValidProfileType(t *testing.T) {
	assert := assert.New(t)
	_, err := generateProfile("dev", vzDir)
	assert.NoError(err)
}

// TestFieldsOmitted tests the following scenario
// GIVEN a call to generate cr of a profileType
// WHEN the profileType is found to be valid
// THEN no error is returned and certain fields
// that were omitted during serialization are not found
func TestFieldsOmitted(t *testing.T) {
	// Expect certain fields to be missing if yaml.Marshal
	// is called on the cr Alias type
	assert := assert.New(t)
	cr, err := generateProfile("dev", vzDir)
	assert.NoError(err)
	crYaml, err := yaml.Marshal(cr)
	assert.NoError(err)
	assert.False(strings.Contains(string(crYaml), "status"))
	assert.False(strings.Contains(string(crYaml), "creationTimestamp"))
	assert.True(strings.Contains(string(crYaml), "spec"))
	assert.True(strings.Contains(string(crYaml), "metadata"))

	// When yaml.Marshal method is called on an object if type v1beta1.Verrazzano
	// then expect status and creationTimestamp to show up
	vzCR := v1beta1.Verrazzano(*cr)
	vzCRYaml, err := yaml.Marshal(&vzCR)
	assert.NoError(err)
	assert.True(strings.Contains(string(vzCRYaml), "status"))
	assert.True(strings.Contains(string(vzCRYaml), "creationTimestamp"))
}
