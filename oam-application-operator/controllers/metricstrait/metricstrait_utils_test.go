// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package metricstrait

import (
	"github.com/Jeffail/gabs/v2"
	asserts "github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

// Test_updateStringMap tests metrics trait utility function updateStringMap
func Test_updateStringMap(t *testing.T) {
	assert := asserts.New(t)
	var input map[string]string
	var output map[string]string

	// GIVEN a nil input map
	// WHEN a new name value pair are added
	// THEN verify a map is returned containing the new name value pair.
	input = nil
	output = updateStringMap(input, "test-name-1", "test-value-1")
	assert.Len(output, 1)
	assert.Equal("test-value-1", output["test-name-1"])

	// GIVEN an empty input map
	// WHEN a new name value pair are added
	// THEN verify a map is returned containing the new name value pair.
	input = map[string]string{}
	output = updateStringMap(input, "test-name-1", "test-value-1")
	assert.Len(output, 1)
	assert.Equal("test-value-1", output["test-name-1"])

	// GIVEN an map with an existing name/value pair
	// WHEN a new value is set for an existing name
	// THEN verify a map contains the new value
	input = map[string]string{"test-name-1": "test-value-1"}
	output = updateStringMap(input, "test-name-1", "test-value-2")
	assert.Len(output, 1)
	assert.Equal("test-value-2", output["test-name-1"])

	// GIVEN an map with an existing name/value pair
	// WHEN a new name and value is set
	// THEN verify a map contains both the old and the new pairs
	input = map[string]string{"test-name-1": "test-value-1"}
	output = updateStringMap(input, "test-name-2", "test-value-2")
	assert.Len(output, 2)
	assert.Equal("test-value-1", output["test-name-1"])
	assert.Equal("test-value-2", output["test-name-2"])
}

// Test_copyStringMapEntries tests metrics trait utility function copyStringMapEntries
func Test_copyStringMapEntries(t *testing.T) {
	assert := asserts.New(t)
	var source map[string]string
	var target map[string]string
	var output map[string]string

	// GIVEN nil source and target maps
	// WHEN a key name is copied from source to target
	// THEN verify the target map is empty
	source = nil
	target = nil
	output = copyStringMapEntries(target, source, "test-name-1")
	assert.NotNil(output)
	assert.Len(output, 0)

	// GIVEN empty source and target maps
	// WHEN a key name is copied from source to target
	// THEN verify the target map is empty
	source = map[string]string{}
	target = map[string]string{}
	output = copyStringMapEntries(target, source, "test-name-1")
	assert.NotNil(output)
	assert.Len(output, 0)

	// GIVEN empty source and target maps
	// WHEN a key name is copied from source to target
	// THEN verify the output and target map have two entries
	source = map[string]string{"test-name-1": "test-value-1"}
	target = map[string]string{"test-name-2": "test-value-2"}
	output = copyStringMapEntries(target, source, "test-name-1")
	assert.NotNil(output)
	assert.Equal("test-value-1", output["test-name-1"])
	assert.Equal("test-value-2", output["test-name-2"])
	assert.Len(output, 2)
	assert.Len(source, 1)
	assert.Len(target, 2)
}

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
	found = stringSliceContainsString(slice, find)
	assert.Equal(found, false)

	// GIVEN a slice with several strings
	// WHEN one of the strings is searched for
	// THEN verify string is found
	slice = []string{"test-value-1", "test-value-2", "test-value-3"}
	find = "test-value-2"
	found = stringSliceContainsString(slice, find)
	assert.Equal(found, true)

	// GIVEN a slice with several strings
	// WHEN a string not in the slice is searched for
	// THEN verify string is not found
	slice = []string{"test-value-1", "test-value-2", "test-value-3"}
	find = "test-value-4"
	found = stringSliceContainsString(slice, find)
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
	output = removeStringFromStringSlice(slice, remove)
	assert.NotNil(output)
	assert.Len(output, 0)

	// GIVEN a slice with several strings
	// WHEN a string in the slice is removed
	// THEN verify slice is correct
	slice = []string{"test-value-1", "test-value-2", "test-value-3"}
	remove = "test-value-2"
	output = removeStringFromStringSlice(slice, remove)
	assert.Equal("test-value-1", slice[0])
	assert.Equal("test-value-2", slice[1])
	assert.Len(output, 2)
}

