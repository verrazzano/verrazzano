// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"flag"
	"fmt"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/buildlog"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/cluster"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/log"
	"github.com/verrazzano/verrazzano/tools/analysis/internal/util/report"
	"go.uber.org/zap"
	"os"
	kzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var analyzerTypeFunctions = map[string]func(log *zap.SugaredLogger, args string) (err error){
	"cluster":  cluster.RunAnalysis,
	"buildlog": buildlog.RunAnalysis,
}

// The analyze tool will analyze information which has already been captured from an environment
func main() {

	help := false

	// TODO: how we surface different analysis report types will likely change up, for now it is specified here, and it may also
	// make sense to treat all cluster dumps the same way whether single or multiple (structure the dumps the same way)
	// We could also have different types of report output formats as well. For example, the current report format is
	// presenting issues/actions/supporting data which would be used by someone with technical background to resolve an issue
	// in their environment. We also could generate a more detailed "bug-report-type" which someone could call which would
	// gather up information, sanitize it in a way that it could be sent along to someone else for further analysis, etc...
	var analyzerType string
	var reportFile string
	var includeInfo bool
	var includeSupport bool
	var includeActions bool
	var minImpact int
	var minConfidence int
	flag.StringVar(&analyzerType, "analysis", "cluster", "Type of analysis: cluster, buildlog")
	flag.StringVar(&reportFile, "reportFile", "", "Name of report output file, default is stdout")
	flag.BoolVar(&includeInfo, "info", true, "Include informational messages, default is true")
	flag.BoolVar(&includeSupport, "support", true, "Include support data in the report, default is true")
	flag.BoolVar(&includeActions, "actions", true, "Include actions in the report, default is true")
	flag.IntVar(&minImpact, "minImpact", 0, "Minimum impact threshold to report for issues, 0-10, default is 0")
	flag.IntVar(&minConfidence, "minConfidence", 0, "Minimum confidence threshold to report for issues, 0-10, default is 0")
	flag.BoolVar(&help, "help", false, "Display usage help")

	// Add the zap logger flag set to the CLI.
	opts := kzap.Options{}
	opts.BindFlags(flag.CommandLine)

	flag.Parse()
	kzap.UseFlagOptions(&opts)
	log.InitLogs(opts)
	log := zap.S()

	if help {
		printUsage()
		os.Exit(0)
	}

	if len(flag.Args()) < 1 {
		fmt.Printf("\nCaptured data directory was not specified for analysis, exiting.\n")
		printUsage()
		os.Exit(0)
	}

	// We already handle finding multiple cluster dumps in a directory, we could look
	// at multiple here as well if that really is needed, for now we expect one root
	// directory
	if len(flag.Args()) > 1 {
		fmt.Printf("\nToo many arguments were supplied, exiting.\n")
		printUsage()
		os.Exit(1)
	}

	// Call the analyzer for the type specified
	err := Analyze(log, analyzerType, flag.Args()[0])
	if err != nil {
		fmt.Printf("\nAnalyze failed: %s, exiting.\n", err)
		os.Exit(1)
	}

	// Generate a report
	err = report.GenerateHumanReport(log, reportFile, includeSupport, includeInfo, includeActions, minConfidence, minImpact)
	if err != nil {
		fmt.Printf("\nReport generation failed, exiting.\n")
		os.Exit(1)
	}

	os.Exit(0)
}

// Exported for unit testing
func Analyze(log *zap.SugaredLogger, analyzerType string, flagArgs string) (err error) {
	// Call the analyzer for the type specified
	analyzerFunc, ok := analyzerTypeFunctions[analyzerType]
	if !ok {
		printUsage()
		return fmt.Errorf("Unknown analyzer type supplied: %s", analyzerType)
	}
	err = analyzerFunc(log, flagArgs)
	if err != nil {
		return err
	}
	return nil
}

// printUsage Prints the help for this program
func printUsage() {
	usageString := `
Usage: go run main.go [options] captured-data-directory
Options:
`
	fmt.Printf(usageString)
	flag.PrintDefaults()
}
