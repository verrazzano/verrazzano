// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package diff

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
	CommandName = "diff"
	helpShort   = "Diffs a chart against a given directory"
	helpLong    = `The command 'diff' diffs the contents of a chart against a directory by executing the shell diff utility and generates a patch file.`
)

func buildExample() string {
	return fmt.Sprintf(constants.CommandWithFlagExampleFormat+" "+
		constants.FlagExampleFormat+" "+
		constants.FlagExampleFormat+" "+
		constants.FlagExampleFormat,
		CommandName, constants.FlagChartName, constants.FlagChartShorthand, constants.FlagChartExampleKeycloak,
		constants.FlagVersionName, constants.FlagPatchVersionShorthand, constants.FlagVersionExample210,
		constants.FlagDirName, constants.FlagDirShorthand, constants.FlagDirExampleLocal,
		constants.FlagDiffSourceName, constants.FlagDiffSourceShorthand, constants.FlagDiffSourceExample)
}

// NewCmdDiff creates a new instance of diff cmd
func NewCmdDiff(vzHelper helpers.VZHelper, inHfs fs.ChartFileSystem) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var hfs fs.ChartFileSystem
		if inHfs == nil {
			hfs = fs.HelmChartFileSystem{}
		} else {
			hfs = inHfs
		}

		return runCmdDiff(cmd, vzHelper, hfs)
	}
	cmd.Example = buildExample()
	cmd.PersistentFlags().StringP(constants.FlagChartName, constants.FlagChartShorthand, "", constants.FlagChartUsage)
	cmd.PersistentFlags().StringP(constants.FlagVersionName, constants.FlagVersionShorthand, "", constants.FlagVersionUsage)
	cmd.PersistentFlags().StringP(constants.FlagDirName, constants.FlagDirShorthand, "", constants.FlagDirUsage)
	cmd.PersistentFlags().StringP(constants.FlagDiffSourceName, constants.FlagDiffSourceShorthand, "", constants.FlagDiffSourceUsage)
	return cmd
}

// runCmdDiff - run the "vcm diff" command to create a file containing diff of a chart with a given directory.
func runCmdDiff(cmd *cobra.Command, vzHelper helpers.VZHelper, hfs fs.ChartFileSystem) error {
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

	sourceDir, err := vcmhelpers.GetMandatoryStringFlagValueOrError(cmd, constants.FlagDiffSourceName, constants.FlagDiffSourceShorthand)
	if err != nil {
		return err
	}

	patchFile, err := hfs.GeneratePatchWithSourceDir(chartsDir, chart, version, sourceDir)
	if err != nil {
		return err
	}

	if patchFile == "" {
		fmt.Fprint(vzHelper.GetOutputStream(), "Nothing to patch.\n")
		return nil
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), "patch file generated at %s.\n", patchFile)
	return nil
}
