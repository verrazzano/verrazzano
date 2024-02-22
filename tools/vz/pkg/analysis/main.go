// Copyright (c) 2021, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analysis

import (
	"fmt"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/cluster"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/internal/util/report"
	"go.uber.org/zap"
)

var analyzerTypeFunctions = map[string]func(vzHelper helpers.VZHelper, log *zap.SugaredLogger, args string) (err error){
	"cluster": cluster.RunAnalysis,
}

var analyzerType = "cluster" //Currently does only cluster analysis
var includeInfo = true
var includeSupport = true
var includeActions = true
var minImpact int
var minConfidence int
var logger *zap.SugaredLogger

// The analyze tool will analyze information which has already been captured from an environment
func AnalysisMain(vzHelper helpers.VZHelper, directory string, reportFile string, reportFormat string, printReportToConsole bool) error {
	logger = zap.S()
	return handleMain(vzHelper, directory, reportFile, reportFormat, printReportToConsole)
}

// handleMain is where the main logic is at, separated here to allow for more test coverage
func handleMain(vzHelper helpers.VZHelper, directory string, reportFile string, reportFormat string, printReportToConsole bool) error {
	// TODO: how we surface different analysis report types will likely change up, for now it is specified here, and it may also
	// make sense to treat all cluster dumps the same way whether single or multiple (structure the dumps the same way)
	// We could also have different types of report output formats as well. For example, the current report format is
	// presentin issues/actions/supporting data which would be used by someone with technical background to resolve an issue
	// in their environment. We also could generate a more detailed "bug-report-type" which someone could call which would
	// gather up information, sanitize it in a way that it could be sent along to someone else for further analysis, etc...

	// Call the analyzer for the type specified
	err := Analyze(vzHelper, logger, analyzerType, directory)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Analyze failed with error: %s, exiting.\n", err.Error())
		return fmt.Errorf("\nanalyze failed with error: %s, exiting", err.Error())
	}
	reportContext := helpers.ReportCtx{ReportFile: reportFile, ReportFormat: reportFormat, IncludeSupportData: includeSupport, IncludeInfo: includeInfo, IncludeActions: includeActions, MinConfidence: minConfidence, MinImpact: minImpact, PrintReportToConsole: printReportToConsole}

	// Generate a report
	err = report.GenerateHumanReport(logger, vzHelper, reportContext)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "\nReport generation failed, exiting.\n")
		return fmt.Errorf("%s", err.Error())
	}
	return nil
}

// Analyze is exported for unit testing
func Analyze(vzHelper helpers.VZHelper, logger *zap.SugaredLogger, analyzerType string, rootDirectory string) (err error) {
	// Call the analyzer for the type specified
	analyzerFunc, ok := analyzerTypeFunctions[analyzerType]
	if !ok {
		return fmt.Errorf("Unknown analyzer type supplied: %s", analyzerType)
	}
	err = analyzerFunc(vzHelper, logger, rootDirectory)
	if err != nil {
		return err
	}
	return nil
}
