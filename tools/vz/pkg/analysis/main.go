// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package analysis

import (
	"flag"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/buildlog"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/cluster"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"go.uber.org/zap"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var analyzerTypeFunctions = map[string]func(log *zap.SugaredLogger, args string) (err error){
	"cluster":  cluster.RunAnalysis,
	"buildlog": buildlog.RunAnalysis,
}

// The analysisToolVersion is set during the build, this value is a default if the tool is built differently
var analysisToolVersion = "development build"

var version = false
var help = false
var analyzerType = "cluster" //Currently does only cluster analysis
var reportFile string
var includeInfo bool
var includeSupport bool
var includeActions bool
var minImpact int
var minConfidence int
var flagArgs []string
var logger *zap.SugaredLogger

// The analyze tool will analyze information which has already been captured from an environment
func AnalysisMain(vzHelper helpers.VZHelper, directory string, reportFile string, reportFormat string) error {
	initFlags()
	return handleMain(vzHelper, directory, reportFile, reportFormat)
}

// initFlags is handled here. Separated out here from the main logic for now to allow for more main test coverage
// TODO: Look at if we can reliably mess with flag variants in Go unit tests
func initFlags() {
	flag.StringVar(&analyzerType, "analysis", "cluster", "Type of analysis: cluster")
	flag.StringVar(&reportFile, "reportFile", "", "Name of report output file, default is stdout")
	flag.BoolVar(&includeInfo, "info", true, "Include informational messages, default is true")
	flag.BoolVar(&includeSupport, "support", true, "Include support data in the report, default is true")
	flag.BoolVar(&includeActions, "actions", true, "Include actions in the report, default is true")
	flag.IntVar(&minImpact, "minImpact", 0, "Minimum impact threshold to report for issues, 0-10, default is 0")
	flag.IntVar(&minConfidence, "minConfidence", 0, "Minimum confidence threshold to report for issues, 0-10, default is 0")
	flag.BoolVar(&help, "help", false, "Display usage help")
	flag.BoolVar(&version, "version", false, "Display version")
	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)
	log.InitLogs(opts)
	logger = zap.S()

	// Doing this to allow unit testing main logic
	flagArgs = flag.Args()
}

// handleMain is where the main logic is at, separated here to allow for more test coverage
func handleMain(vzHelper helpers.VZHelper, directory string, reportFile string, reportFormat string) error {
	// TODO: how we surface different analysis report types will likely change up, for now it is specified here, and it may also
	// make sense to treat all cluster dumps the same way whether single or multiple (structure the dumps the same way)
	// We could also have different types of report output formats as well. For example, the current report format is
	// presenting issues/actions/supporting data which would be used by someone with technical background to resolve an issue
	// in their environment. We also could generate a more detailed "bug-report-type" which someone could call which would
	// gather up information, sanitize it in a way that it could be sent along to someone else for further analysis, etc...

	// Call the analyzer for the type specified
	err := Analyze(logger, analyzerType, directory)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Analyze failed with error: %s, exiting.\n", err.Error())
		return fmt.Errorf("\nAnalyze failed with error: %s, exiting.\n", err.Error())
	}

	// Generate a report
	err = report.GenerateHumanReport(logger, reportFile, reportFormat, includeSupport, includeInfo, includeActions, minConfidence, minImpact, vzHelper)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "\nReport generation failed, exiting.\n")
		return fmt.Errorf("\nReport generation failed, exiting.\n")
	}
	return nil
}

// Analyze is exported for unit testing
func Analyze(logger *zap.SugaredLogger, analyzerType string, rootDirectory string) (err error) {
	// Call the analyzer for the type specified
	analyzerFunc, ok := analyzerTypeFunctions[analyzerType]
	if !ok {
		//printUsage()
		return fmt.Errorf("Unknown analyzer type supplied: %s", analyzerType)
	}
	err = analyzerFunc(logger, rootDirectory)
	if err != nil {
		return err
	}
	return nil
}

// printUsage Prints the help for this program
func printUsage() {
	usageString := `
Usage: verrazzano-analysis [options] captured-data-directory
Options:
`
	fmt.Printf(usageString)
	flag.PrintDefaults()
}

// printVersion Prints the version for this program
func printVersion() {
	fmt.Printf("%s\n", analysisToolVersion)
}
