// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/templates"
	capiversion "sigs.k8s.io/cluster-api/version"
)

var cliVersion string
var buildDate string
var gitCommit string

const (
	CommandName = "version"
	helpShort   = "PSR CLI version information"
	helpLong    = `The command 'version' reports information about the version of the psrctl tool being run`
	helpExample = `psrctl version`
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
		return runCmdVersion(vzHelper)
	}

	return cmd
}

func runCmdVersion(vzHelper helpers.VZHelper) error {

	templateValues := map[string]string{
		"cli_version": cliVersion,
		"build_date":  buildDate,
		"git_commit":  gitCommit,
	}

	result, err := templates.ApplyTemplate(versionOutputTemplate, templateValues)
	if err != nil {
		return fmt.Errorf("Failed to generate %s command output: %s", CommandName, err.Error())
	}
	_, _ = fmt.Fprintf(vzHelper.GetOutputStream(), result)

	// Put in cluster-api package for testing purposes
	_ = capiversion.Get()

	return nil
}

func GetEffectiveDocsVersion() string {
	if os.Getenv("USE_V8O_DOC_STAGE") == "true" || len(cliVersion) == 0 {
		return "devel"
	}
	var re = regexp.MustCompile(`(?m)(\d.\d)(.*)`)
	s := re.FindAllStringSubmatch(cliVersion, -1)[0][1] //This will get the group 1 of 1st match which is "1.4.0" to "1.4"
	return fmt.Sprintf("v%s", s)                        //return v1.4 by appending prefex 'v'
}

func GetCLIVersion() string {
	return cliVersion
}
