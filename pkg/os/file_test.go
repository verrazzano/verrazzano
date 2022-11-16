// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package os

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestCreateTempFile tests the CreateTempFile function
// GIVEN a call to CreateTempFile
//
//	WHEN with a valid prefix and/or suffix
//	THEN the temp file is created
func TestCreateTempFile(t *testing.T) {
	var tests = []struct {
		name          string
		createPattern string
	}{
		{name: "WithPrefixAndSuffix", createPattern: "test1-*.yaml"},
		{name: "WithPrefixOnly", createPattern: "test2-*"},
		{name: "WithSuffixOnly", createPattern: "*-test3"},
		{name: "NoPrefixNorSuffix", createPattern: "test4"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			temp, err := CreateTempFile(test.createPattern, nil)
			defer RemoveTempFiles(zap.S(), temp.Name())
			assert.FileExists(temp.Name())
			fmt.Println(temp.Name())
			assert.NoErrorf(err, "Unable to create temp file %s: %s", temp.Name(), err)
		})
	}

}

// TestRemoveTempFiles tests the RemoveTempFiles function
// GIVEN a call to RemoveTempFiles
//
//	WHEN with a valid prefix and/or suffix
//	THEN any matching temp files are correctly deleted
func TestRemoveTempFiles(t *testing.T) {

	tests := []struct {
		name          string
		createPattern string
		deletePattern string
	}{
		{name: "WithPrefixAndSuffix", createPattern: "test1-*.yaml", deletePattern: "test1-.*\\.yaml"},
		{name: "WithPrefixOnly", createPattern: "test2-*", deletePattern: "test2-.*"},
		{name: "WithSuffixOnly", createPattern: "*-mike", deletePattern: ".*-mike"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			temp, err := os.CreateTemp(os.TempDir(), test.createPattern)
			assert.NoErrorf(err, "Unable to create temp file %s for testing: %s", temp.Name(), err)
			assert.FileExists(temp.Name())
			err = RemoveTempFiles(zap.S(), test.deletePattern)
			assert.NoErrorf(err, "Unable to remove temp file %s: %s", temp.Name(), err)
			assert.NoFileExists(temp.Name())
		})
	}
	// Verify error returned on invalid regex pattern
	assert.Error(t, RemoveTempFiles(zap.S(), "["))
}

// TestFileExists tests the FileExists function
// GIVEN a call to FileExists
//
//	WHEN with a valid file path
//	THEN the file name is checked for existence
func TestFileExists(t *testing.T) {
	tests := []struct {
		name          string
		createPattern string
	}{
		{name: "WithPrefixAndSuffix", createPattern: "test1-*.yaml"},
		{name: "WithPrefixOnly", createPattern: "test2-*"},
		{name: "WithSuffixOnly", createPattern: "*-test3"},
		{name: "NoPrefixNorSuffix", createPattern: "test"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			tmpFile, err := CreateTempFile(test.createPattern, nil)
			assert.NoErrorf(err, "Unable to create temp file %s for testing: %s", tmpFile, err)
			exists, err := FileExists(tmpFile.Name())
			assert.True(exists)
			assert.NoError(err)
			//Returns false when file does not exist
			exists, err = FileExists("nofile")
			assert.False(exists)
			assert.NoError(err)
		})
	}
}
