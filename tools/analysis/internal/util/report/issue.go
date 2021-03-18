// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package report handles reporting
package report

import (
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/files"
	"go.uber.org/zap"
)

// NOTE: This is part of the contract with the analyzers however it is currently an initial stake in the ground and
//		 will be evolving rapidly initially as we add analysis cases

// An issue describes a specific problem that has been found and includes information such as
//     A Summary of the issue
//     A list of Actions which can be taken
//         - Actions are reported in the order specified in this list (so actions more likely to mitigate an issue
//         should be specified first).
//         - Each action may have Steps to take and/or give a list of runbook Links
//     A list of supporting data (TBD)
//         - Source which helped identify the issue
//         - Indicators that identified the issue (search matches, json elements)
//         - etc...
//     A Confidence level (TBD)
//         This is and indication of how confident the analysis is that the issue is really causing
//         problems. The analysis will attempt to weed out things that are not causing an issue and will
//         not report them if it is certain. However there may be situations where something that is found
//         could be causing problems but it is not certain.

// JSONPath is a JSON path
type JSONPath struct {
	File string // Json filename
	Path string // Json Path
}

// SupportData is data which helps a user to further identify an issue TODO: Shake this out more as we add more types, see what we really end up needing here
type SupportData struct {
	Messages     []string          // Optional, Messages and/or descriptions the supporting data
	RelatedFiles []string          // Optional, if present provides a list of related files that support the issue identification
	TextMatches  []files.TextMatch // Optional, if present provides search results that support the issue identification
	JSONPaths    []JSONPath        // Optional, if present provides a list of Json paths that support the issue identification
}

// Issue holds the information about an issue, supporting data, and actions
type Issue struct {
	Type          string   // Required, This identifies the type of issue. This is either a Known Issue type, or a custom type name
	Source        string   // Required, This is the source of the analysis, It may be the root of the cluster analyzed (ie: there can be multiple)
	Informational bool     // Defaults to false, if this is not an issue but an Informational note (TBD: may separate these)
	Summary       string   // Required, there must be a Summary of the issue included
	Actions       []Action // Optional, if Actions are known these are included. Actions will be reported in the order specified

	SupportingData []SupportData // Optional but highly desirable for issues when possible. Data that helps support issue identification
	Confidence     int           // Required if not informational 0-10 ()
	Impact         int           // Optional 0-10 (TBD: This is a swag at how broad the impact is, 0 low, 10 high, defaults to -1 unknown)
}

// Validate validates an issue. A zeroed Issue is not valid, there is some amount of information that must be specified for the Issue to
// be useful. Currently the report will validate that the issues contributed are valid at the point where they are
// being contributed.
func (issue *Issue) Validate(log *zap.SugaredLogger, mapSource string) (err error) {
	if len(issue.Type) == 0 {
		return errors.New("A Type is required for an Issue")
	}
	if len(issue.Source) == 0 {
		return errors.New("A Source is required for an Issue")
	}
	// If there was a map source supplied, this means we are additionally checking that the source key
	// for the map matches the issue source as well (ie: when handed a map/slice of issues and a source
	// key, we check these here). If there is no mapSource supplied it just means the issue Source is used for
	// map insertions.
	if len(mapSource) != 0 && issue.Source != mapSource {
		return fmt.Errorf("The issue source %s doesn't match the map source supplied %s", issue.Source, mapSource)
	}
	if len(issue.Summary) == 0 {
		return errors.New("A Summary is required for an Issue")
	}
	if len(issue.Actions) > 0 {
		for _, action := range issue.Actions {
			err = action.Validate(log)
			if err != nil {
				log.Debugf("Action related to issue %s was invalid", issue.Summary, err)
				return err
			}
		}
	}
	if issue.Confidence < 0 || issue.Confidence > 10 {
		log.Debugf("Confidence %d is out of range, related to issue %s", issue.Confidence, issue.Summary)
		return fmt.Errorf("Confidence %d is out of range, related to issue %s", issue.Confidence, issue.Summary)
	}
	return nil
}

// Known Issue Types.
const (
	ImagePullBackOff       = "ImagePullBackOff"
	InsufficientMemory     = "InsufficientMemory"
	PodProblemsNotReported = "PodProblemsNotReported"
)

// NOTE: How we are handling the issues/actions/reporting is still very much evolving here. Currently supplying some
// helpers to reduce boilerplate when creating/reporting issues with common cases.

// Known Issue Templates. While analyzers are free to roll their own custom Issues, the preference for well-known issues is to capture them
// here so they are more generally available.
var knownIssues = map[string]Issue{
	ImagePullBackOff:       {Type: ImagePullBackOff, Summary: "ImagePullBackOff detected", Informational: false, Impact: 10, Confidence: 10, Actions: []Action{KnownActions[ImagePullBackOff]}},
	InsufficientMemory:     {Type: InsufficientMemory, Summary: "InsufficientMemory detected", Informational: false, Impact: 10, Confidence: 10, Actions: []Action{KnownActions[InsufficientMemory]}},
	PodProblemsNotReported: {Type: PodProblemsNotReported, Summary: "Problem pods were detected, but no issues were detected", Informational: true, Impact: 0, Confidence: 10, Actions: []Action{KnownActions[PodProblemsNotReported]}},
}

