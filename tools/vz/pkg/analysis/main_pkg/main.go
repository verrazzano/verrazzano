// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main_pkg

import (
	"flag"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/buildlog"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/cluster"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis/internal/util/report"
	"go.uber.org/zap"
	"os"
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
var analyzerType string
var reportFile string
var includeInfo bool
var includeSupport bool
var includeActions bool
var minImpact int
var minConfidence int
var flagArgs []string
var logger *zap.SugaredLogger

// The analyze tool will analyze information which has already been captured from an environment
func AnalysisMain(directory string) {
	initFlags(directory)
	os.Exit(handleMain())
}

// initFlags is handled here. Separated out here from the main logic for now to allow for more main test coverage
// TODO: Look at if we can reliably mess with flag variants in Go unit tests
func initFlags(directory string) {
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
	//flagArgs = flag.Args()
	flagArgs = append(flagArgs, directory)
}

// handleMain is where the main logic is at, separated here to allow for more test coverage
func handleMain() (exitCode int) {
	// TODO: how we surface different analysis report types will likely change up, for now it is specified here, and it may also
	// make sense to treat all cluster dumps the same way whether single or multiple (structure the dumps the same way)
	// We could also have different types of report output formats as well. For example, the current report format is
	// presenting issues/actions/supporting data which would be used by someone with technical background to resolve an issue
	// in their environment. We also could generate a more detailed "bug-report-type" which someone could call which would
	// gather up information, sanitize it in a way that it could be sent along to someone else for further analysis, etc...

	if version {
		printVersion()
		return 0
	}

	if help {
		printUsage()
		return 0
	}

	if len(flagArgs) < 1 {
		fmt.Printf("\nCaptured data directory was not specified for analysis, exiting.\n")
		printUsage()
		return 1
	}

	// We already handle finding multiple cluster dumps in a directory, we could look
	// at multiple here as well if that really is needed, for now we expect one root
	// directory
	if len(flagArgs) > 1 {
		fmt.Printf("\nToo many arguments were supplied, exiting.\n")
		printUsage()
		return 1
	}

	if minConfidence < 0 || minConfidence > 10 {
		fmt.Printf("\nminConfidence is out of range %d, exiting.\n", minConfidence)
		printUsage()
		return 1
	}

	if minImpact < 0 || minImpact > 10 {
		fmt.Printf("\nminImpact is out of range %d, exiting.\n", minImpact)
		printUsage()
		return 1
	}

	// Call the analyzer for the type specified
	err := Analyze(logger, analyzerType, flagArgs[0])
	if err != nil {
		fmt.Printf("\nAnalyze failed: %s, exiting.\n", err)
		return 1
	}

	// Generate a report
	err = report.GenerateHumanReport(logger, reportFile, includeSupport, includeInfo, includeActions, minConfidence, minImpact)
	if err != nil {
		fmt.Printf("\nReport generation failed, exiting.\n")
		return 1
	}
	return 0
}

// Analyze is exported for unit testing
func Analyze(logger *zap.SugaredLogger, analyzerType string, flagArgs string) (err error) {
	// Call the analyzer for the type specified
	analyzerFunc, ok := analyzerTypeFunctions[analyzerType]
	if !ok {
		//printUsage()
		return fmt.Errorf("Unknown analyzer type supplied: %s", analyzerType)
	}
	err = analyzerFunc(logger, flagArgs)
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
