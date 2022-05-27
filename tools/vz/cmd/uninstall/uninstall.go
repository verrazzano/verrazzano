// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName   = "uninstall"
	purgeFlag     = "purge"
	purgeFlagHelp = "Completely remove all resources including cluster-wide resources from cluster."
	helpShort     = "Uninstall Verrazzano"
	helpLong      = `Uninstall the Verrazzano Platform Operator and all of the currently installed components.`
	helpExample   = `
TBD`
)

var logsEnum = cmdhelpers.LogsFormatPretty

func NewCmdUninstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdUninstall(cmd, args, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, false, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an uninstall.")
	cmd.PersistentFlags().Var(&logsEnum, constants.LogsFlag, constants.LogsFlagHelp)
	cmd.PersistentFlags().Bool(purgeFlag, false, purgeFlagHelp)

	return cmd
}

func runCmdUninstall(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	fmt.Fprintf(vzHelper.GetOutputStream(), "Not implemented yet\n")
	return nil
}
