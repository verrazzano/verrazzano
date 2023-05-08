// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package root

import (
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/diff"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/patch"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/cmd/pull"
	"github.com/verrazzano/verrazzano/tools/charts-manager/vcm/pkg/fs"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "vcm"
	helpShort   = "The vcm tool is a command-line utility that enables developers to pull and customize helm charts."
)

// NewRootCmd - create the root cobra command
func NewRootCmd(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpShort)
	hfs := fs.HelmChartFileSystem{}
	// Add commands
	cmd.AddCommand(pull.NewCmdPull(vzHelper, hfs, nil))
	cmd.AddCommand(diff.NewCmdDiff(vzHelper, hfs))
	cmd.AddCommand(patch.NewCmdPatch(vzHelper, hfs))
	return cmd
}
