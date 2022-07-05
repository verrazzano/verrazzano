// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"fmt"
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	vzbugreport "github.com/verrazzano/verrazzano/tools/vz/pkg/bugreport"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"strings"
)

const (
	CommandName = "bug-report"
	helpShort   = "Capture data from the cluster"
	helpLong    = `Verrazzano command line utility to capture the data from the cluster, to report an issue`
	helpExample = `# Run bug report tool by providing the name for the report file
$vz bug-report --report-file <name of the file>
`
)

func NewCmdBugReport(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdBugReport(cmd, args, vzHelper)
	}

	cmd.Example = helpExample
	cmd.PersistentFlags().String(constants.BugReportFileFlagName, constants.BugReportFileFlagValue, constants.BugReportFileFlagUsage)
	cmd.MarkPersistentFlagRequired(constants.BugReportFileFlagName)
	return cmd
}

func runCmdBugReport(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	bugReportFile, err := cmd.PersistentFlags().GetString(constants.BugReportFileFlagName)
	if err != nil {
		return fmt.Errorf("error fetching flag: %s", err.Error())
	}

	// Validate the report file format
	if !strings.HasSuffix(bugReportFile, constants.BugReportFileExtn) {
		return fmt.Errorf("unsupported report-file: %s, set a .tar.gz file", bugReportFile)
	}

	// Get the kubernetes clientset, which will validate that the kubeconfig and context are valid.
	kubeClient, err := vzHelper.GetKubeClient(cmd)
	if err != nil {
		return err
	}

	// Get the controller runtime client
	client, err := vzHelper.GetClient(cmd)
	if err != nil {
		return err
	}

	// Generate the bug report
	return vzbugreport.GenerateBugReport(kubeClient, client, bugReportFile, vzHelper)
}
