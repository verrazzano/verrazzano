// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	capi "github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/cluster/clusterapi"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
	"go.uber.org/zap"
)

const (
	clusterAPIReadySnapshot     = "../../../test/cluster/cluster-api/clusters-ready/cluster-snapshot"
	clustersAPINotReadySnapshot = "../../../test/cluster/cluster-api/clusters-not-ready/cluster-snapshot"
)

type capiTestCase struct {
	Function       func(clusterRoot string, namespace string, issueReporter *report.IssueReporter) error
	ClusterRoot    string
	Namespaced     bool
	ExpectedIssues int
}

var capiTestCases = []capiTestCase{
	{
		Function:       capi.AnalyzeClusters,
		ClusterRoot:    clusterAPIReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       capi.AnalyzeClusters,
		ClusterRoot:    clustersAPINotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       capi.AnalyzeOCIClusters,
		ClusterRoot:    clusterAPIReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       capi.AnalyzeOCIClusters,
		ClusterRoot:    clustersAPINotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       capi.AnalyzeOCNEControlPlanes,
		ClusterRoot:    clusterAPIReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       capi.AnalyzeOCNEControlPlanes,
		ClusterRoot:    clustersAPINotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       capi.AnalyzeMachines,
		ClusterRoot:    clusterAPIReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       capi.AnalyzeMachines,
		ClusterRoot:    clustersAPINotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       capi.AnalyzeMachineDeployments,
		ClusterRoot:    clusterAPIReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       capi.AnalyzeMachineDeployments,
		ClusterRoot:    clustersAPINotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       capi.AnalyzeOCNEConfigs,
		ClusterRoot:    clusterAPIReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       capi.AnalyzeOCNEConfigs,
		ClusterRoot:    clustersAPINotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       capi.AnalyzeOCIMachines,
		ClusterRoot:    clusterAPIReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       capi.AnalyzeOCIMachines,
		ClusterRoot:    clustersAPINotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       capi.AnalyzeMachineSets,
		ClusterRoot:    clusterAPIReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       capi.AnalyzeMachineSets,
		ClusterRoot:    clustersAPINotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
	{
		Function:       capi.AnalyzeClusterResourceSets,
		ClusterRoot:    clusterAPIReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 0,
	},
	{
		Function:       capi.AnalyzeClusterResourceSets,
		ClusterRoot:    clustersAPINotReadySnapshot,
		Namespaced:     true,
		ExpectedIssues: 1,
	},
}

// Test analyze Cluster API resources with different cluster snapshots.
func TestAnalyzeClusterAPI(t *testing.T) {
	var issueReporter = report.IssueReporter{
		PendingIssues: make(map[string]report.Issue),
	}
	logger := zap.S()

	for _, test := range capiTestCases {
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
				assert.Equal(t, "ClusterAPIClusterIssues", reportedIssues[0].Type)
			}
		}
	}
}
