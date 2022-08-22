package create

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/capi"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	clusterSubCommandName = "cluster"
	clusterHelpShort      = "Verrazzano create cluster"
	clusterHelpLong       = `The command 'create cluster' provisions a new local Kind cluster`
	clusterHelpExample    = `vz create cluster ???`
)

func newSubcmdCluster(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, clusterSubCommandName, clusterHelpShort, clusterHelpLong)
	cmd.Example = clusterHelpExample
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdCreateCluster(vzHelper)
	}

	return cmd
}

func runCmdCreateCluster(vzHelper helpers.VZHelper) error {

	// templateValues := map[string]string{
	// 	"cli_version": cliVersion,
	// 	"build_date":  buildDate,
	// 	"git_commit":  gitCommit,
	// }

	// result, err := templates.ApplyTemplate(versionOutputTemplate, templateValues)
	// var err error
	// if err != nil {
	// 	return fmt.Errorf("Failed to generate %s command output: %s", CommandName, err.Error())
	// }
	// _, _ = fmt.Fprintf(vzHelper.GetOutputStream(), result)
	//
	// // Put in cluster-api package for testing purposes
	// _ = capiversion.Get()

	fmt.Println("DEVA create cluster command invoked")
	return capi.NewBoostrapCluster().Create()
}
