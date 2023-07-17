// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/cluster/rancher"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
)

// Test analyze Rancher resources with different cluster snapshots.
func TestAnalyzeRancher(t *testing.T) {
	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}
	logger := zap.S()

	// Expect no errors and no reported issues.
	report.ClearReports()
	assert.NoError(t, rancher.AnalyzeRancherClusters(logger, "../../../test/cluster/clusters/clusters-ready/cluster-snapshot", &issueReporter))
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.Empty(t, reportedIssues)

	// Expect no errors and one reported issue that a Rancher Cluster is not ready.
	report.ClearReports()
	assert.NoError(t, rancher.AnalyzeRancherClusters(logger, "../../../test/cluster/clusters/clusters-not-ready/cluster-snapshot", &issueReporter))
	reportedIssues = report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.Len(t, reportedIssues, 1)
	assert.Equal(t, "RancherIssues", reportedIssues[0].Type)
}
