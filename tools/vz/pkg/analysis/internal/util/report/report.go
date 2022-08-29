// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package report handles reporting
package report

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"go.uber.org/zap"
	"os"
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
	log.Debugf("ContributeIssues called for source %s with %d issues", len(issues))
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
	log.Debugf("ContributeIssue called for source %s with %v", issue)
	err = issue.Validate(log, "")
	if err != nil {
		log.Debugf("Validate failed", err)
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
	var writeOut = bufio.NewWriter(vzHelper.GetOutputStream())
	if len(reportFile) > 0 {
		log.Debugf("Generating human report to file: %s", reportFile)
		// Open the file for write
		fileOut, err := os.Create(reportFile)
		if err != nil {
			log.Errorf("Failed to create report file %s", reportFile, err)
			return err
		}
		defer fileOut.Close()
		writeOut = bufio.NewWriter(fileOut)
	} else {
		log.Debugf("Generating human report to stdout")
	}

	// Lock the report data while generating the report itself
	reportMutex.Lock()
	sourcesWithoutIssues := allSourcesAnalyzed
	for source, reportIssues := range reports {
		log.Debugf("Will report on %d issues that were reported for %s", len(reportIssues), source)

		// We need to filter and sort the list of Issues that will be reported
		// TODO: Need to sort them as well eventually
		actuallyReported := filterReportIssues(log, reportIssues, includeInfo, minConfidence, minImpact)
		if len(actuallyReported) == 0 {
			log.Debugf("No issues to report for source: %s")
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
		sep := strings.Repeat(constants.LineSeparator, len(issuesDetected))
		fmt.Fprintf(writeOut, "\n"+issuesDetected+"\n")
		fmt.Fprintf(writeOut, sep+"\n")

		for _, issue := range actuallyReported {
			// Display only summary and action when the report-format is set to summary
			if reportFormat == constants.SummaryReport {
				_, err = fmt.Fprintf(writeOut, "\n\tISSUE (%s): %s\n", issue.Type, issue.Summary)
				if err != nil {
					return err
				}
				for _, action := range issue.Actions {
					_, err = fmt.Fprintf(writeOut, "\t%s\n", action.Summary)
					if err != nil {
						return err
					}
				}
				continue
			}

			// Print the Issue out
			_, err = fmt.Fprintf(writeOut, "\n\tISSUE (%s)\n\t\tsummary: %s\n", issue.Type, issue.Summary)
			if err != nil {
				return err
			}
			// Revisit to display confidence and impact, if/when required
			//_, err = fmt.Fprintf(writeOut, "\t\tconfidence: %d\n", issue.Confidence)
			//if err != nil {
			//	return err
			//}
			//_, err = fmt.Fprintf(writeOut, "\t\timpact: %d\n", issue.Impact)
			//if err != nil {
			//	return err
			//}
			if len(issue.Actions) > 0 && includeActions {
				log.Debugf("Output actions")
				_, err = fmt.Fprintf(writeOut, "\t\tactions:\n")
				if err != nil {
					return err
				}
				for _, action := range issue.Actions {
					_, err = fmt.Fprintf(writeOut, "\t\t\taction: %s\n", action.Summary)
					if err != nil {
						return err
					}
					if len(action.Steps) > 0 {
						_, err = fmt.Fprintf(writeOut, "\t\t\t\tSteps:\n")
						if err != nil {
							return err
						}
						for i, step := range action.Steps {
							_, err = fmt.Fprintf(writeOut, "\t\t\t\t\tStep %d: %s\n", i+1, step)
							if err != nil {
								return err
							}
						}
					}
					if len(action.Links) > 0 {
						_, err = fmt.Fprintf(writeOut, "\t\t\t\tLinks:\n")
						if err != nil {
							return err
						}
						for _, link := range action.Links {
							_, err = fmt.Fprintf(writeOut, "\t\t\t\t\t%s\n", link)
							if err != nil {
								return err
							}
						}
					}
				}
			}
			if len(issue.SupportingData) > 0 && includeSupportData {
				log.Debugf("Output supporting data")
				_, err = fmt.Fprintf(writeOut, "\t\tsupportingData:\n")
				if err != nil {
					return err
				}
				for _, data := range issue.SupportingData {
					if len(data.Messages) > 0 {
						_, err = fmt.Fprintf(writeOut, "\t\t\tmessages:\n")
						if err != nil {
							return err
						}
						for _, message := range data.Messages {
							_, err = fmt.Fprintf(writeOut, "\t\t\t\t%s\n", message)
							if err != nil {
								return err
							}
						}
					}
					if len(data.TextMatches) > 0 {
						_, err = fmt.Fprintf(writeOut, "\t\t\tsearch matches:\n")
						if err != nil {
							return err
						}
						for _, match := range data.TextMatches {
							if helpers.GetIsLiveCluster() {
								_, err = fmt.Fprintf(writeOut, "\t\t\t%s: %s\n", match.FileName, match.MatchedText)
							} else {
								_, err = fmt.Fprintf(writeOut, "\t\t\t\t%s:%d: %s\n", match.FileName, match.FileLine, match.MatchedText)
							}
							if err != nil {
								return err
							}
						}
					}
					if len(data.JSONPaths) > 0 {
						_, err = fmt.Fprintf(writeOut, "\t\t\trelated json:\n")
						if err != nil {
							return err
						}
						for _, path := range data.JSONPaths {
							_, err = fmt.Fprintf(writeOut, "\t\t\t\t%s: %s\n", path.File, path.Path)
							if err != nil {
								return err
							}
						}
					}
					if len(data.RelatedFiles) > 0 {
						_, err = fmt.Fprintf(writeOut, "\t\t\trelated resource(s):\n")
						if err != nil {
							return err
						}
						for _, fileName := range data.RelatedFiles {
							_, err = fmt.Fprintf(writeOut, "\t\t\t\t%s\n", fileName)
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}
	}

	if includeInfo {
		if len(sourcesWithoutIssues) > 0 {
			_, err = fmt.Fprintf(writeOut, "\n\n")
		}
		if len(sourcesWithoutIssues) == 1 {
			// This is a workaround to avoid printing the source when analyzing the live cluster, although it impacts the
			// regular use case with a directory containing a single cluster snapshot
			_, err = fmt.Fprintf(writeOut, "Verrazzano analysis CLI did not detect any issue in the cluster\n")
		} else {
			for _, source := range sourcesWithoutIssues {
				_, err = fmt.Fprintf(writeOut, "Verrazzano analysis CLI did not detect any issue in %s\n", source)
			}
		}
	}

	log.Debugf("Flushing output")
	err = writeOut.Flush()
	if err != nil {
		log.Errorf("Failed to flush writer for file %s", reportFile, err)
		return err
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
	return filtered
}
