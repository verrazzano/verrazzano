// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"testing"

	asserts "github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
)

// TestConvertAPIVersionToGroupAndVersion tests multiple use cases for parsing APIVersion
func TestConvertAPIVersionToGroupAndVersion(t *testing.T) {
	assert := asserts.New(t)
	var g, v string

	// GIVEN a normal group/version string
	// WHEN it is parsed into group and version parts
	// THEN ensure the parts are correct.
	g, v = ConvertAPIVersionToGroupAndVersion("group/version")
	assert.Equal("group", g)
	assert.Equal("version", v)

	// GIVEN a normal group/version string with no group.
	// WHEN it is parsed into group and version parts
	// THEN ensure the group is the empty string and the version is correct.
	// This is the case for older standard kubernetes core resources.
	g, v = ConvertAPIVersionToGroupAndVersion("/version")
	assert.Equal("", g)
	assert.Equal("version", v)

	// GIVEN a normal group/version string with no group.
	// WHEN it is parsed into group and version parts
	// THEN ensure the group is the empty string and the version is correct.
	// This is the case for older standard kubernetes core resources.
	g, v = ConvertAPIVersionToGroupAndVersion("version")
	assert.Equal("", g)
	assert.Equal("version", v)
}

// TestIsWorkloadMarkedForUpgrade tests IsWorkloadMarkedForUpgrade to ensure that it returns the correct response
// based on a map of labels and a current version.
func TestIsWorkloadMarkedForUpgrade(t *testing.T) {
	assert := asserts.New(t)
	labels := map[string]string{"foo": "bar", constants.LabelUpgradeVersion: "12345"}

	// GIVEN a current upgrade version that matches the corresponding label
	// WHEN IsWorkloadMarkedForUpgrade is called
	// THEN false is returned
	assert.False(IsWorkloadMarkedForUpgrade(labels, "12345"))

	// GIVEN a current upgrade version that doesn't match the corresponding label
	// WHEN IsWorkloadMarkedForUpgrade is called
	// THEN true is returned
	assert.True(IsWorkloadMarkedForUpgrade(labels, "99999"))
}
