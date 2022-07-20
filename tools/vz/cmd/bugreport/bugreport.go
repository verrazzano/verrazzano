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
	helpShort   = "Collect information from the cluster to report an issue"
	helpLong    = `Verrazzano command line utility to collect data from the cluster, to report an issue`
	helpExample = `
# Create a bug report bugreport.tar.gz by collecting data from the cluster
vz bug-report --report-file bugreport.tar.gz

When the --report-file is not provided, the command attempts to create bug-report.tar.gz in the current directory.

# Create a bug report bugreport.tgz, including additional namespace ns1 from the cluster
vz bug-report --report-file bugreport.tgz --include-namespaces ns1

The flag --include-namespaces accepts comma separated values. The flag can also be specified multiple times.
For example, the following commands create a bug report by including additional namespaces ns1, ns2 and ns3
   a. vz bug-report --report-file bugreport.tgz --include-namespaces ns1,ns2,ns3
   b. vz bug-report --report-file bugreport.tgz --include-namespaces ns1,ns2 --include-namespaces ns3

The values specified for the flag --include-namespaces are case-sensitive.
`
)

func NewCmdBugReport(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdBugReport(cmd, args, vzHelper)
	}

	cmd.Example = helpExample
	cmd.PersistentFlags().StringP(constants.BugReportFileFlagName, constants.BugReportFileFlagShort, constants.BugReportFileFlagValue, constants.BugReportFileFlagUsage)
	cmd.PersistentFlags().StringSliceP(constants.BugReportIncludeNSFlagName, constants.BugReportIncludeNSFlagShort, []string{}, constants.BugReportIncludeNSFlagUsage)

	return cmd
}

func runCmdBugReport(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	start := time.Now()
	bugReportFile, err := getBugReportFile(cmd, vzHelper)
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

	// Read the additional namespaces provided using flag --include-namespaces
	moreNS, err := cmd.PersistentFlags().GetStringSlice(constants.BugReportIncludeNSFlagName)
	if err != nil {
		return fmt.Errorf("an error occurred while reading values for the flag --include-namespaces: %s", err.Error())
	}

	// Generate the bug report
	err = vzbugreport.GenerateBugReport(kubeClient, dynamicClient, client, bugRepFile, moreNS, vzHelper)
	if err != nil {
		os.Remove(bugReportFile)
		return fmt.Errorf(err.Error())
	}

	brf, _ := os.Stat(bugReportFile)
	if brf.Size() > 0 {
		fmt.Fprintf(vzHelper.GetOutputStream(), fmt.Sprintf("Successfully created the bug report: %s in %s\n", bugReportFile, time.Since(start)))

		// TODO: Display a message to look at the standard err, if the command reported any error

		// Display a warning message to review the contents of the report
		fmt.Fprint(vzHelper.GetOutputStream(), "WARNING: Please examine the contents of the bug report for sensitive data.\n")
	} else {
		// When Verrazzano is not installed, remove the empty bug report file
		os.Remove(bugReportFile)
	}
	return nil
}

// getBugReportFile determines the bug report file
func getBugReportFile(cmd *cobra.Command, vzHelper helpers.VZHelper) (string, error) {
	bugReport, err := cmd.PersistentFlags().GetString(constants.BugReportFileFlagName)
	if err != nil {
		return "", fmt.Errorf("error fetching flag: %s", err.Error())
	}
	if bugReport == "" {
		currentDir, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("error determining the current directory: %s", err.Error())
		}
		return currentDir + string(os.PathSeparator) + constants.BugReportFileDefaultValue, nil
	}
	return bugReport, nil
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
