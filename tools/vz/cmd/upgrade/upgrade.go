// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upgrade

import (
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName  = "upgrade"
	helpShort    = "Upgrade Verrazzano"
	helpLong     = `Upgrade the Verrazzano Platform Operator to the specified version and update all of the currently installed components.`
	helpExamples = `
TBD
`
)

func NewCmdUpgrade(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Run = runCmdVersion
	cmd.Example = helpExamples

	cmd.PersistentFlags().Bool(constants.WaitFlag, false, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*20, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().String(constants.VersionFlag, "latest", constants.VersionFlagHelp)
	cmd.PersistentFlags().StringSlice(constants.FilenameFlag, []string{}, constants.FilenameFlagHelp)
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an upgrade")
	cmd.PersistentFlags().String(constants.OperatorFileFlag, "", constants.OperatorFileFlagHelp)
	cmd.PersistentFlags().Bool(constants.LogsFlag, false, constants.LogsFlagHelp)

	return cmd
}

func runCmdVersion(cmd *cobra.Command, args []string) {
	fmt.Println("Not implemented yet")
}
