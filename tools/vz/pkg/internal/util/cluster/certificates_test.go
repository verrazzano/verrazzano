// Copyright (c) 2023, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package cluster

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
)

// TestAnalyzeCertificateIssues tests whether an error does not occur if a valid input is provided and if an error occurs if a valid input is provided
// GIVEN a call to analyze certificate related issues in a cluster-snapshot
// WHEN a valid input or an invalid input is provided
// THEN an error is invoked when an invalid input is provided and invoked when a valid input is provided
func TestAnalyzeCertificateIssues(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	assert.NoError(t, AnalyzeCertificateRelatedIssues(logger, "../../test/cluster/testCertificateExpirationIssue/cluster-snapshot"))
	assert.Error(t, AnalyzeCertificateRelatedIssues(logger, "../../test/cluster/testCertificateExpirationIssueInvalid/cluster-snapshot"))
	report.ClearReports()

}

// TestDetermineIfVZClientIsHangingDueToCerts tests whether the function is able to detect the VZ Client hanging on certificate-related issues
// GIVEN a call to see if the VZ Client is currently hanging
// WHEN VPO logs indicate that the VZ Client is hanging, or VPO logs do not indicate than the VZ Client is hanging
// THEN an appropriate output is provided depending on the VPO logs
func TestDetermineIfVZClientIsHangingDueToCerts(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	listOfCerts, err := determineIfVZClientIsHangingDueToCerts(logger, "../../test/cluster/testCLIHangingIssue/cluster-snapshot")
	assert.Equal(t, err, nil)
	assert.Greater(t, len(listOfCerts), 0)
	listOfCerts, err = determineIfVZClientIsHangingDueToCerts(logger, "../../test/cluster/testCertificateExpirationIssue/cluster-snapshot")
	assert.Equal(t, err, nil)
	assert.Equal(t, len(listOfCerts), 0)
}

// TestGetLatestCondition tests whether the certificate condition receives the latest condition and if the function ignores conditions if they do not have a timestamp
// GIVEN a call to analyze a cluster-snapshot and report if issues with certificates exist
// WHEN a condition in a certificate's conditionlist does not have a timestamp
// THEN it is ignored
func TestGetLatestCondition(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	// In this example, two certificates have generic issues, but only one of the certificates has a condition time associated with its condition that reports this
	err := AnalyzeCertificateRelatedIssues(logger, "../../test/cluster/testLatestCondition/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.NotNil(t, reportedIssues)
	assert.True(t, len(reportedIssues) > 0)
	problemsFound := 0
	for _, issue := range reportedIssues {
		if issue.Type == report.CertificateExperiencingIssuesInCluster {
			problemsFound++
		}
	}
	assert.True(t, problemsFound == 1)
}

// TestNoIssuesFoundInCertificates tests that no issues are reported when a cluster-snapshot with no certificate issues is given to the function
// GIVEN a call to analyze cluster-snapshots
// WHEN these cluster-snapshots either have no certificate issues or a cluster-snapshot that does not possess any certificates.json files
// THEN no issues should be reported and no errors should be raised
func TestNoIssuesFoundInCertificates(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	err := AnalyzeCertificateRelatedIssues(logger, "../../test/cluster/testValidCertificates/cluster-snapshot")
	assert.Nil(t, err)
	err = AnalyzeCertificateRelatedIssues(logger, "../../test/cluster/testNoCertificates/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.Nil(t, reportedIssues)
	assert.True(t, len(reportedIssues) == 0)
}

// TestCertificatesAreNotGrantedReturnsNoError tests that an issue is reported when a cluster-snapshot has certificates that are waiting to be issued
// GIVEN a call to analyze a cluster-snapshot
// WHEN the cluster snapshot has certificates that are hanging/not yet granted
// Then a generic certificate issue should be reported and no errors should be raised
func TestCertificatesAreNotGrantedReturnsNoError(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	err := AnalyzeCertificateRelatedIssues(logger, "../../test/cluster/testCertificatesNotGranted/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.True(t, len(reportedIssues) == 1)
}

func TestCaCertInfoFileWithNoIssueReturnsNoError(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	err := AnalyzeCertificateRelatedIssues(logger, "../../test/cluster/testCaCertsNotExpired/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.True(t, len(reportedIssues) == 0)
}

func TestCaCertInfoFileWithExpirationReportsAnIssue(t *testing.T) {
	report.ClearReports()
	logger := log.GetDebugEnabledLogger()
	err := AnalyzeCertificateRelatedIssues(logger, "../../test/cluster/testCaCertsExpired/cluster-snapshot")
	assert.Nil(t, err)
	reportedIssues := report.GetAllSourcesFilteredIssues(logger, true, 0, 0)
	assert.True(t, len(reportedIssues) == 1)
}
