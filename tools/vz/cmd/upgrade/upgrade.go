// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upgrade

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "upgrade"
	helpShort   = "Upgrade Verrazzano"
	helpLong    = `Upgrade the Verrazzano Platform Operator to the specified version and update all of the currently installed components.`
	helpExample = `
# Upgrade to the latest version of Verrazzano and wait for the command to complete.
vz upgrade --wait

# Upgrade to Verrazzano v1.3.0 and stream the logs to the console.
vz upgrade --version v1.3.0 --logs`
)

var logsEnum = cmdhelpers.LogsFormatSimple

func NewCmdUpgrade(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdUpgrade(cmd, args, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, false, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().String(constants.VersionFlag, "latest", constants.VersionFlagHelp)
	cmd.PersistentFlags().Var(&logsEnum, constants.LogsFlag, constants.LogsFlagHelp)

	// Initially the operator-file flag may be for internal use, hide from help until
	// a decision is made on supporting this option.
	cmd.PersistentFlags().String(constants.OperatorFileFlag, "", constants.OperatorFileFlagHelp)
	cmd.PersistentFlags().MarkHidden(constants.OperatorFileFlag)

	// Dry run flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an upgrade.")
	cmd.PersistentFlags().MarkHidden(constants.DryRunFlag)

	return cmd
}

func runCmdUpgrade(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	fmt.Fprintf(vzHelper.GetOutputStream(), "Not implemented yet\n")
	return nil
}
