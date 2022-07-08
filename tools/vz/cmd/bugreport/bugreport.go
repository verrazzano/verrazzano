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
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	CommandName = "bug-report"
	helpShort   = "Capture data from the cluster"
	helpLong    = `Verrazzano command line utility to capture the data from the cluster, to report an issue`
	helpExample = `# Run bug report tool by providing the name for the report file
$vz bug-report --report-file <name of the file to include cluster data, a .tar.gz or .tgz file> --include <one or more application namespaces to collect information>
`
)

func NewCmdBugReport(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdBugReport(cmd, args, vzHelper)
	}

	cmd.Example = helpExample
	cmd.PersistentFlags().String(constants.BugReportFileFlagName, constants.BugReportFileFlagValue, constants.BugReportFileFlagUsage)
	cmd.PersistentFlags().String(constants.BugReportIncludeFlagName, constants.BugReportFileFlagValue, constants.BugReportFileFlagUsage
	cmd.MarkPersistentFlagRequired(constants.BugReportFileFlagName)

	return cmd
}

func runCmdBugReport(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
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

	includeFlag := cmd.PersistentFlags().Lookup(constants.BugReportIncludeFlagName)
	includes := ""
	if includeFlag == nil {
		includes = includeFlag.Value.String()
	}

	// Generate the bug report
	err = vzbugreport.GenerateBugReport(kubeClient, client, bugRepFile, includes, vzHelper)
	if err != nil {
		os.Remove(bugReportFile)
	}
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
