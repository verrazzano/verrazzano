// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package cluster handles cluster analysis
package cluster

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"testing"
)

// TestAnalyzeNetworkingIssues tests whether an error does not occur if a valid input is provided and if an error occurs if a valid input is provided
// GIVEN a call to analyze network related issues in a cluster-snapshot
// WHEN a valid input is provided
// THEN the function does not generate an error
func TestAnalyzeNamespaceRelatedIssues(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzeNetworkingIssues(logger, "../../../test/cluster/testTCPKeepIdle/cluster-snapshot"))
	report.ClearReports()

}
