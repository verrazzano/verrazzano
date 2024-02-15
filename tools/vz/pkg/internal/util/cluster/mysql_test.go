// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
)

// TestAnalyzeMySQLRelatedIssueWhenNamespaceAndMetadataNotPresent tests whether an error does not occur if a valid input is provided
// GIVEN a call to analyze mySQL related issues in a cluster-snapshot
// WHEN a valid input is provided, but the innoDBCluster resource  and time capture data is not present
// THEN the function does not generate an error
func TestAnalyzeMySQLRelatedIssueWhenNamespaceAndMetadataNotPresent(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzeMySQLRelatedIssues(logger, "../../test/cluster/testTCPKeepIdle/cluster-snapshot"))
	report.ClearReports()
}

// TestAnalyzeMySQLRelatedIssueWhenInputIsNotVali tests whether an error occurs when an invalid input is provided
// GIVEN a call to analyze MySQL related issues in a cluster-snapshot
// WHEN an invalid input is provided
// THEN the function does not generate an error
func TestAnalyzeMySQLRelatedIssueWhenInputIsNotValid(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	assert.Error(t, AnalyzeMySQLRelatedIssues(logger, "../../test/cluster/does-not-exist/cluster-snapshot"))
	report.ClearReports()
}

// TestAnalyzeMySQLRelatedIssuesWhenMetadataFileIsNotProvided tests whether only one issue is created and no error is generated when a valid input without a metadata.json file and an innoDbCluster resource is provided
// GIVEN a call to analyze MySQL related issues in a cluster-snapshot
// WHEN a valid input is provided that has an innoDBCluster resource with issues and does not have a metadata.json file
// THEN the function does not generate an error and does not create an issue
func TestAnalyzeMySQLRelatedIssuesWhenMetadataFileIsNotProvided(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	err := AnalyzeMySQLRelatedIssues(logger, "../../test/cluster/inno-db-cluster-stuck-terminating-no-metadata-file/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 5, 0)
	assert.True(t, len(reportedIssues) == 0)
}

// TestAnalyzeMySQLRelatedIssuesWhenMetadataFileIsNotProvided tests whether only one issue is created and no error is generated when a valid input when a metadata.json file and an inno-db-cluster.json file is provided
// GIVEN a call to analyze MySQL related issues in a cluster-snapshot
// WHEN a valid input is provided that has an innoDBCluster resource with issues and has a metadata.json file
// THEN the function does not generate an error and only creates one issue
func TestAnalyzeMySQLRelatedIssues(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	err := AnalyzeMySQLRelatedIssues(logger, "../../test/cluster/inno-db-cluster-stuck-terminating-with-metadata-file/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 5, 0)
	assert.True(t, len(reportedIssues) == 1)
	assert.True(t, reportedIssues[0].Type == report.InnoDBClusterResourceCurrentlyInTerminatingStateForLongDuration)
}
