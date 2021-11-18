// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"testing"

	asserts "github.com/stretchr/testify/assert"
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
