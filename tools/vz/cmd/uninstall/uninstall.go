// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package uninstall

import (
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
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

func NewCmdUninstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Run = runCmdVersion
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, false, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*20, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an uninstall")
	cmd.PersistentFlags().String(constants.LogsFlag, constants.LogsFormatPretty, constants.LogsFlagHelp)
	cmd.PersistentFlags().Bool(purgeFlag, false, purgeFlagHelp)

	return cmd
}

func runCmdVersion(cmd *cobra.Command, args []string) {
	fmt.Println("Not implemented yet")
}
