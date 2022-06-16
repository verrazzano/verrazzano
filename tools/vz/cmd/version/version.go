// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"fmt"
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/templates"
)

var cliVersion string
var buildDate string
var gitCommit string

const (
	CommandName = "version"
	helpShort   = "Verrazzano version information"
	helpLong    = `The command 'version' reports information about the version of the vz tool being run`
	helpExample = `vz version`
)

// statusOutputTemplate - template for output of status command
const versionOutputTemplate = `
Version: v{{.cli_version}}
BuildDate: {{.build_date}}
GitCommit: {{.git_commit}}
`

func NewCmdVersion(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Example = helpExample
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdVersion(cmd, args, vzHelper)
	}

	return cmd
}

func runCmdVersion(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {

	templateValues := map[string]string{
		"cli_version": cliVersion,
		"build_date":  buildDate,
		"git_commit":  gitCommit,
	}

	result, err := templates.ApplyTemplate(versionOutputTemplate, templateValues)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), result)
	return nil
}