// NewKnownIssueSupportingData adds a known issue
func NewKnownIssueSupportingData(issueType string, source string, supportingData []SupportData) (issue Issue) {
	knownIssue, ok := knownIssues[issueType]
	if !ok {
		panic("Not a known issue, add a new known issue or create a custom issue")
	}
	issue = knownIssue
	issue.Source = source
	issue.SupportingData = supportingData
	return issue
}

// NewKnownIssueMessagesFiles adds a known issue
func NewKnownIssueMessagesFiles(issueType string, source string, messages []string, fileNames []string) (issue Issue) {
	issue = getKnownIssueOrDie(issueType)
	issue = knownIssues[issueType]
	issue.Source = source
	issue.SupportingData = make([]SupportData, 1, 1)
	issue.SupportingData[0] = SupportData{
		Messages:     messages,
		RelatedFiles: fileNames,
	}
	return issue
}

// NewKnownIssueMessagesMatches adds a known issue
func NewKnownIssueMessagesMatches(issueType string, source string, messages []string, matches []files.TextMatch) (issue Issue) {
	issue = getKnownIssueOrDie(issueType)
	issue.Source = source
	issue.SupportingData = make([]SupportData, 1, 1)
	issue.SupportingData[0] = SupportData{
		Messages:    messages,
		TextMatches: matches,
	}
	return issue
}

// IssueReporter is a helper for consolidating known issues before contributing them to the report
// An analyzer may is free to use the IssueReporter NewKnown* helpers for known issues, however they
// are not required to do so and are free to form fully custom issues and Contribute
// those directly to the report.Contribute* helpers. This allows analyzers flexibility, but the goal
// here is that the IssueReporter can evolve to support all of the cases if possible.
type IssueReporter struct {
	PendingIssues map[string]Issue
}

// AddKnownIssueSupportingData adds a known issue
func (issueReporter *IssueReporter) AddKnownIssueSupportingData(issueType string, source string, supportingData []SupportData) {
	confirmKnownIssueOrDie(issueType)

	// If this is a new issue, get a new one
	if issue, ok := issueReporter.PendingIssues[issueType]; !ok {
		issueReporter.PendingIssues[issueType] = NewKnownIssueSupportingData(issueType, source, supportingData)
	} else {
		issue.SupportingData = append(issue.SupportingData, supportingData...)
		issueReporter.PendingIssues[issueType] = issue
	}
}

// AddKnownIssueMessagesFiles adds a known issue
func (issueReporter *IssueReporter) AddKnownIssueMessagesFiles(issueType string, source string, messages []string, fileNames []string) {
	confirmKnownIssueOrDie(issueType)

	// If this is a new issue, get a new one
	if issue, ok := issueReporter.PendingIssues[issueType]; !ok {
		issueReporter.PendingIssues[issueType] = NewKnownIssueMessagesFiles(issueType, source, messages, fileNames)
	} else {
		supportData := SupportData{
			Messages:     messages,
			RelatedFiles: fileNames,
		}
		issue.SupportingData = append(issue.SupportingData, supportData)
		issueReporter.PendingIssues[issueType] = issue
	}
}

// AddKnownIssueMessagesMatches adds a known issue
func (issueReporter *IssueReporter) AddKnownIssueMessagesMatches(issueType string, source string, messages []string, matches []files.TextMatch) {
	confirmKnownIssueOrDie(issueType)

	// If this is a new issue, get a new one
	if issue, ok := issueReporter.PendingIssues[issueType]; !ok {
		issueReporter.PendingIssues[issueType] = NewKnownIssueMessagesMatches(issueType, source, messages, matches)
	} else {
		supportData := SupportData{
			Messages:    messages,
			TextMatches: matches,
		}
		issue.SupportingData = append(issue.SupportingData, supportData)
		issueReporter.PendingIssues[issueType] = issue
	}
}

// The helpers that work with known issue types only support working with those types
// If code is supplying an issueType that is not known, that is a coding error and we
// panic so that is clear immediately to the developer.
func getKnownIssueOrDie(issueType string) (issue Issue) {
	issue, ok := knownIssues[issueType]
	if !ok {
		panic("This helper is used with known issue types only")
	}
	return issue
}

func confirmKnownIssueOrDie(issueType string) {
	_, ok := knownIssues[issueType]
	if !ok {
		panic("This helper is used with known issue types only")
	}
}

// Contribute will contribute issues which have been added to the issue reporter
func (issueReporter *IssueReporter) Contribute(log *zap.SugaredLogger, source string) {
	if len(issueReporter.PendingIssues) == 0 {
		return
	}
	// Contribute the issues all at once
	ContributeIssuesMap(log, source, issueReporter.PendingIssues)
	issueReporter.PendingIssues = make(map[string]Issue)
}

// SingleMessage is a helper which is useful when adding a single message to supporting data
func SingleMessage(message string) (messages []string) {
	messages = make([]string, 1, 1)
	messages[0] = message
	return messages
}
