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

type testCase struct {
	Function       func(log *zap.SugaredLogger, clusterRoot string, issueReporter *report.IssueReporter) error
	ClusterRoot    string
	ExpectedIssues int
}

var testCases = []testCase{
	{
		Function:       rancher.AnalyzeManagementClusters,
		ClusterRoot:    "../../../test/cluster/clusters/clusters-ready/cluster-snapshot",
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeManagementClusters,
		ClusterRoot:    "../../../test/cluster/clusters/clusters-not-ready/cluster-snapshot",
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeClusterRepos,
		ClusterRoot:    "../../../test/cluster/clusters/clusters-ready/cluster-snapshot",
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeClusterRepos,
		ClusterRoot:    "../../../test/cluster/clusters/clusters-not-ready/cluster-snapshot",
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeCatalogs,
		ClusterRoot:    "../../../test/cluster/clusters/clusters-ready/cluster-snapshot",
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeCatalogs,
		ClusterRoot:    "../../../test/cluster/clusters/clusters-not-ready/cluster-snapshot",
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeProvisioningClusters,
		ClusterRoot:    "../../../test/cluster/clusters/clusters-ready/cluster-snapshot",
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeProvisioningClusters,
		ClusterRoot:    "../../../test/cluster/clusters/clusters-not-ready/cluster-snapshot",
		ExpectedIssues: 1,
	},
}

// Test analyze Rancher resources with different cluster snapshots.
func TestAnalyzeRancher(t *testing.T) {
	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}
	logger := zap.S()

	for _, test := range testCases {
		report.ClearReports()
		assert.NoError(t, test.Function(logger, test.ClusterRoot, &issueReporter))
		reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
		if test.ExpectedIssues == 0 {
			assert.Empty(t, reportedIssues)
		} else {
			assert.Len(t, reportedIssues, test.ExpectedIssues)
			assert.Equal(t, "RancherIssues", reportedIssues[0].Type)
		}
	}
}
