// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package cluster

import (
	"github.com/stretchr/testify/assert"
	capi "github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/cluster/clusterapi"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
	"testing"
)

const (
	clusterAPIReadySnapshotNamespaced     = "../../../test/cluster/cluster-api/clusters-ready/cluster-snapshot/namespaced"
	clustersAPINotReadySnapshotNamespaced = "../../../test/cluster/cluster-api/clusters-not-ready/cluster-snapshot/namespaced"
)

type capiTestCase struct {
	Function       func(clusterRoot string, issueReporter *report.IssueReporter) error
	ClusterRoot    string
	ExpectedIssues int
}

var capiTestCases = []capiTestCase{
	{
		Function:       capi.AnalyzeClusters,
		ClusterRoot:    clusterAPIReadySnapshotNamespaced,
		ExpectedIssues: 0,
	},
	{
		Function:       capi.AnalyzeClusters,
		ClusterRoot:    clustersAPINotReadySnapshotNamespaced,
		ExpectedIssues: 1,
	},
}

// Test analyze Cluster API resources with different cluster snapshots.
func TestAnalyzeClusterAPI(t *testing.T) {
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
