// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "install"
	helpShort   = "Install Verrazzano"
	helpLong    = `Install the Verrazzano Platform Operator and install the Verrazzano components specified by the Verrazzano CR provided on the command line.`
	helpExample = `
# Install the latest version of Verrazzano using the prod profile. Stream the logs to the console until the install completes.
vz install --logs

# Install version 1.3.0 using a dev profile, timeout the command after 20 minutes.
vz install --version v1.3.0 --set profile=dev --wait --timeout 20m

# Install version 1.3.0 using a dev profile with elasticsearch disabled and wait for the install to complete.
vz install --version v1.3.0 --set profile=dev --set components.elasticsearch.enabled=false --wait

# Install the latest version of Verrazzano using CR overlays and explicit value sets.  Output the logs in json format.
vz install -f base.yaml -f custom.yaml --set profile=prod --logs json`
)

var logsEnum = cmdhelpers.LogsFormatSimple

func NewCmdInstall(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdInstall(cmd, args, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, constants.WaitFlagDefault, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().String(constants.VersionFlag, "latest", constants.VersionFlagHelp)
	cmd.PersistentFlags().StringSliceP(constants.FilenameFlag, constants.FilenameFlagShorthand, []string{}, constants.FilenameFlagHelp)
	cmd.PersistentFlags().Var(&logsEnum, constants.LogFormatFlag, constants.LogFormatHelp)
	cmd.PersistentFlags().StringArrayP(constants.SetFlag, constants.SetFlagShorthand, []string{}, constants.SetFlagHelp)

	// Initially the operator-file flag may be for internal use, hide from help until
	// a decision is made on supporting this option.
	cmd.PersistentFlags().String(constants.OperatorFileFlag, "", constants.OperatorFileFlagHelp)
	cmd.PersistentFlags().MarkHidden(constants.OperatorFileFlag)

	// Dry run flag is still being discussed - keep hidden for now
	cmd.PersistentFlags().Bool(constants.DryRunFlag, false, "Simulate an install.")
	cmd.PersistentFlags().MarkHidden(constants.DryRunFlag)

	return cmd
}

func runCmdInstall(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	fmt.Fprintf(vzHelper.GetOutputStream(), "Not implemented yet\n")
	return nil
}
