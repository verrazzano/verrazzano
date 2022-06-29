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

// TestSearchFilesGood Tests that we can find the expected set of files with a matching expression
// GIVEN a call to SearchFiles
// WHEN with a valid rootDirectory, list of files, and regular expression
// THEN search matches will be returned
func TestSearchFilesGood(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	rootDirectory := "../../../test"
	myFiles, err := GetMatchingFiles(logger, rootDirectory, regexp.MustCompile(".*"))
	assert.Nil(t, err)
	assert.NotNil(t, myFiles)
	assert.True(t, len(myFiles) > 0)
	myMatches, err := SearchFiles(logger, rootDirectory, myFiles, regexp.MustCompile("ghcr.io/.*/rancher"), nil)
	assert.Nil(t, err)
	assert.NotNil(t, myMatches)
	assert.True(t, len(myMatches) > 0)
	for _, match := range myMatches {
		assert.True(t, len(checkMatch(logger, match)) == 0)
	}
}

// TestFindFilesAndSearchGood Tests that we can find the expected set of files with a matching expression
// GIVEN a call to FindFilesAndSearch
// WHEN with a valid rootDirectory, list of files, and regular expression
// THEN search matches will be returned
func TestFindFilesAndSearchGood(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	myMatches, err := FindFilesAndSearch(logger, "../../../test", regexp.MustCompile(".*"), regexp.MustCompile("ghcr.io/.*/rancher"), nil)
	assert.Nil(t, err)
	assert.NotNil(t, myMatches)
	assert.True(t, len(myMatches) > 0)
	for _, match := range myMatches {
		assert.True(t, len(checkMatch(logger, match)) == 0)
	}
}

// TestBadExpressions Tests that we fail correctly when given bad search expressions
// GIVEN a call to helpers
// WHEN with invalid regular expression
// THEN error will be returned
func TestBadExpressions(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	_, err := FindFilesAndSearch(logger, "../../../test", nil, regexp.MustCompile("ghcr.io/.*/rancher"), nil)
	assert.NotNil(t, err)
	_, err = FindFilesAndSearch(logger, "../../../test", regexp.MustCompile(".*"), nil, nil)
	assert.NotNil(t, err)
	_, err = GetMatchingFiles(logger, "../../../test", nil)
	assert.NotNil(t, err)
	myFiles := []string{"test file"}
	_, err = SearchFiles(logger, "../../../test", myFiles, nil, nil)
	assert.NotNil(t, err)
}

func checkMatch(logger *zap.SugaredLogger, match TextMatch) string {
	logger.Debugf("Matched file: %s", match.FileName)
	logger.Debugf("Matched line: %d", match.FileLine)
	logger.Debugf("Matched text: %s", len(match.MatchedText))
	failText := ""
	stat, err := os.Stat(match.FileName)
	if err != nil {
		logger.Errorf("Stat failed for matched file: %s", match.FileName, err)
		failText = fmt.Sprintf("Stat failed for matched file: %s", match.FileName)
	} else if stat.IsDir() {
		failText = fmt.Sprintf("Matched file was a directory: %s", match.FileName)
	} else if match.FileLine <= 0 {
		failText = fmt.Sprintf("Matched linenumber %d was invalid for: %s", match.FileLine, match.FileName)
	} else if len(match.MatchedText) == 0 {
		failText = fmt.Sprintf("Matched text was empty for linenumber %d was invalid for: %s", match.FileLine, match.FileName)
	}
	if len(failText) > 0 {
		logger.Error(failText)
	}
	return failText
}

// TODO: Add more test cases (more result validation, time ranges, more expression variants, negative cases, etc...)
