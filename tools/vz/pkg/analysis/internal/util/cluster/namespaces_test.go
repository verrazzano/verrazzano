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
