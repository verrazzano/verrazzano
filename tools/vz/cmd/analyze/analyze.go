// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"fmt"
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis"
	vzbugreport "github.com/verrazzano/verrazzano/tools/vz/pkg/bugreport"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io/ioutil"
	"os"
	"path/filepath"
)

const (
	CommandName = "analyze"
	helpShort   = "Analyze cluster"
	helpLong    = `Analyze cluster for identifying issues and providing advice`
	helpExample = `
# Run analysis tool on captured directory
vz analyze --capture-dir <path>

# Run analysis tool on the live cluster
vz analyze
`
)

func NewCmdAnalyze(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return validateReportFormat(cmd)
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdAnalyze(cmd, args, vzHelper)
	}

	cmd.Example = helpExample
	cmd.PersistentFlags().String(constants.DirectoryFlagName, constants.DirectoryFlagValue, constants.DirectoryFlagUsage)
	cmd.PersistentFlags().String(constants.ReportFileFlagName, constants.ReportFileFlagValue, constants.ReportFileFlagUsage)
	cmd.PersistentFlags().String(constants.ReportFormatFlagName, constants.SummaryReport, constants.ReportFormatFlagUsage)
	cmd.PersistentFlags().BoolP(constants.VerboseFlag, constants.VerboseFlagShorthand, constants.VerboseFlagDefault, constants.VerboseFlagUsage)
	return cmd
}

func runCmdAnalyze(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	reportFileName, err := cmd.PersistentFlags().GetString(constants.ReportFileFlagName)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "error fetching flags: %s", err.Error())
	}
	reportFormat := getReportFormat(cmd)

	// set the flag to control the display the resources captured
	isVerbose, err := cmd.PersistentFlags().GetBool(constants.VerboseFlag)
	if err != nil {
		return fmt.Errorf("an error occurred while reading value for the flag %s: %s", constants.VerboseFlag, err.Error())
	}
	helpers.SetVerboseOutput(isVerbose)

	directoryFlag := cmd.PersistentFlags().Lookup(constants.DirectoryFlagName)

	directory := ""
	if directoryFlag == nil || directoryFlag.Value.String() == "" {
		// Analyze live cluster by capturing the snapshot, when capture-dir is not set

		// Get the kubernetes clientset, which will validate that the kubeconfig and context are valid.
		kubeClient, err := vzHelper.GetKubeClient(cmd)
		if err != nil {
			return err
		}

		// Get the controller runtime client
		client, err := vzHelper.GetClient(cmd)
		if err != nil {
			return err
		}

		// Get the dynamic client to retrieve OAM resources
		dynamicClient, err := vzHelper.GetDynamicClient(cmd)
		if err != nil {
			return err
		}

		// Create a temporary directory to place the generated files, which will also be the input for analyze command
		directory, err = ioutil.TempDir("", constants.BugReportDir)
		if err != nil {
			return fmt.Errorf("an error occurred while creating the directory to place cluster resources: %s", err.Error())
		}
		defer os.RemoveAll(directory)

		// Create a directory for the analyze command
		reportDirectory := filepath.Join(directory, constants.BugReportRoot)
		err = os.MkdirAll(reportDirectory, os.ModePerm)
		if err != nil {
			return fmt.Errorf("an error occurred while creating the directory %s: %s", reportDirectory, err.Error())
		}

		// Get the list of namespaces with label verrazzano-managed=true, where the applications are deployed
		moreNS := helpers.GetVZManagedNamespaces(kubeClient)

		// Instruct the helper to display the message for analyzing the live cluster
		helpers.SetIsLiveCluster()

		// Capture cluster snapshot
		err = vzbugreport.CaptureClusterSnapshot(kubeClient, dynamicClient, client, reportDirectory, moreNS, vzHelper)

		if err != nil {
			return fmt.Errorf(err.Error())
		}
	} else {
		directory, err = cmd.PersistentFlags().GetString(constants.DirectoryFlagName)
		if err != nil {
			fmt.Fprintf(vzHelper.GetOutputStream(), "error fetching flags: %s", err.Error())
		}
	}
	return analysis.AnalysisMain(vzHelper, directory, reportFileName, reportFormat)
}

// validateReportFormat validates the value specified for flag report-format
func validateReportFormat(cmd *cobra.Command) error {
	reportFormatValue := getReportFormat(cmd)
	switch reportFormatValue {
	case constants.SummaryReport, constants.DetailedReport:
		return nil
	default:
		return fmt.Errorf("%q is not valid for flag report-format, only %q and %q are valid", reportFormatValue, constants.SummaryReport, constants.DetailedReport)
	}
}

// getReportFormat returns the value set for flag report-format
func getReportFormat(cmd *cobra.Command) string {
	reportFormat := cmd.PersistentFlags().Lookup(constants.ReportFormatFlagName)
	if reportFormat == nil {
		return constants.SummaryReport
	}
	return reportFormat.Value.String()
}
