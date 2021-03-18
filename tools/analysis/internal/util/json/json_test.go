// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package json

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/log"
	"reflect"
	"testing"
)

// TestGetJSONDataFromFileGoodData Tests that we can get Json data from a valid Json file
// GIVEN a call to getJsonDataFromFile
// WHEN with a valid json file path
// THEN valid json data will be returned
func TestGetJSONDataFromFileGoodData(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	myData, err := GetJSONDataFromFile(logger, "../../../test/json/nodes.json")
	assert.Nil(t, err)
	assert.NotNil(t, myData)
	apiVersion, err := GetJSONValue(logger, myData, "apiVersion")
	assert.Nil(t, err)
	assert.NotNil(t, apiVersion)
	assert.Equal(t, apiVersion, "v1")

	resVersion, err := GetJSONValue(logger, myData, "metadata.resourceVersion")
	assert.Nil(t, err)
	assert.NotNil(t, resVersion)
	fmt.Printf("metadata.resourceVersion: %s\n", resVersion)

	names, err := GetJSONValue(logger, myData, "items.metadata.name")
	assert.Nil(t, err)
	assert.NotNil(t, names)
	fmt.Printf("items.metadata.name: %s\n", names)

	images, err := GetJSONValue(logger, myData, "items.status.images.names")
	assert.Nil(t, err)
	assert.NotNil(t, images)
	fmt.Printf("items.status.images.names: %s\n", images)

	// Get it again, this should find it in the cache
	myData, err = GetJSONDataFromFile(logger, "../../../test/json/nodes.json")
	assert.Nil(t, err)
	assert.NotNil(t, myData)
	assert.True(t, cacheHits > 0)

	// Make sure we can call debugMap
	debugMap(logger, myData.(map[string]interface{}))
}

// TestGetJSONDataFromFileFileNotFound Tests that we fail as expected when Json file is not found
// GIVEN a call to getJsonDataFromFile
// WHEN with an invalid json file path
// THEN we will fail as expected
func TestGetJSONDataFromFileFileNotFound(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	_, err := GetJSONDataFromFile(logger, "file-not-found.json")
	assert.NotNil(t, err)
}

// TestGetJSONDataFromFileBadData Tests that we fail as expected when file with invalid Json format is supplied
// GIVEN a call to getJsonDataFromFile
// WHEN with a file with invalid Json data
// THEN we will fail as expected
func TestGetJSONDataFromFileBadData(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	_, err := GetJSONDataFromFile(logger, "../../../test/json/bogus.json")
	assert.NotNil(t, err)
}

// TestGetJSONArrays Tests that we can get Json array data valid Json files
// GIVEN a call to getJsonDataFromFile
// WHEN with a valid json file paths with array variants
// THEN valid json data will be returned
func TestGetJSONArrays(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	myData, err := GetJSONDataFromFile(logger, "../../../test/json/basic_array.json")
	assert.Nil(t, err)
	assert.NotNil(t, myData)
	value, err := GetJSONValue(logger, myData, "[1]")
	assert.Nil(t, err)
	assert.NotNil(t, value)
	assert.Equal(t, value, "one")
	arrayValue, err := GetJSONValue(logger, myData, "[]")
	assert.Nil(t, err)
	assert.NotNil(t, arrayValue)
	assert.True(t, len(arrayValue.([]interface{})) == 4)
	noNameArrayValue, err := GetJSONValue(logger, myData, "")
	assert.Nil(t, err)
	assert.NotNil(t, noNameArrayValue)
	assert.True(t, len(noNameArrayValue.([]interface{})) == 4)
	assert.True(t, reflect.DeepEqual(arrayValue, noNameArrayValue))
}

// TODO: Add many more variants of data to these, file read access issues, etc...
