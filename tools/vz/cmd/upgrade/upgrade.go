// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upgrade

import (
	"context"
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
	helpLong    = `Upgrade the Verrazzano Platform Operator to the specified version and update all of the currently installed components`
	helpExample = `
# Upgrade to the latest version of Verrazzano and wait for the command to complete.  Stream the logs to the console until the upgrade completes.
vz upgrade

# Upgrade to Verrazzano v1.3.0, stream the logs to the console and timeout after 20m
vz upgrade --version v1.3.0 --timeout 20m`
)

var logsEnum = cmdhelpers.LogFormatSimple

func NewCmdUpgrade(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdUpgrade(cmd, args, vzHelper)
	}
	cmd.Example = helpExample

	cmd.PersistentFlags().Bool(constants.WaitFlag, constants.WaitFlagDefault, constants.WaitFlagHelp)
	cmd.PersistentFlags().Duration(constants.TimeoutFlag, time.Minute*30, constants.TimeoutFlagHelp)
	cmd.PersistentFlags().String(constants.VersionFlag, constants.VersionFlagDefault, constants.VersionFlagUpgradeHelp)
	cmd.PersistentFlags().Var(&logsEnum, constants.LogFormatFlag, constants.LogFormatHelp)

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
	// Get the controller runtime client
	client, err := vzHelper.GetClient(cmd)
	if err != nil {
		return err
	}

	// Get the verrazzano resource that needs to updated for the new version
	vz, err := helpers.FindVerrazzanoResource(client)
	if err != nil {
		return err
	}

	// Get the timeout value for the upgrade command
	_, err = cmdhelpers.GetWaitTimeout(cmd)
	if err != nil {
		return err
	}

	// Get the log format value
	_, err = cmdhelpers.GetLogFormat(cmd)
	if err != nil {
		return err
	}

	// Get the kubernetes clientset.  This will validate that the kubeconfig and context are valid.
	_, err = vzHelper.GetKubeClient(cmd)
	if err != nil {
		return err
	}

	// Get the Verrazzano version we are upgrading to
	version, err := cmdhelpers.GetVersion(cmd, vzHelper)
	if err != nil {
		return err
	}

	// Show the version of Verrazzano we are upgrading to
	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Upgrading Verrazzano to version %s\n", version))

	// Apply the Verrazzano operator.yaml.
	err = cmdhelpers.ApplyPlatformOperatorYaml(cmd, client, vzHelper, version)
	if err != nil {
		return err
	}

	// Wait for the platform operator to be ready before we create the Verrazzano resource.
	_, err = cmdhelpers.WaitForPlatformOperator(client, vzHelper)
	if err != nil {
		return err
	}

	// Update the version in verrazzano install resource
	vz.Spec.Version = version
	err = client.Update(context.TODO(), vz)
	if err != nil {
		return fmt.Errorf("Failed to set upgrade version in verrazzano resource: %s", err.Error())
	}

	return nil
}
