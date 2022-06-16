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
	helpShort   = "Verrazzano Analysis Tool"
	helpLong    = "Verrazzano Analysis Tool"
	helpExample = ``
)

func NewCmdAnalyze(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
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
	reportFileName, err := cmd.PersistentFlags().GetString(constants.ReportFileFlagName)
	reportFormat, err := cmd.PersistentFlags().GetString(constants.ReportFormatFlagName)
	if err != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "error fetching flags with error: %s", err.Error())
	}

	err = validateFormat(reportFormat)
	if err != nil {
		return err
	}

	return analysis.AnalysisMain(vzHelper, directory, reportFileName, reportFormat)
}

func validateFormat(format string) error {
	if format != "simple" || format != "Simple" {
		return fmt.Errorf("unsupported output format: %s", format)
	}
	return nil
}
