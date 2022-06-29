// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package analyze

import (
	"fmt"
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/analysis"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "analyze"
	helpShort   = "Analyze cluster"
	helpLong    = `Analyze cluster for identifying issues and providing advice`
	helpExample = `# Run analysis tool on captured directory
$vz analyze --capture-dir <path>
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
	cmd.PersistentFlags().String(constants.ReportFormatFlagName, constants.ReportFormatFlagValue, constants.ReportFormatFlagUsage)
	cmd.MarkPersistentFlagRequired(constants.DirectoryFlagName)
	return cmd
}

func runCmdAnalyze(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	directory, err := cmd.PersistentFlags().GetString(constants.DirectoryFlagName)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "error fetching flags: %s", err.Error())
	}
	reportFileName, err := cmd.PersistentFlags().GetString(constants.ReportFileFlagName)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "error fetching flags: %s", err.Error())
	}
	reportFormat := GetReportFormat(cmd)

	return analysis.AnalysisMain(vzHelper, directory, reportFileName, reportFormat.String())
}

func validateReportFormat(cmd *cobra.Command) error {
	reportFormatValue := GetReportFormat(cmd)
	if reportFormatValue == "simple" {
		return nil
	}
	return fmt.Errorf("unsupported output format: %s, only supported type is \"simple\"", reportFormatValue)
}

func GetReportFormat(cmd *cobra.Command) cmdhelpers.LogFormat {
	reportFormat := cmd.PersistentFlags().Lookup(constants.ReportFormatFlagName)
	if reportFormat == nil {
		return cmdhelpers.LogFormatSimple
	}
	return cmdhelpers.LogFormat(reportFormat.Value.String())
}
