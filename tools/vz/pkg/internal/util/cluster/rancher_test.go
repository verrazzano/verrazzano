// Copyright (c) 2023, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/cluster/rancher"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
	"go.uber.org/zap"
)

type testCase struct {
	Function       func(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error
	ClusterRoot    string
	Namespaced     bool
	ExpectedIssues int
}

const (
	clustersReadySnapshot    = "../../test/cluster/rancher/clusters-ready/cluster-snapshot"
	clustersNotReadySnapshot = "../../test/cluster/rancher/clusters-not-ready/cluster-snapshot"
)

var testCases = []testCase{
	{
		Function:       rancher.AnalyzeManagementClusters,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     false,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeManagementClusters,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     false,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeClusterRepos,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     false,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeClusterRepos,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     false,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeCatalogs,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     false,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeCatalogs,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     false,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeProvisioningClusters,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeProvisioningClusters,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeBundleDeployments,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeBundleDeployments,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeBundles,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeBundles,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeClusterGroups,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeClusterGroups,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeClusterRegistrations,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeClusterRegistrations,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeFleetClusters,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeFleetClusters,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeCatalogApps,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeCatalogApps,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeKontainerDrivers,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     false,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeKontainerDrivers,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     false,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeGitRepos,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeGitRepos,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeGitJobs,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeGitJobs,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeNodes,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeNodes,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       rancher.AnalyzeManagedCharts,
		ClusterRoot:    clustersReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       rancher.AnalyzeManagedCharts,
		ClusterRoot:    clustersNotReadySnapshot,
		Namespaced:     true,
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
		namespace := ""
		if test.Namespaced {
			namespace = "namespaced"
		}
		assert.NoError(t, test.Function(test.ClusterRoot, namespace, &issueReporter))
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
