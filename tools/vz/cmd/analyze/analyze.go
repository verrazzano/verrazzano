// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis"
	vzbugreport "github.com/verrazzano/verrazzano/tools/vz/pkg/bugreport"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
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

var directory string

func NewCmdAnalyze(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return validateReportFormat(cmd)
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdAnalyze(cmd, vzHelper)
	}

	cmd.Example = helpExample
	cmd.PersistentFlags().String(constants.DirectoryFlagName, constants.DirectoryFlagValue, constants.DirectoryFlagUsage)
	cmd.PersistentFlags().String(constants.ReportFileFlagName, constants.ReportFileFlagValue, constants.ReportFileFlagUsage)
	cmd.PersistentFlags().String(constants.ReportFormatFlagName, constants.SummaryReport, constants.ReportFormatFlagUsage)
	cmd.PersistentFlags().BoolP(constants.VerboseFlag, constants.VerboseFlagShorthand, constants.VerboseFlagDefault, constants.VerboseFlagUsage)
	return cmd
}

// analyzeLiveCluster Analyzes live cluster by capturing the snapshot, when capture-dir is not set
func analyzeLiveCluster(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	// Get the kubernetes clientset, which will validate that the kubeconfig and context are valid.
	kubeClient, err := vzHelper.GetKubeClient(cmd)
	if err != nil {
		return err
	}

	// Get the dynamic client to retrieve OAM resources
	dynamicClient, err := vzHelper.GetDynamicClient(cmd)
	if err != nil {
		return err
	}

	// Get the controller runtime client
	client, err := vzHelper.GetClient(cmd)
	if err != nil {
		return err
	}

	// Create a temporary directory to place the generated files, which will also be the input for analyze command
	directory, err = os.MkdirTemp("", constants.BugReportDir)
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
	podLogs := vzbugreport.PodLogs{
		IsPodLog: true,
		Duration: int64(0),
	}
	return vzbugreport.CaptureClusterSnapshot(kubeClient, dynamicClient, client, reportDirectory, moreNS, vzHelper, podLogs)
}

func runCmdAnalyze(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	directoryFlag := cmd.PersistentFlags().Lookup(constants.DirectoryFlagName)
	if err := setVzK8sVersion(directoryFlag, vzHelper, cmd); err == nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), helpers.GetVersionOut())
	}
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
	if directoryFlag == nil || directoryFlag.Value.String() == "" {
		return analyzeLiveCluster(cmd, vzHelper)
	} else {
		directory, err = cmd.PersistentFlags().GetString(constants.DirectoryFlagName)
		if err != nil {
			fmt.Fprintf(vzHelper.GetOutputStream(), "error fetching flags: %s", err.Error())
		}
	}

	return analysis.AnalysisMain(vzHelper, directory, reportFileName, reportFormat)
}

// setVzK8sVersion sets vz and k8s version
func setVzK8sVersion(directoryFlag *pflag.Flag, vzHelper helpers.VZHelper, cmd *cobra.Command) error {
	if directoryFlag == nil || directoryFlag.Value.String() == "" {
		// Get the controller runtime client
		client, err := vzHelper.GetClient(cmd)
		if err != nil {
			return err
		}
		// set vz version
		if err := helpers.SetVzVer(&client); err != nil {
			return err
		}
		// set cluster k8s version
		if err := helpers.SetK8sVer(); err != nil {
			return err
		}
		// print k8s and vz version on console stdout
		return nil
	}
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
