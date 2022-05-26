// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName  = "install"
	helpShort    = "Install Verrazzano"
	helpLong     = `Install the Verrazzano Platform Operator and install the Verrazzano components specified by the Verrazzano CR provided on the command line.`
	helpExamples = `
vz install --version v1.3.0 --wait --timeout 20m
vz install --version v1.3.0 --dry-run
vz install --version v1.3.0 --logs
`
)

func NewCmdInstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Run = runCmdVersion
	cmd.Example = helpExamples

	cmd.PersistentFlags().Bool(constants.WaitFlag, false, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*20, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().String(constants.VersionFlag, "latest", constants.VersionFlagHelp)
	cmd.PersistentFlags().StringSlice(constants.FilenameFlag, []string{}, constants.FilenameFlagHelp)
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an install")
	cmd.PersistentFlags().String(constants.OperatorFileFlag, "", constants.OperatorFileFlagHelp)
	cmd.PersistentFlags().Bool(constants.LogsFlag, false, constants.LogsFlagHelp)

	return cmd
}

func runCmdVersion(cmd *cobra.Command, args []string) {
	fmt.Println("Not implemented yet")
}
