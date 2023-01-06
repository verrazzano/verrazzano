// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package version

import (
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/psr/psrctl/cmd/constants"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/templates"
	"os"
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
Version:     v{{.cli_version}}
BuildDate:   {{.build_date}}
GitCommit:   {{.git_commit}}
WorkerImage: {{.worker_image}}
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
		"cli_version":  cliVersion,
		"build_date":   buildDate,
		"git_commit":   gitCommit,
		"worker_image": constants.GetDefaultWorkerImage(),
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
	return os.Getenv("USE_V8O_DOC_STAGE")
}

func GetCLIVersion() string {
	return cliVersion
}
