// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package report handles reporting
package report

import (
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"go.uber.org/zap"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"sync"
)

// NOTE: This is part of the contract with the analyzers however it is currently an initial stake in the ground and
//		 will be evolving rapidly initially as we add analysis cases

// TODO: We have rudimentary settings and a rudimentary dump of the report to start with here. Ie: Bare bones to get
//       the details out for now, go from there... But there are some things that are already on the radar here:
//
//      1) Format of the human readable report will evolve (wrapping long lines, etc...)
//      2) Other format outputs suitable for automation to process (ie: other automation that will look at the results
//         and do something automatically with it), maybe CSV file, JSON, etc...
//      3) Detail consolidation/redacting suitable for sharing as a Bug report
//      4) Etc...
//

// For example, when we report them we will want to report:
//		1) Per source (cluster, build, etc...)
//		2) Sort in priority order (worst first...) TODO

// Tossing around whether per-source, if we have a map for tracking Issues so we have one Issue per type of issue
// and allow contributing supporting data to it (rather than separate issues for each case found if found in different spots
// I'm hesitant to do that now as that reduces the flexibility, and until we really have the analysis drill-down patterns and
// more scenarios in place I think it is premature to do that (we could maybe allow both, but again not sure we need
// that complexity yet either). One good example is that when there a a bunch of pods impacted by the same root cause
// issue, really don't need to spam with a bunch of the same issue (we could add additional supporting data to one root
// issue instead of having an issue for each occurrence), but the analyzer can manage that and knows more about whether
// it is really a different issue or not.

// We have a map per source. The source is a string here. Generally for clusters it would be something that identifies
// the cluster. But other analyzers may not be looking at a cluster, so they may have some other identification.
// For the current implementation, these are the root file path that the analyzer is looking at.
var reports = make(map[string][]Issue)
var allSourcesAnalyzed = make(map[string]string)
var reportMutex = &sync.Mutex{}

// ContributeIssuesMap allows a map of issues to be contributed
func ContributeIssuesMap(log *zap.SugaredLogger, source string, issues map[string]Issue) (err error) {
	log.Debugf("ContributeIssues called for source %s with %d issues", source, len(issues))
	if len(source) == 0 {
		return errors.New("ContributeIssues requires a non-empty source be specified")
	}
	for _, issue := range issues {
		err = issue.Validate(log, source)
		if err != nil {
			return err
		}
	}
	reportMutex.Lock()
	reportIssues := reports[source]
	if len(reportIssues) == 0 {
		reportIssues = make([]Issue, 0, 10)
	}
	for _, issue := range issues {
		issue.SupportingData = DeduplicateSupportingData(issue.SupportingData)
		reportIssues = append(reportIssues, issue)
	}
	reports[source] = reportIssues
	reportMutex.Unlock()
	return nil
}

// ContributeIssue allows a single issue to be contributed
func ContributeIssue(log *zap.SugaredLogger, issue Issue) (err error) {
	log.Debugf("ContributeIssue called for source with %v", issue)
	err = issue.Validate(log, "")
	if err != nil {
		log.Debugf("Validate failed %s", err.Error())
		return err
	}
	reportMutex.Lock()
	reportIssues := reports[issue.Source]
	if len(reportIssues) == 0 {
		reportIssues = make([]Issue, 0, 10)
	}
	issue.SupportingData = DeduplicateSupportingData(issue.SupportingData)
	reportIssues = append(reportIssues, issue)
	reports[issue.Source] = reportIssues
	reportMutex.Unlock()
	return nil
}

