// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/report"
	"strings"
	"testing"
)


// TestAnalyzeBad Tests the main Analyze function
// GIVEN a call to Analyze
// WHEN with invalid inputs
// THEN errors are generated as expected
func TestExecuteAnalysisBadArgs(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	// Call the analyzer with an unknown type, give it a good cluster dump directory
	err := Analyze(logger, "badnamehere", "../test/cluster/image-pull-case1")
	assert.NotNil(t, err)
	// TODO: Check error message is what we expected here

}

// TestImagePullCase1 Tests that analysis of a cluster dump with image pull issues is handled
// GIVEN a call to analyze a cluster-dump
// WHEN the cluster-dump shows image pull issues
// THEN a report is generated with image pull issues identified
func TestImagePull(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/image-pull-case1")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	imagePullsFound := 0
	for _, issue := range reportedIssues {
		if strings.Contains(issue.Summary, report.ImagePullBackOff) {
			imagePullsFound++
		}
	}
	assert.True(t, imagePullsFound > 0)
}

// Add insufficient memory test

// Add Problem pods found test
