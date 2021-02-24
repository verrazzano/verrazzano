// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"testing"

	asserts "github.com/stretchr/testify/assert"
)

// Test_stringSliceContainsString tests metrics trait utility function stringSliceContainsString
func Test_stringSliceContainsString(t *testing.T) {
	assert := asserts.New(t)
	var slice []string
	var find string
	var found bool

	// GIVEN a nil slice
	// WHEN an empty string is searched for
	// THEN verify false is returned
	slice = nil
	found = StringSliceContainsString(slice, find)
	assert.Equal(found, false)

	// GIVEN a slice with several strings
	// WHEN one of the strings is searched for
	// THEN verify string is found
	slice = []string{"test-value-1", "test-value-2", "test-value-3"}
	find = "test-value-2"
	found = StringSliceContainsString(slice, find)
	assert.Equal(found, true)

	// GIVEN a slice with several strings
	// WHEN a string not in the slice is searched for
	// THEN verify string is not found
	slice = []string{"test-value-1", "test-value-2", "test-value-3"}
	find = "test-value-4"
	found = StringSliceContainsString(slice, find)
	assert.Equal(found, false)
}

// Test_removeStringFromStringSlice tests metrics trait utility function removeStringFromStringSlice
func Test_removeStringFromStringSlice(t *testing.T) {
	assert := asserts.New(t)
	var slice []string
	var remove string
	var output []string

	// GIVEN a nil slice and an empty string to remove
	// WHEN the empty string is removed from the nil slice
	// THEN verify that an empty slice is returned
	slice = nil
	remove = ""
	output = RemoveStringFromStringSlice(slice, remove)
	assert.NotNil(output)
	assert.Len(output, 0)

	// GIVEN a slice with several strings
	// WHEN a string in the slice is removed
	// THEN verify slice is correct
	slice = []string{"test-value-1", "test-value-2", "test-value-3"}
	remove = "test-value-2"
	output = RemoveStringFromStringSlice(slice, remove)
	assert.Equal("test-value-1", slice[0])
	assert.Equal("test-value-2", slice[1])
	assert.Len(output, 2)
}

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