// Test_getClusterNameFromObjectMetaOrDefault tests metrics trait utility function getClusterNameFromObjectMetaOrDefault
func Test_getClusterNameFromObjectMetaOrDefault(t *testing.T) {
	assert := asserts.New(t)
	var meta metav1.ObjectMeta
	var name string

	// GIVEN metadata with a blank cluster name
	// WHEN the cluster name is retrieved
	// THEN verify the "default" cluster name is returned
	name = getClusterNameFromObjectMetaOrDefault(meta)
	assert.Equal("default", name)

	// GIVEN metadata with a non-blank cluster name
	// WHEN the cluster name is retrieved
	// THEN verify the correct cluster name is returned
	meta = metav1.ObjectMeta{ClusterName: "test-cluster-name-1"}
	name = getClusterNameFromObjectMetaOrDefault(meta)
	assert.Equal("test-cluster-name-1", name)
}

// Test_getNamespaceFromObjectMetaOrDefault tests metrics trait utility function getNamespaceFromObjectMetaOrDefault
func Test_getNamespaceFromObjectMetaOrDefault(t *testing.T) {
	assert := asserts.New(t)
	var meta metav1.ObjectMeta
	var name string

	// GIVEN metadata with a blank namespace name
	// WHEN the namespace name is retrieved
	// THEN verify the "default" namespace name is returned
	name = getNamespaceFromObjectMetaOrDefault(meta)
	assert.Equal("default", name)

	// GIVEN metadata with a non-blank namespace name
	// WHEN the namespace name is retrieved
	// THEN verify the correct namespace name is returned
	meta = metav1.ObjectMeta{Namespace: "test-namespace-1"}
	name = getNamespaceFromObjectMetaOrDefault(meta)
	assert.Equal("test-namespace-1", name)
}

// Test_parseYAMLString tests metrics trait utility function parseYAMLString
//func parseYAMLString(s string) (*gabs.Container, error) {
func Test_parseYAMLString(t *testing.T) {
	assert := asserts.New(t)
	var cont *gabs.Container
	var str string
	var err error

	// GIVEN an empty yaml string
	// WHEN the yaml string is parsed
	// THEN verify that the unstructured objects are empty
	str = ""
	cont, err = parseYAMLString(str)
	assert.NoError(err)
	assert.Equal(nil, cont.Data())

	// GIVEN an invalid yaml string
	// WHEN the yaml string is parsed
	// THEN verify that an error is returned
	str = ":"
	cont, err = parseYAMLString(str)
	assert.Error(err)

	// GIVEN an simple yaml string
	// WHEN the yaml string is parsed
	// THEN verify that the unstructured objects contain the correct data
	str = "test-name-1: test-value-1"
	cont, err = parseYAMLString(str)
	assert.NoError(err)
	assert.Equal("test-value-1", cont.Path("test-name-1").Data().(string))
}

// Test_writeYAMLString tests metrics trait utility function writeYAMLString
func Test_writeYAMLString(t *testing.T) {
	assert := asserts.New(t)
	var str string
	var err error
	var cont *gabs.Container

	// GIVEN an simple yaml container
	// WHEN the yaml container is written to a string
	// THEN verify that the string is correct
	cont = gabs.New()
	cont.Set("test-value-1", "test-name-1")
	str, err = writeYAMLString(cont)
	assert.NoError(err)
	assert.Equal("test-name-1: test-value-1\n", str)
}

// Test_mergeTemplateWithContext tests metrics trait utility function mergeTemplateWithContext
func Test_mergeTemplateWithContext(t *testing.T) {
	assert := asserts.New(t)
	var input string
	var output string
	var context map[string]string

	// GIVEN an empty template and nil context
	// WHEN the template and context are merged
	// THEN verify that the result is an empty string
	input = ""
	context = nil
	output = mergeTemplateWithContext(input, context)
	assert.Equal("", output)

	// GIVEN an template with no placeholders and nil context
	// WHEN the template and context are merged
	// THEN verify that the result is the same as the input
	input = "no-place-holders"
	context = nil
	output = mergeTemplateWithContext(input, context)
	assert.Equal("no-place-holders", output)

	// GIVEN an template with duplicate placeholder and a newline
	// WHEN the context contains a value for the placeholder
	// THEN verify that the output has the value twice with a newline separating them
	input = "{{template-name-1}}\n{{template-name-1}}"
	context = map[string]string{"{{template-name-1}}": "template-value-2"}
	output = mergeTemplateWithContext(input, context)
	assert.Equal("template-value-2\ntemplate-value-2", output)
}
