package sanitize

// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

import (
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	flagErrorStr = "error fetching flag: %s"
	CommandName  = "sanitize"
	helpShort    = "Sanitize information from an existing cluster snapshot"
	helpLong     = "sanitize function"
)

type directoryAndTarValidationStruct struct {
	directory  string
	tarFile    string
	reportFile string
	isVerbose  bool
}

func NewCmdSanitize(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Hidden = true

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdSanitize(cmd, args, vzHelper)
	}

	cmd.PersistentFlags().String(constants.InputDirectoryFlagName, constants.InputDirectoryFlagValue, constants.InputDirectoryFlagUsage)
	cmd.PersistentFlags().String(constants.OutputDirectoryFlagName, constants.OutputDirectoryFlagValue, constants.OutputDirectoryFlagUsage)
	cmd.PersistentFlags().String(constants.InputTarFileFlagName, constants.InputTarFileFlagValue, constants.InputTarFileFlagUsage)
	cmd.PersistentFlags().String(constants.OutputTarGZFileFlagName, constants.OutputTarGZFileFlagValue, constants.OutputTarGZFileFlagUsage)

	// Verifies that the CLI args are not set at the creation of a command
	vzHelper.VerifyCLIArgsNil(cmd)

	return cmd
}
func runCmdSanitize(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	return nil

}
