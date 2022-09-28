// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package report

import (
	"github.com/stretchr/testify/assert"
	utilfiles "github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/files"
	"testing"
)

const test_msg = "test message"

// TestHandlingUnknownIssues Tests unknown issue types are handled correctly when using the known issue helpers
// GIVEN a call to *KnownIssue* helper
// WHEN with an unknown issue type
// THEN the call panic's (this is catching a coding error)
func TestHandlingUnknownIssues(t *testing.T) {
	var issueReporter = IssueReporter{
		PendingIssues: make(map[string]Issue),
	}
	rootDirectory := "test root directory"
	messages := []string{test_msg}
	files := []string{"test file name"}
	matches := make([]utilfiles.TextMatch, 1)
	matches[0] = utilfiles.TextMatch{
		FileName:    "test file",
		FileLine:    50,
		MatchedText: "test matched text",
	}
	supportingData := make([]SupportData, 1)
	supportingData[0] = SupportData{
		Messages:    messages,
		TextMatches: matches,
	}
	assert.Panics(t, func() { issueReporter.AddKnownIssueMessagesFiles("BADISSUETYPE", rootDirectory, messages, files) })
	assert.Panics(t, func() { issueReporter.AddKnownIssueSupportingData("BADISSUETYPE", rootDirectory, supportingData) })
	assert.Panics(t, func() { issueReporter.AddKnownIssueMessagesMatches("BADISSUETYPE", rootDirectory, messages, matches) })
	assert.Panics(t, func() { NewKnownIssueMessagesFiles("BADISSUETYPE", rootDirectory, messages, files) })
	assert.Panics(t, func() { NewKnownIssueSupportingData("BADISSUETYPE", rootDirectory, supportingData) })
	assert.Panics(t, func() { NewKnownIssueMessagesMatches("BADISSUETYPE", rootDirectory, messages, matches) })
}

// TestHandlingKnownIssues Tests the known issue helpers
// GIVEN a call to *KnownIssue* helper
// WHEN with a known issue type
// THEN the issue is successfully added or created
func TestHandlingKnownIssues(t *testing.T) {
	var issueReporter = IssueReporter{
		PendingIssues: make(map[string]Issue),
	}
	rootDirectory := "test root directory"
	messages := []string{test_msg}
	files := []string{"test file name"}
	matches := make([]utilfiles.TextMatch, 1)
	matches[0] = utilfiles.TextMatch{
		FileName:    "test file",
		FileLine:    50,
		MatchedText: "test matched text",
	}
	supportingData := make([]SupportData, 1)
	supportingData[0] = SupportData{
		Messages:    messages,
		TextMatches: matches,
	}
	issueA := NewKnownIssueMessagesFiles(ImagePullBackOff, rootDirectory, messages, files)
	assert.NotNil(t, issueA)
	issueB := NewKnownIssueSupportingData(InsufficientMemory, rootDirectory, supportingData)
	assert.NotNil(t, issueB)
	issueC := NewKnownIssueMessagesMatches(PodProblemsNotReported, rootDirectory, messages, matches)
	assert.NotNil(t, issueC)
	issueReporter.AddKnownIssueMessagesFiles(ImagePullBackOff, rootDirectory, messages, files)
	issueReporter.AddKnownIssueSupportingData(InsufficientMemory, rootDirectory, supportingData)
	issueReporter.AddKnownIssueMessagesMatches(PodProblemsNotReported, rootDirectory, messages, matches)
}

// TestMiscHelpers tests misc helpers
func TestMiscHelpers(t *testing.T) {
	messages := SingleMessage(test_msg)
	assert.NotNil(t, messages)
	assert.True(t, len(messages) == 1)
}
