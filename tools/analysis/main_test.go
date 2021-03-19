// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/report"
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
		if issue.Type == report.ImagePullBackOff {
			imagePullsFound++
		}
	}
	assert.True(t, imagePullsFound > 0)
}

// TestInsufficientMemory Tests that analysis of a cluster dump with pods that failed due to insufficient memory
// GIVEN a call to analyze a cluster-dump
// WHEN the cluster-dump shows pods with insufficient memory problems
// THEN a report is generated with issues identified
func TestInsufficientMemory(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/insufficient-mem")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	issuesFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.InsufficientMemory {
			issuesFound++
		}
	}
	assert.True(t, issuesFound > 0)
}

// TestProblemPodsNotReported Tests that analysis of a cluster dump with pods that have unknown issues is handled
// GIVEN a call to analyze a cluster-dump
// WHEN the cluster-dump shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestProblemPodsNotReported(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/problem-pods")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemPodsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.PodProblemsNotReported {
			problemPodsFound++
		}
	}
	assert.True(t, problemPodsFound > 0)
}
