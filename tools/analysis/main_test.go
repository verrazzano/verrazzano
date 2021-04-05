// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/report"
	"testing"
)

// TestHandleMain Tests the handleMain function
// GIVEN a call to handleMain
// WHEN with valid/invalid inputs
// THEN exit codes returned are as expected
func TestHandleMain(t *testing.T) {
	// This is setting up the main.logger, do NOT set it as a var here (or you will get a nil reference running
	// the test)
	logger = log.GetDebugEnabledLogger()

	// Calling handleMain without any flags/args set will print usage and return 1 exit code
	flagArgs = make([]string, 0)
	exitCode := handleMain()
	assert.True(t, exitCode == 1)

	// Calling handleMain with help=true will print usage and return 0 exit code
	help = true
	exitCode = handleMain()
	assert.True(t, exitCode == 0)
	help = false

	// Calling handleMain with a valid cluster root path will analyze and return 0 exit code
	flagArgs = make([]string, 1)
	flagArgs[0] = "test/cluster/image-pull-case1"
	analyzerType = "cluster"
	exitCode = handleMain()
	assert.True(t, exitCode == 0)

	// Calling handleMain with a valid cluster root path and unknown analyzer type will print usage
	// and return 1 exit code
	analyzerType = "BadAnalyzerType"
	exitCode = handleMain()
	assert.True(t, exitCode == 1)

	// Calling handleMain with a valid cluster root path and bad minConfidence will print usage
	// and return 1 exit code
	analyzerType = "cluster"
	minConfidence = -1
	exitCode = handleMain()
	assert.True(t, exitCode == 1)

	// Calling handleMain with a valid cluster root path and bad minConfidence will print usage
	// and return 1 exit code
	minConfidence = 11
	exitCode = handleMain()
	assert.True(t, exitCode == 1)

	// Calling handleMain with a valid cluster root path and bad minImpact will print usage
	// and return 1 exit code
	minImpact = -1
	exitCode = handleMain()
	assert.True(t, exitCode == 1)

	// Calling handleMain with a valid cluster root path and bad minImpact will print usage
	// and return 1 exit code
	minImpact = 11
	exitCode = handleMain()
	assert.True(t, exitCode == 1)

	minImpact = 0
	minConfidence = 0
	analyzerType = "cluster"
	exitCode = handleMain()
	assert.True(t, exitCode == 0)
}

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
		if issue.Type == report.ImagePullNotFound {
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

// TestLBIpNotSet Tests that analysis of a cluster dump where LB issue occurred with no IP set is handled
// GIVEN a call to analyze a cluster-dump
// WHEN the cluster-dump shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestLBIpNotSet(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/lb-ipnotset")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.IngressNoLoadBalancerIP {
			problemsFound++
		}
	}
	assert.True(t, problemsFound > 0)
}

// TestIngressInstall Tests that analysis of a cluster dump where Ingress install failed without more info handled
// GIVEN a call to analyze a cluster-dump
// WHEN the cluster-dump shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestIngressInstall(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/ingress-install-unknown")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.IngressInstallFailure {
			problemsFound++
		}
	}
	assert.True(t, problemsFound > 0)
}

// TestOciIPLimitExceeded Tests that analysis of a cluster dump where Ingress install failed due to OCI limit handled
// GIVEN a call to analyze a cluster-dump
// WHEN the cluster-dump shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestOciIPLimitExceeded(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/ingress-oci-limit")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.IngressOciIPLimitExceeded {
			problemsFound++
		}
	}
	assert.True(t, problemsFound > 0)
}

// TestPendingPods that analysis of a cluster dump where pending pods only is handled
// GIVEN a call to analyze a cluster-dump
// WHEN the cluster-dump shows pods with problems that are not known issues
// THEN a report is generated with problem pod issues identified
func TestPendingPods(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	err := Analyze(logger, "cluster", "test/cluster/pending-pods")
	assert.Nil(t, err)

	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.PendingPods {
			problemsFound++
		}
	}
	assert.True(t, problemsFound > 0)
}