// GenerateHumanReport is a basic report generator
// TODO: This is super basic for now, need to do things like sort based on Confidence, add other formats on output, etc...
// Also add other niceties like time, Summary of what was analyzed, if no issues were found, etc...
func GenerateHumanReport(log *zap.SugaredLogger, reportFile string, reportFormat string, includeSupportData bool, includeInfo bool, includeActions bool, minConfidence int, minImpact int, vzHelper helpers.VZHelper) (err error) {
	// Default to stdout if no reportfile is supplied
	//TODO: Eventually add support other reportFormat type (json)
	writeOut, writeSummaryOut, sepOut := "", "", ""

	// Lock the report data while generating the report itself
	reportMutex.Lock()
	sourcesWithoutIssues := allSourcesAnalyzed
	for source, reportIssues := range reports {
		log.Debugf("Will report on %d issues that were reported for %s", len(reportIssues), source)
		// We need to filter and sort the list of Issues that will be reported
		// TODO: Need to sort them as well eventually
		actuallyReported := filterReportIssues(log, reportIssues, includeInfo, minConfidence, minImpact)
		if len(actuallyReported) == 0 {
			log.Debugf("No issues to report for source: %s", source)
			continue
		}

		// Print the Source as it has issues
		delete(sourcesWithoutIssues, source)
		var issuesDetected string
		if helpers.GetIsLiveCluster() {
			issuesDetected = fmt.Sprintf("Detected %d issues in the cluster:", len(actuallyReported))
		} else {
			issuesDetected = fmt.Sprintf("Detected %d issues for %s:", len(actuallyReported), source)
		}
		sepOut = "\n" + issuesDetected + "\n" + strings.Repeat(constants.LineSeparator, len(issuesDetected)) + "\n"
		for _, issue := range actuallyReported {
			// Display only summary and action when the report-format is set to summary
			if reportFormat == constants.SummaryReport {
				writeSummaryOut += fmt.Sprintf("\n\tISSUE (%s): %s\n", issue.Type, issue.Summary)
				for _, action := range issue.Actions {
					writeSummaryOut += fmt.Sprintf("\t%s\n", action.Summary)
				}

			}
			writeOut += fmt.Sprintf("\n\tISSUE (%s)\n\t\tsummary: %s\n", issue.Type, issue.Summary)
			if len(issue.Actions) > 0 && includeActions {
				log.Debugf("Output actions")
				writeOut += "\t\tactions:\n"
				for _, action := range issue.Actions {
					writeOut += fmt.Sprintf("\t\t\taction: %s\n", action.Summary)
					if len(action.Steps) > 0 {
						writeOut += "\t\t\t\tSteps:\n"
						for i, step := range action.Steps {
							writeOut += fmt.Sprintf("\t\t\t\t\tStep %d: %s\n", i+1, step)
						}
					}
					if len(action.Links) > 0 {
						writeOut += "\t\t\t\tLinks:\n"
						for _, link := range action.Links {
							writeOut += fmt.Sprintf("\t\t\t\t\t%s\n", link)
						}
					}
				}
			}
			if len(issue.SupportingData) > 0 && includeSupportData {
				log.Debugf("Output supporting data")
				writeOut += "\t\tsupportingData:\n"
				for _, data := range issue.SupportingData {
					if len(data.Messages) > 0 {
						writeOut += "\t\t\tmessages:\n"
						for _, message := range data.Messages {
							writeOut += fmt.Sprintf("\t\t\t\t%s\n", message)
						}
					}
					if len(data.TextMatches) > 0 {
						writeOut += "\t\t\tsearch matches:\n"
						for _, match := range data.TextMatches {
							if helpers.GetIsLiveCluster() {
								writeOut += fmt.Sprintf("\t\t\t%s: %s\n", match.FileName, match.MatchedText)
							} else {
								writeOut += fmt.Sprintf("\t\t\t\t%s:%d: %s\n", match.FileName, match.FileLine, match.MatchedText)
							}
						}
					}
					if len(data.JSONPaths) > 0 {
						writeOut += "\t\t\trelated json:\n"
						for _, path := range data.JSONPaths {
							writeOut += fmt.Sprintf("\t\t\t\t%s: %s\n", path.File, path.Path)
						}
					}
					if len(data.RelatedFiles) > 0 {
						writeOut += "\t\t\trelated resource(s):\n"
						for _, fileName := range data.RelatedFiles {
							writeOut += fmt.Sprintf("\t\t\t\t%s\n", fileName)
						}
					}
				}
			}
		}
	}

	if len(writeOut) > 0 {
		var fileOut *os.File
		if reportFile == "" {
			reportFile = constants.VzAnalysisReportTmpFile
			fileOut, err = os.CreateTemp(".", reportFile)
			if err != nil && errors.Is(err, fs.ErrPermission) {
				fmt.Fprintf(vzHelper.GetOutputStream(), "Warning: %s to open report file in current directory\n", fs.ErrPermission)
				fileOut, err = os.CreateTemp("", reportFile)
			}
		} else {
			fileOut, err = os.OpenFile(reportFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		}
		if err != nil {
			log.Errorf("Failed to create report file : %s, error found : %s", reportFile, err.Error())
			return err
		}
		_, err = fileOut.Write([]byte(helpers.GetVersionOut() + sepOut + writeOut))
		if reportFormat == constants.DetailedReport {
			fmt.Fprintf(vzHelper.GetOutputStream(), sepOut+writeOut)
		} else if reportFormat == constants.SummaryReport {
			fmt.Fprintf(vzHelper.GetOutputStream(), sepOut+writeSummaryOut)
		}
		if err != nil {
			log.Errorf("Failed to write to report file %s, error found : %s", reportFile, err.Error())
			return err
		}
		defer func() {
			fmt.Fprintf(os.Stdout, "\nDetailed report available in %s\n", fileOut.Name())
			fileOut.Close()
		}()
	} else {
		if includeInfo {
			if len(sourcesWithoutIssues) > 0 {
				writeOut += "\n\n"
			}
			if len(sourcesWithoutIssues) == 1 {
				// This is a workaround to avoid printing the source when analyzing the live cluster, although it impacts the
				// regular use case with a directory containing a single cluster snapshot
				writeOut += "Verrazzano analysis CLI did not detect any issue in the cluster\n"
			} else {
				for _, source := range sourcesWithoutIssues {
					writeOut += fmt.Sprintf("Verrazzano analysis CLI did not detect any issue in %s\n", source)
				}
			}
			fmt.Fprintf(vzHelper.GetOutputStream(), writeOut)
		}
	}
	reportMutex.Unlock()
	return nil
}

// AddSourceAnalyzed tells the report which sources have been analyzed. This way it knows
// the entire set of sources which were analyzed (not just the ones which had issues detected)
func AddSourceAnalyzed(source string) {
	reportMutex.Lock()
	allSourcesAnalyzed[source] = source
	reportMutex.Unlock()
}

// GetAllSourcesFilteredIssues is only being exported for the unit tests so they can inspect issues found in a report
func GetAllSourcesFilteredIssues(log *zap.SugaredLogger, includeInfo bool, minConfidence int, minImpact int) (filtered []Issue) {
	reportMutex.Lock()
	for _, reportIssues := range reports {
		subFiltered := filterReportIssues(log, reportIssues, includeInfo, minConfidence, minImpact)
		if len(subFiltered) > 0 {
			filtered = append(filtered, subFiltered...)
		}
	}
	reportMutex.Unlock()
	return filtered
}

// ClearReports clears the reports map, only for unit tests
func ClearReports() {
	reportMutex.Lock()
	reports = make(map[string][]Issue)
	reportMutex.Unlock()
}

// compare two structs are same or not
func isEqualStructs(s1, s2 any) bool {
	return reflect.DeepEqual(s1, s2)
}

// filter out duplicate issues
func deDuplicateIssues(reportIssues []Issue) []Issue {
	var deDuplicates = make([]Issue, 0, len(reportIssues))
	for _, i1 := range reportIssues {
		issueVisited := false
		for _, i2 := range deDuplicates {
			if isEqualStructs(i1, i2) {
				issueVisited = true
				break
			}
		}
		if !issueVisited {
			deDuplicates = append(deDuplicates, i1)
		}
	}
	return deDuplicates
}

func filterReportIssues(log *zap.SugaredLogger, reportIssues []Issue, includeInfo bool, minConfidence int, minImpact int) (filtered []Issue) {
	filtered = make([]Issue, 0, len(reportIssues))
	for _, issue := range reportIssues {
		// Skip issues that are Informational or lower Confidence that we want
		if issue.Informational && !includeInfo || issue.Confidence < minConfidence || issue.Impact < minImpact {
			log.Debugf("Skipping issue %s based on informational/confidence/impact settings", issue.Summary)
			continue
		}
		filtered = append(filtered, issue)
	}
	return deDuplicateIssues(filtered)
}
