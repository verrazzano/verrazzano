// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package bugreport

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	vzbugreport "github.com/verrazzano/verrazzano/tools/vz/pkg/bugreport"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

const (
	CommandName = "bug-report"
	helpShort   = "Capture data from the cluster"
	helpLong    = `Verrazzano command line utility to capture the data from the cluster, to report an issue`
	helpExample = `# Run bug report tool by providing the name for the report file
$vz bug-report --report-file <name of the file to include cluster data, a .tar.gz or .tgz file> --include-namespaces <one or more application namespaces to collect information>
`
)

func NewCmdBugReport(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdBugReport(cmd, args, vzHelper)
	}

	cmd.Example = helpExample
	cmd.PersistentFlags().StringP(constants.BugReportFileFlagName, constants.BugReportFileFlagShort, constants.BugReportFileFlagValue, constants.BugReportFileFlagUsage)
	cmd.PersistentFlags().StringP(constants.BugReportIncludeNSFlagName, constants.BugReportIncludeNSFlagShort, constants.BugReportIncludeNSFlagValue, constants.BugReportIncludeNSFlagUsage)
	cmd.MarkPersistentFlagRequired(constants.BugReportFileFlagName)

	return cmd
}

func runCmdBugReport(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	start := time.Now()
	bugReportFile, err := cmd.PersistentFlags().GetString(constants.BugReportFileFlagName)
	if err != nil {
		return fmt.Errorf("error fetching flag: %s", err.Error())
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

	// Get the dynamic client to retrieve OAM resources
	dynamicClient, err := vzHelper.GetDynamicClient(cmd)
	if err != nil {
		return err
	}

	// Check whether the file already exists
	err = checkExistingFile(bugReportFile)
	if err != nil {
		return err
	}

	// Create the bug report file
	bugRepFile, err := os.Create(bugReportFile)
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return fmt.Errorf("permission denied to create the bug report: %s", bugReportFile)
		}
		return fmt.Errorf("an error occurred while creating %s: %s", bugReportFile, err.Error())
	}
	defer bugRepFile.Close()

	// Read and parse the additional namespaces provided using --include-namespaces
	includeNSFlag := cmd.PersistentFlags().Lookup(constants.BugReportIncludeNSFlagName)
	moreNS := ""
	if includeNSFlag != nil {
		moreNS = includeNSFlag.Value.String()
	}

	// Generate the bug report
	err = vzbugreport.GenerateBugReport(kubeClient, dynamicClient, client, bugRepFile, moreNS, vzHelper)
	if err != nil {
		os.Remove(bugReportFile)
		return fmt.Errorf(err.Error())
	}

	fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Successfully created the bug report: %s in %s\n", bugReportFile, time.Since(start)))
	fmt.Fprintf(vzHelper.GetOutputStream(), "Please go through errors (if any), in the standard output.\n")

	// Display a warning message to review the contents of the report
	fmt.Fprint(vzHelper.GetOutputStream(), "WARNING: Please examine the contents of the bug report for sensitive data.\n")
	return nil
}

// checkExistingFile determines whether a file / directory with the name bugReportFile already exists
func checkExistingFile(bugReportFile string) error {
	// Fail if the bugReportFile already exists or is a directory
	fileInfo, err := os.Stat(bugReportFile)
	if fileInfo != nil {
		if fileInfo.IsDir() {
			return fmt.Errorf("%s is an existing directory", bugReportFile)
		}
		return fmt.Errorf("file %s already exists", bugReportFile)
	}

	// check if the parent directory exists
	if err != nil {
		fi, fe := os.Stat(filepath.Dir(bugReportFile))
		if fi != nil {
			if !fi.IsDir() {
				return fmt.Errorf("%s is not a directory", filepath.Dir(bugReportFile))
			}
		}
		if fe != nil {
			return fmt.Errorf("an error occurred while creating %s: %s", bugReportFile, fe.Error())
		}
	}
	return nil
}
