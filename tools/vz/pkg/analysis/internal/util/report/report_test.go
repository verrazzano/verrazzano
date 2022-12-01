// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package report

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	help "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"os"
	"strings"
	"testing"
)

// TestInvalidIssues Tests the helpers with invalid issues
// GIVEN a call to helper
// WHEN with an invalid issue
// THEN the appropriate error is returned
// WHEN with a valid issue
// THEN no error is returned
func TestInvalidIssues(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	// We start with a custom issue which is created without being populated. This is
	// invalid as there are a few fields which are required
	var invalidIssue = Issue{}
	invalidIssue.Informational = true
	// This will fail as there is no Type specified
	err := ContributeIssue(logger, invalidIssue)
	assert.NotNil(t, err)
	logger.Debugf("Err was", err)
	assert.True(t, strings.Contains(err.Error(), "Type"))

	// Next set the Type on the issue, it should then complain that no Source is specified
	invalidIssue.Type = "MyIssueType"
	err = ContributeIssue(logger, invalidIssue)
	assert.NotNil(t, err)
	logger.Debugf("Err was", err)
	assert.True(t, strings.Contains(err.Error(), "Source"))

	// Next set the Source on the issue, it should then complain that no Summary is specified
	invalidIssue.Source = "MyIssueSource"
	err = ContributeIssue(logger, invalidIssue)
	assert.NotNil(t, err)
	logger.Debugf("Err was", err)
	assert.True(t, strings.Contains(err.Error(), "Summary"))

	// Next set the summary but also set a confidence to a value out of range
	invalidIssue.Summary = "MyIssueSummary"
	invalidIssue.Confidence = 11
	err = ContributeIssue(logger, invalidIssue)
	assert.NotNil(t, err)
	logger.Debugf("Err was", err)
	assert.True(t, strings.Contains(err.Error(), "Confidence"))

	// to get no issues, set the actions and confidence to a value in range
	invalidIssue.Actions = []Action{{Summary: invalidIssue.Summary}}
	invalidIssue.Confidence = 8
	assert.Nil(t, ContributeIssue(logger, invalidIssue))

	// to get no issues from contribute issues map
	assert.Nil(t, ContributeIssuesMap(logger, "MyIssueSource", map[string]Issue{"issue": invalidIssue}))
	assert.Empty(t, GetAllSourcesFilteredIssues(logger, true, 8, 11))
	AddSourceAnalyzed(invalidIssue.Source)

	// Send stdout stderr to a byte buffer
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rc := help.NewFakeRootCmdContext(genericclioptions.IOStreams{In: os.Stdin, Out: buf, ErrOut: errBuf})
	assert.NoError(t, GenerateHumanReport(logger, "report", constants.SummaryReport, true, true, true, 11, 11, rc))
}

// We start with a custom issue which is created without being populated. This is
// valid as there are a few fields which are required
// this is mostly concerned for testing the de duplicates of issues
// issue 2 and 3 are same and unlike to issue 1
// filter issues must have 2 issues post filter

func TestFilterReportIssues(t *testing.T) {
	logger := log.GetDebugEnabledLogger()

	var validIssues = make([]Issue, 3)
	validIssues[0].Type = "ISSUE 1"
	validIssues[0].Summary = "Verrazzano install failed as no IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer\n\tConsult https://verrazzano.io/v1.5/docs/troubleshooting/diagnostictools/analysisadvice/ingressnoloadbalancerip using supporting details identified in the report"
	validIssues[0].Actions = []Action{{Summary: "Test Summary 1", Links: []string{"l1", "l2"}, Steps: []string{"s1", "s2"}}}
	validIssues[0].Source = "Test Source 1"
	validIssues[0].Informational = false
	validIssues[0].Impact = 10
	validIssues[0].Confidence = 10

	validIssues[1].Type = "ISSUE 2"
	validIssues[1].Summary = "Verrazzano install failed as no IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer\n\tConsult https://verrazzano.io/v1.5/docs/troubleshooting/diagnostictools/analysisadvice/ingressnoloadbalancerip using supporting details identified in the report"
	validIssues[1].Actions = []Action{{Summary: "Test Summary 2", Links: []string{"l1", "l2"}, Steps: []string{"s1", "s2"}}}
	validIssues[1].Source = "Test Source 2"
	validIssues[1].Informational = false
	validIssues[1].Impact = 10
	validIssues[1].Confidence = 10

	validIssues[2].Type = "ISSUE 2"
	validIssues[2].Summary = "Verrazzano install failed as no IP found for service ingress-controller-ingress-nginx-controller with type LoadBalancer\n\tConsult https://verrazzano.io/v1.5/docs/troubleshooting/diagnostictools/analysisadvice/ingressnoloadbalancerip using supporting details identified in the report"
	validIssues[2].Actions = []Action{{Summary: "Test Summary 2", Links: []string{"l1", "l2"}, Steps: []string{"s1", "s2"}}}
	validIssues[2].Source = "Test Source 2"
	validIssues[2].Informational = false
	validIssues[2].Impact = 10
	validIssues[2].Confidence = 10

	filteredIssue := filterReportIssues(logger, validIssues, false, 7, 8)
	assert.Len(t, filteredIssue, 2, "duplicate issues must be filtered out")
}
