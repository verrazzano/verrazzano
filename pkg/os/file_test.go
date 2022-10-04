// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package os

import (
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"os"
	"testing"
)

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
