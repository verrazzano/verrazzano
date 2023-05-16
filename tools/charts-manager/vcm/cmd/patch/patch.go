// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package patch

import (
	"fmt"

	"github.com/spf13/cobra"
	vcmhelpers "github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/fs"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "patch"
	helpShort   = "Patches a chart against a given patch file."
	helpLong    = `The command 'patch' applies changes from a patch file to a chart by executing the shell patch command.`
)

func buildExample() string {
	return fmt.Sprintf(constants.CommandWithFlagExampleFormat+" "+
		constants.FlagExampleFormat+" "+
		constants.FlagExampleFormat+" "+
		constants.FlagExampleFormat,
		CommandName, constants.FlagChartName, constants.FlagChartShorthand, constants.FlagChartExampleKeycloak,
		constants.FlagVersionName, constants.FlagPatchVersionShorthand, constants.FlagVersionExample210,
		constants.FlagDirName, constants.FlagDirShorthand, constants.FlagDirExampleLocal,
		constants.FlagPatchFileName, constants.FlagPatchFileShorthand, constants.FlagPatchFileExample)
}

// NewCmdPatch creates a new instance of patch cmd
func NewCmdPatch(vzHelper helpers.VZHelper, inHfs fs.ChartFileSystem) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var hfs fs.ChartFileSystem
		if inHfs == nil {
			hfs = fs.HelmChartFileSystem{}
		} else {
			hfs = inHfs
		}

		return runCmdPatch(cmd, vzHelper, hfs)
	}
	cmd.Example = buildExample()
	cmd.PersistentFlags().StringP(constants.FlagChartName, constants.FlagChartShorthand, "", constants.FlagChartUsage)
	cmd.PersistentFlags().StringP(constants.FlagVersionName, constants.FlagVersionShorthand, "", constants.FlagVersionUsage)
	cmd.PersistentFlags().StringP(constants.FlagDirName, constants.FlagDirShorthand, "", constants.FlagDirUsage)
	cmd.PersistentFlags().StringP(constants.FlagPatchFileName, constants.FlagPatchFileShorthand, "", constants.FlagPatchFileUsage)
	return cmd
}

// runCmdPatch - run the "vcm patch" command to apply a given patch on a chart.
func runCmdPatch(cmd *cobra.Command, vzHelper helpers.VZHelper, hfs fs.ChartFileSystem) error {
	chart, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagChartName, constants.FlagChartShorthand)
	if err != nil {
		return err
	}

	version, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagVersionName, constants.FlagVersionShorthand)
	if err != nil {
		return err
	}

	chartsDir, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagDirName, constants.FlagDirShorthand)
	if err != nil {
		return err
	}

	patchFile, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagPatchFileName, constants.FlagPatchFileShorthand)
	if err != nil {
		return err
	}

	_, err = hfs.ApplyPatchFile(chartsDir, vzHelper, chart, version, patchFile)
	if err != nil {
		return err
	}

	return nil
}
