// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
)

// TestAnalyzeCertificateIssues tests whether an error does not occur if a valid input is provided and if an error occurs if a valid input is provided
// GIVEN a call to analyze certificate related issues in a cluster-snapshot
// WHEN a valid input or an invalid input is provided
// THEN an error is invoked when an invalid input is provided and invoked when a valid input is provided
func TestAnalyzeNetworkingIssues(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzeNetworkingIssues(logger, "../../../test/cluster/testTCPKeepIdle/cluster-snapshot"))
	report.ClearReports()

}
func TestDetermineIfTCPKeepIdleIssueHasOccurred(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	err := AnalyzeNetworkingIssues(logger, "../../../test/cluster/testTCPKeepIdle/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.True(t, len(reportedIssues) == 1)
}
