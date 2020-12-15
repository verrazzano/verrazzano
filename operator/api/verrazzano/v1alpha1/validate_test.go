// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1alpha1

import (
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

const validChartYAML = `
# Copyright (c) 2020, Oracle and/or its affiliates.
# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
apiVersion: v1
description: A Helm chart for Verrazzano
name: verrazzano
version: 0.7.0
appVersion: 0.7.0
`

// TestValidUpgradeRequestNoCurrentVersion Tests the SemVersion parser for valid version strings
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the current version is not specified
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeRequestNoCurrentVersion(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
	}
	assert.Nil(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidUpgradeRequestCurrentVersionExists Tests the SemVersion parser for valid version strings
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version is valid and the current version is less than the current version
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeRequestCurrentVersionExists(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Version: "v0.6.0",
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
	}
	assert.Nil(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestValidUpgradeRequestCurrentVersionExists Tests the SemVersion parser for valid version strings
// GIVEN an edit to update a Verrazzano spec to a new version
// WHEN the new version and the current version are at the latest version
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestValidUpgradeNotNecessary(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Version: "v0.7.0",
		Profile: "dev",
	}
	assert.Nil(t, ValidateUpgradeRequest(currentSpec, newSpec))
}

// TestNoVersionsSpecified Tests the SemVersion parser for valid version strings
// GIVEN an edit to update a Verrazzano spec
// WHEN the new version and the current version are not specified
// THEN ensure no error is returned from ValidateUpgradeRequest
func TestNoVersionsSpecified(t *testing.T) {
	chartYaml := validChartYAML
	readFileFunction = func(string) ([]byte, error) {
		return []byte(chartYaml), nil
	}
	defer func() {
		readFileFunction = ioutil.ReadFile
	}()
	currentSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	newSpec := &VerrazzanoSpec{
		Profile: "dev",
	}
	assert.Nil(t, ValidateUpgradeRequest(currentSpec, newSpec))
}
