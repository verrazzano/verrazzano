// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"testing"
)

// TestAnalyzeNamespaceRelatedIssueWhenNamespaceAndMetadataNotPresent tests whether an error does not occur if a valid input is provided
// GIVEN a call to analyze namespace related issues in a cluster-snapshot
// WHEN a valid input is provided, but namespace and time capture data is not present
// THEN the function does not generate an error
func TestAnalyzeNamespaceRelatedIssueWhenNamespaceAndMetadataNotPresent(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzeNamespaceRelatedIssues(logger, "../../../test/cluster/testTCPKeepIdle/cluster-snapshot"))
	report.ClearReports()
}

// TestDetermineIfNamespaceTerminationIssueHasOccurred tests whether the relevant issue is reported when a namespace is stuck in a terminating state
// GIVEN a call to analyze namespace related issues in a cluster-snapshot
// WHEN a valid input is provided, which indicates that a namespace is stuck terminating
// THEN the function does not generate an error and reports an issue
func TestDetermineIfNamespaceTerminationIssueHasOccurred(t *testing.T) {
	// This test confirms that it detects the issue when it appears in a log
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	err := AnalyzeNamespaceRelatedIssues(logger, "../../../test/cluster/namespace-stuck-terminating-on-finalizers/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.True(t, len(reportedIssues) == 1)
	// This test tests that this issue is not reported when it does not occur
	report.ClearReports()
}
