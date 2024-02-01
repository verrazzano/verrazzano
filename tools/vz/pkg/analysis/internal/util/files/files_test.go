// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package files

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"go.uber.org/zap"
)

// TestGetMatchingFilesGood Tests that we can find the expected set of files with a matching expression
// GIVEN a call to GetMatchingDirectories
// WHEN with a valid rootDirectory and regular expression
// THEN files that matched will be returned
func TestGetMatchingFilesGood(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	myFiles, err := GetMatchingFiles(logger, "../../../test/json", regexp.MustCompile(`node.*\.json$`))
	assert.Nil(t, err)
	assert.NotNil(t, myFiles)
	assert.True(t, len(myFiles) > 0)
	for _, file := range myFiles {
		assert.True(t, len(checkIsRegularFile(logger, file)) == 0)
	}
	myFiles, err = GetMatchingFiles(logger, "../../../test/json", regexp.MustCompile(`node.*\.none_shall_match`))
	assert.Nil(t, err)
	assert.Nil(t, myFiles)
}

// TestGetMatchingDirectoriesGood Tests that we can find the expected set of files with a matching expression
// GIVEN a call to GetMatchingDirectories
// WHEN with a valid rootDirectory and regular expression
// THEN files that matched will be returned
func TestGetMatchingDirectoriesGood(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	// the .*son will match directories with names like "json"
	myFiles, err := GetMatchingDirectories(logger, "../../../test", regexp.MustCompile(".*son$"))
	assert.Nil(t, err)
	assert.NotNil(t, myFiles)
	assert.True(t, len(myFiles) > 0)
	for _, file := range myFiles {
		assert.True(t, len(checkIsDirectory(logger, file)) == 0)
	}
	myFiles, err = GetMatchingDirectories(logger, "../../../test", regexp.MustCompile("none_shall_match"))
	assert.Nil(t, err)
	assert.Nil(t, myFiles)
}

// TestGetMatchingBad Tests that we can find the expected set of files with a matching expression
// GIVEN a call to GetMatching* utilities
// WHEN with invalid inputs
// THEN we get failures as expected
func TestGetMatchingInvalidInputs(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	_, err := GetMatchingDirectories(logger, "../../../test", nil)
	assert.NotNil(t, err)
	filesFound, err := GetMatchingDirectories(logger, "../../../test-not-found", regexp.MustCompile(".*son$"))
	assert.Nil(t, err)
	assert.Nil(t, filesFound)
	_, err = GetMatchingFiles(logger, "../../../test", nil)
	assert.NotNil(t, err)
	filesFound, err = GetMatchingFiles(logger, "../../../test-not-found", regexp.MustCompile(".*son$"))
	assert.Nil(t, err)
	assert.Nil(t, filesFound)

}

// TestMiscUtils Tests that the misc small utilities work as expected
// GIVEN a call to GetMiscUtils
// WHEN with good and bad inputs
// THEN utility functions behave as expected
func TestMiscUtils(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	filename := FindFileInClusterRoot("../../../test/cluster/problem-pods/cluster-snapshot/problem-pods", "default")
	assert.NotNil(t, filename)
	namespaces, err := FindNamespaces(logger, "../../../test/cluster/problem-pods/cluster-snapshot")
	assert.Nil(t, err)
	assert.NotNil(t, namespaces)
	assert.True(t, len(namespaces) > 0)
	_, err = FindNamespaces(logger, "../../../test/problem-pods/not-found")
	assert.NotNil(t, err)
}

// TODO: Add more test cases (more expression variants, negative cases, etc...)

func checkIsDirectory(logger *zap.SugaredLogger, fileName string) string {
	failText := ""
	stat, err := os.Stat(fileName)
	if err != nil {
		logger.Errorf("Stat failed for file: %s", fileName, err)
		failText = fmt.Sprintf("Stat failed for file: %s", fileName)
	} else if !stat.IsDir() {
		failText = fmt.Sprintf("Matched file was not a directory: %s", fileName)
	}
	if len(failText) > 0 {
		logger.Error(failText)
	}
	return failText
}

func checkIsRegularFile(logger *zap.SugaredLogger, fileName string) string {
	failText := ""
	stat, err := os.Stat(fileName)
	if err != nil {
		logger.Errorf("Stat failed for file: %s", fileName, err)
		failText = fmt.Sprintf("Stat failed for file: %s", fileName)
	} else if stat.IsDir() {
		failText = fmt.Sprintf("Matched file was a directory: %s", fileName)
	} else if !stat.Mode().IsRegular() {
		failText = fmt.Sprintf("Matched file was not a regular file: %s", fileName)
	}
	if len(failText) > 0 {
		logger.Error(failText)
	}
	return failText
}

// TestGetTimeOfCapture tests that a metadata.json file can be successfully parsed and a time.Time object is created without error
// GIVEN a metadata.json file
// WHEN I call GetTimeOfCapture and pass in this file
// THEN expect it to successfully create a time.Time object with the correct information and no error should be returned.
func TestGetTimeOfCapture(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	timeObject, err := GetTimeOfCapture(logger, "../../../test/cluster/multiple-namespaces-stuck-terminating-on-finalizers/cluster-snapshot")
	assert.NotNil(t, timeObject)
	assert.Nil(t, err)
	assert.True(t, timeObject.UTC().Format(time.RFC3339) == "2024-01-24T13:44:11Z")

}
