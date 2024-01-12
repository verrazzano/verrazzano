// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
)

// TestAnalyzeNetworkingIssues tests whether an error does not occur if a valid input is provided and if an error occurs if a valid input is provided
// GIVEN a call to analyze network related issues in a cluster-snapshot
// WHEN a valid input is provided
// THEN the function does not generate an error
func TestAnalyzeNetworkingIssues(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzeNetworkingIssues(logger, "../../../test/cluster/testTCPKeepIdle/cluster-snapshot"))
	report.ClearReports()

}

// TestDetermineIfTCPKeepIdleIssueHasOccurred tests that an issue is reported when a cluster-snapshot has a TCP Keep Idle issue in the istio-system namespace
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster snapshot has a TCP Keep Idle issue in the istio-system namespace
// Then a TCP Keep Idle Issue should be reported and no errors should be raised
func TestDetermineIfTCPKeepIdleIssueHasOccurred(t *testing.T) {
	// This test confirms that it detects the issue when it appears in a log
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	err := AnalyzeNetworkingIssues(logger, "../../../test/cluster/testTCPKeepIdle/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.True(t, len(reportedIssues) == 1)
	// This test tests that this issue is not reported when it does not occur
	report.ClearReports()
	err = AnalyzeNetworkingIssues(logger, "../../../test/cluster/testTCPKeepIdleNotOccuring/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues = report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.True(t, len(reportedIssues) == 0)
	// This test tests that when this issue occurs multiple times in multiple files, only one issue is reported
	// This test also confirms that the informational data in the report contains the names of all files where this single issue occurred
	report.ClearReports()
	err = AnalyzeNetworkingIssues(logger, "../../../test/cluster/testTCPKeepIdleIssueOccursInTwoPods/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues = report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.True(t, len(reportedIssues) == 1)
	messageForSupportingDataForIssue := reportedIssues[0].SupportingData[0].Messages[0]
	assert.True(t, strings.Contains(messageForSupportingDataForIssue, "cluster-snapshot/istio-system/testIstioPodNameTwo/logs.txt"))
	assert.True(t, strings.Contains(messageForSupportingDataForIssue, "cluster-snapshot/istio-system/testIstioPodName/logs.txt"))
}
