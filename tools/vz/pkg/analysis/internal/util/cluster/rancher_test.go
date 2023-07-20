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
	Function       func(clusterRoot string, issueReporter *report.IssueReporter) error
	ClusterRoot    string
	ExpectedIssues int
}

const (
	clustersReadySnapshot              = "../../../test/cluster/rancher/clusters-ready/cluster-snapshot"
	clustersNotReadySnapshot           = "../../../test/cluster/rancher/clusters-not-ready/cluster-snapshot"
	clustersReadySnapshotNamespaced    = "../../../test/cluster/rancher/clusters-ready/cluster-snapshot/namespaced"
	clustersNotReadySnapshotNamespaced = "../../../test/cluster/rancher/clusters-not-ready/cluster-snapshot/namespaced"
)

var testCases = []testCase{
	{
		Function:       rancher.AnalyzeManagementClusters,
		ClusterRoot:    clustersReadySnapshot,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeManagementClusters,
		ClusterRoot:    clustersNotReadySnapshot,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeClusterRepos,
		ClusterRoot:    clustersReadySnapshot,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeClusterRepos,
		ClusterRoot:    clustersNotReadySnapshot,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeCatalogs,
		ClusterRoot:    clustersReadySnapshot,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeCatalogs,
		ClusterRoot:    clustersNotReadySnapshot,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeProvisioningClusters,
		ClusterRoot:    clustersReadySnapshotNamespaced,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeProvisioningClusters,
		ClusterRoot:    clustersNotReadySnapshotNamespaced,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeBundleDeployments,
		ClusterRoot:    clustersReadySnapshotNamespaced,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeBundleDeployments,
		ClusterRoot:    clustersNotReadySnapshotNamespaced,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeBundles,
		ClusterRoot:    clustersReadySnapshotNamespaced,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeBundles,
		ClusterRoot:    clustersNotReadySnapshotNamespaced,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeClusterGroups,
		ClusterRoot:    clustersReadySnapshotNamespaced,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeClusterGroups,
		ClusterRoot:    clustersNotReadySnapshotNamespaced,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeClusterRegistrations,
		ClusterRoot:    clustersReadySnapshotNamespaced,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeClusterRegistrations,
		ClusterRoot:    clustersNotReadySnapshotNamespaced,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeFleetClusters,
		ClusterRoot:    clustersReadySnapshotNamespaced,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeFleetClusters,
		ClusterRoot:    clustersNotReadySnapshotNamespaced,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeCatalogApps,
		ClusterRoot:    clustersReadySnapshotNamespaced,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeCatalogApps,
		ClusterRoot:    clustersNotReadySnapshotNamespaced,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeKontainerDrivers,
		ClusterRoot:    clustersReadySnapshot,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeKontainerDrivers,
		ClusterRoot:    clustersNotReadySnapshot,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeGitRepos,
		ClusterRoot:    clustersReadySnapshotNamespaced,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeGitRepos,
		ClusterRoot:    clustersNotReadySnapshotNamespaced,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeNodes,
		ClusterRoot:    clustersReadySnapshotNamespaced,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeNodes,
		ClusterRoot:    clustersNotReadySnapshotNamespaced,
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
		assert.NoError(t, test.Function(test.ClusterRoot, &issueReporter))
		issueReporter.Contribute(logger, test.ClusterRoot)
		reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
		if test.ExpectedIssues == 0 {
			assert.Empty(t, reportedIssues)
		} else {
			assert.Len(t, reportedIssues, test.ExpectedIssues)
			if len(reportedIssues) != 0 {
				assert.Equal(t, "RancherIssues", reportedIssues[0].Type)
			}
		}
	}
}
