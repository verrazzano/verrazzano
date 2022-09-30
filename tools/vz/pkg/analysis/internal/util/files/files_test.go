// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package files

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"go.uber.org/zap"
	"os"
	"regexp"
	"testing"
)

const testDir = "../../../test"
const testNotFound = "../../../test-not-found"
const jsonRegex = ".*son$"
const templateStatFailed = "Stat failed for file: %s"

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
	myFiles, err := GetMatchingDirectories(logger, testDir, regexp.MustCompile(jsonRegex))
	assert.Nil(t, err)
	assert.NotNil(t, myFiles)
	assert.True(t, len(myFiles) > 0)
	for _, file := range myFiles {
		assert.True(t, len(checkIsDirectory(logger, file)) == 0)
	}
	myFiles, err = GetMatchingDirectories(logger, testDir, regexp.MustCompile("none_shall_match"))
	assert.Nil(t, err)
	assert.Nil(t, myFiles)
}

// TestGetMatchingBad Tests that we can find the expected set of files with a matching expression
// GIVEN a call to GetMatching* utilities
// WHEN with invalid inputs
// THEN we get failures as expected
func TestGetMatchingInvalidInputs(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	_, err := GetMatchingDirectories(logger, testDir, nil)
	assert.NotNil(t, err)
	filesFound, err := GetMatchingDirectories(logger, testNotFound, regexp.MustCompile(jsonRegex))
	assert.Nil(t, err)
	assert.Nil(t, filesFound)
	_, err = GetMatchingFiles(logger, testDir, nil)
	assert.NotNil(t, err)
	filesFound, err = GetMatchingFiles(logger, testNotFound, regexp.MustCompile(jsonRegex))
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

// Add more test cases (more expression variants, negative cases, etc...)

func checkIsDirectory(logger *zap.SugaredLogger, fileName string) string {
	failText := ""
	stat, err := os.Stat(fileName)
	if err != nil {
		logger.Errorf(templateStatFailed, fileName, err)
		failText = fmt.Sprintf(templateStatFailed, fileName)
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
		logger.Errorf(templateStatFailed, fileName, err)
		failText = fmt.Sprintf(templateStatFailed, fileName)
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
