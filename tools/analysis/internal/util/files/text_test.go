// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package files

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/log"
	"go.uber.org/zap"
	"os"
	"testing"
)

// TestSearchFilesGood Tests that we can find the expected set of files with a matching expression
// GIVEN a call to SearchFiles
// WHEN with a valid rootDirectory, list of files, and regular expression
// THEN search matches will be returned
func TestSearchFilesGood(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	rootDirectory := "../../../test"
	myFiles, err := GetMatchingFiles(logger, rootDirectory, ".*")
	assert.Nil(t, err)
	assert.NotNil(t, myFiles)
	assert.True(t, len(myFiles) > 0)
	myMatches, err := SearchFiles(logger, rootDirectory, myFiles, "ghcr.io/.*/rancher")
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
	myMatches, err := FindFilesAndSearch(logger, "../../../test", ".*", "ghcr.io/.*/rancher")
	assert.Nil(t, err)
	assert.NotNil(t, myMatches)
	assert.True(t, len(myMatches) > 0)
	for _, match := range myMatches {
		assert.True(t, len(checkMatch(logger, match)) == 0)
	}
}

// TestGetAllMatches Tests that we can find the expected set of files with a matching expression
// GIVEN a call to GetAllMatches
// WHEN with a valid rootDirectory, list of files, and regular expression
// THEN search matches will be returned
func TestGetAllMatches(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	matched, err := GetAllMatches(logger, []byte("testAAthisAAoutAA"), "AA", 2)
	assert.Nil(t, err)
	assert.NotNil(t, matched)
	assert.True(t, len(matched) == 2)
}

// TestBadExpressions Tests that we fail correctly when given bad search expressions
// GIVEN a call to helpers
// WHEN with invalid regular expression
// THEN error will be returned
func TestBadExpressions(t *testing.T) {
	logger := log.GetDebugEnabledLogger()
	badExpression := "~(foo"
	_, err := GetAllMatches(logger, []byte("testAAthisAAoutAA"), badExpression, 2)
	assert.NotNil(t, err)
	_, err = FindFilesAndSearch(logger, "../../../test", badExpression, "ghcr.io/.*/rancher")
	assert.NotNil(t, err)
	_, err = FindFilesAndSearch(logger, "../../../test", ".*", badExpression)
	assert.NotNil(t, err)
	_, err = GetMatchingFiles(logger, "../../../test", badExpression)
	assert.NotNil(t, err)
	myFiles := []string{"test file"}
	_, err = SearchFiles(logger, "../../../test", myFiles, badExpression)
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

// TODO: Add more test cases (more result validation, more expression variants, negative cases, etc...)
