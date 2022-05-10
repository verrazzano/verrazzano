// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package utils_test

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/klog"
	"github.com/verrazzano/verrazzano/verrazzano-backup/lib/utils"
	"go.uber.org/zap"
	"os"
	"strings"
	"testing"
)

func logHelper() (*zap.SugaredLogger, string) {
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("verrazzano-%s-hook-*.log", strings.ToLower("TEST")))
	if err != nil {
		fmt.Printf("Unable to create temp file")
		os.Exit(1)
	}
	defer file.Close()
	log, _ := klog.Logger(file.Name())
	return log, file.Name()
}

// TestCreateTempFileWithData tests the CreateTempFileWithData method create a temp file for snapshot registration
// GIVEN input data as a []byte
// WHEN file needs to be created as a temp file
// THEN creates a files under temp and returns the filepath
func TestCreateTempFileWithData(t *testing.T) {
	nullBody := make(map[string]interface{})
	data, _ := json.Marshal(nullBody)
	file, err := utils.CreateTempFileWithData(data)
	defer os.Remove(file)
	assert.Nil(t, err)
	assert.NotNil(t, file)
}

// TestGenerateRandom tests the GenerateRandom method
// GIVEN min and max limits
// WHEN invoked from another method
// THEN generates a crypto safe random number in a predefined range
func TestGenerateRandom(t *testing.T) {
	d := utils.GenerateRandom()
	assert.NotNil(t, d)
}

// TestHTTPHelper tests the HTTPHelper method for the following use case.
// GIVEN a URL
// WHEN the method type is GET/POST/DELETE with a body
// THEN Invoke the http request and return the appropriate response
func TestHTTPHelper(t *testing.T) {
	log, fname := logHelper()
	defer os.Remove(fname)
	URL := "http://127.0.0.1"
	data := make(map[string]interface{})

	err := utils.HTTPHelper("GET", URL, nil, &data, log)
	assert.NotNil(t, err)

	err = utils.HTTPHelper("POST", URL, nil, &data, log)
	assert.NotNil(t, err)

	err = utils.HTTPHelper("DELETE", URL, nil, &data, log)
	assert.NotNil(t, err)
}

// TestReadTempCredsFile tests the ReadTempCredsFile method for the following use case.
// GIVEN an existing file to read
// WHEN the file exists
// THEN read the keys from the file
func TestReadTempCredsFile(t *testing.T) {
	file, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("test-%s-hook-*.log", strings.ToLower("TEST")))
	if err != nil {
		fmt.Printf("Unable to create temp file")
		os.Exit(1)
	}
	file.Close()
	data1, data2, err := utils.ReadTempCredsFile(file.Name())
	assert.Equal(t, "", data1)
	assert.Equal(t, "", data2)
	assert.Nil(t, err)
	os.Remove(file.Name())

	fileNotExist := "/tmp/foo.txt"
	data1, data2, err = utils.ReadTempCredsFile(fileNotExist)
	assert.Equal(t, "", data1)
	assert.Equal(t, "", data2)
	assert.Nil(t, err)
}
