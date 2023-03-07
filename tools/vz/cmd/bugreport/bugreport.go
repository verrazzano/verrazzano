// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
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
	"strings"
	"time"
)

const (
	CommandName = "bug-report"
	helpShort   = "Collect information from the cluster to report an issue"
	helpLong    = `Verrazzano command line utility to collect data from the cluster, to report an issue`
	helpExample = `
# Create a bug report file, bugreport.tar.gz, by collecting data from the cluster:
vz bug-report --report-file bugreport.tar.gz

When --report-file is not provided, the command creates bug-report.tar.gz in the current directory.

# Create a bug report file, bugreport.tar.gz, including the additional namespace ns1 from the cluster:
vz bug-report --report-file bugreport.tgz --include-namespaces ns1

# The flag --include-namespaces accepts comma-separated values and can be specified multiple times. For example, the following commands create a bug report by including additional namespaces ns1, ns2, and ns3:
   a. vz bug-report --report-file bugreport.tgz --include-namespaces ns1,ns2,ns3
   b. vz bug-report --report-file bugreport.tgz --include-namespaces ns1,ns2 --include-namespaces ns3

The values specified for the flag --include-namespaces are case-sensitive.

# Use the --include-logs flag to collect the logs from the pods in one or more namespaces, by specifying the --include-namespaces flag.
vz bug-report --report-file bugreport.tgz --include-namespaces ns1,ns2 --include-logs

# The flag --duration collects logs for a specific period. The default value is 0, which collects the complete pod log. It supports seconds, minutes, and hours.
   a. vz bug-report --report-file bugreport.tgz --include-namespaces ns1 --include-logs --duration 3h
   b. vz bug-report --report-file bugreport.tgz --include-namespaces ns1,ns2 --include-logs --duration 5m
   c. vz bug-report --report-file bugreport.tgz --include-namespaces ns1,ns2 --include-logs --duration 300s
`
)

const minLineLength = 100

var kubeconfigFlagValPointer string
var contextFlagValPointer string

func NewCmdBugReport(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return RunCmdBugReport(cmd, vzHelper)
	}

	cmd.Example = helpExample
	cmd.PersistentFlags().StringP(constants.BugReportFileFlagName, constants.BugReportFileFlagShort, constants.BugReportFileFlagValue, constants.BugReportFileFlagUsage)
	cmd.PersistentFlags().StringSliceP(constants.BugReportIncludeNSFlagName, constants.BugReportIncludeNSFlagShort, []string{}, constants.BugReportIncludeNSFlagUsage)
	cmd.PersistentFlags().BoolP(constants.VerboseFlag, constants.VerboseFlagShorthand, constants.VerboseFlagDefault, constants.VerboseFlagUsage)
	cmd.PersistentFlags().BoolP(constants.BugReportLogFlagName, constants.BugReportLogFlagNameShort, constants.BugReportLogFlagDefault, constants.BugReportLogFlagNameUsage)
	cmd.PersistentFlags().DurationP(constants.BugReportTimeFlagName, constants.BugReportTimeFlagNameShort, constants.BugReportTimeFlagDefaultTime, constants.BugReportTimeFlagNameUsage)
	return cmd
}

func RunCmdBugReport(cmd *cobra.Command, vzHelper helpers.VZHelper) error {
	start := time.Now()
	bugReportFile, err := getBugReportFile(cmd, vzHelper)
	if err != nil {
		return fmt.Errorf("error fetching flag: %s", err.Error())
	}

	// Get the kubernetes clientset, which will validate that the kubeconfigFlagValPointer and contextFlagValPointer are valid.
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
	// If additional namespaces pods logs needs to be capture using flag --include-logs
	isPodLog, err := cmd.PersistentFlags().GetBool(constants.BugReportLogFlagName)
	if err != nil {
		return fmt.Errorf("an error occurred while reading values for the flag --include-logs: %s", err.Error())
	}

	// If additional namespaces pods logs needs to be capture using flag with duration --duration
	durationString, err := cmd.PersistentFlags().GetDuration(constants.BugReportTimeFlagName)
	if err != nil {
		return fmt.Errorf("an error occurred while reading values for the flag --duration: %s", err.Error())
	}
	durationValue := int64(durationString.Seconds())
	if err != nil {
		return fmt.Errorf("an error occurred,invalid value --duration: %s", err.Error())
	}
	if durationValue < 0 {
		return fmt.Errorf("an error occurred, invalid duration can't be less than 1s: %d", durationValue)
	}

	// Create a temporary directory to place the cluster data
	bugReportDir, err := os.MkdirTemp("", constants.BugReportDir)
	if err != nil {
		return fmt.Errorf("an error occurred while creating the directory to place cluster resources: %s", err.Error())
	}
	defer os.RemoveAll(bugReportDir)

	// set the flag to control the display the resources captured
	isVerbose, err := cmd.PersistentFlags().GetBool(constants.VerboseFlag)
	if err != nil {
		return fmt.Errorf("an error occurred while reading value for the flag %s: %s", constants.VerboseFlag, err.Error())
	}
	helpers.SetVerboseOutput(isVerbose)

	// Capture cluster snapshot
	err = vzbugreport.CaptureClusterSnapshot(kubeClient, dynamicClient, client, bugReportDir, moreNS, vzHelper, vzbugreport.PodLogs{IsPodLog: isPodLog, Duration: durationValue})
	if err != nil {
		os.Remove(bugReportFile)
		return fmt.Errorf(err.Error())
	}

	// Return an error when the command fails to collect anything from the cluster
	// There will be bug-report.out and bug-report.err in bugReportDir, ignore them
	if isDirEmpty(bugReportDir, 2) {
		return fmt.Errorf("The bug-report command did not collect any file from the cluster. " +
			"Please go through errors (if any), in the standard output.\n")
	}

	// Generate the bug report
	err = helpers.CreateReportArchive(bugReportDir, bugRepFile)
	if err != nil {
		return fmt.Errorf("there is an error in creating the bug report, %s", err.Error())
	}

	brf, _ := os.Stat(bugReportFile)
	if brf.Size() > 0 {
		msg := fmt.Sprintf("Created bug report: %s in %s\n", bugReportFile, time.Since(start))
		fmt.Fprintf(vzHelper.GetOutputStream(), msg)
		// Display a message to check the standard error, if the command reported any error and continued
		if helpers.IsErrorReported() {
			fmt.Fprintf(vzHelper.GetOutputStream(), constants.BugReportError+"\n")
		}
		displayWarning(msg, vzHelper)
	} else {
		// Verrazzano is not installed, remove the empty bug report file
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
		return filepath.Join(currentDir, constants.BugReportFileDefaultValue), nil
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

// displayWarning logs a warning message to check the contents of the bug report
func displayWarning(successMessage string, helper helpers.VZHelper) {
	// This might be the efficient way, but does the job of displaying a formatted message

	// Draw a line to differentiate the warning from the info message
	count := len(successMessage)
	if len(successMessage) < minLineLength {
		count = minLineLength
	}
	sep := strings.Repeat(constants.LineSeparator, count)

	// Any change in BugReportWarning, requires a change here to adjust the whitespace characters before the message
	wsCount := count - len(constants.BugReportWarning)

	fmt.Fprintf(helper.GetOutputStream(), sep+"\n")
	fmt.Fprintf(helper.GetOutputStream(), strings.Repeat(" ", wsCount/2)+constants.BugReportWarning+"\n")
	fmt.Fprintf(helper.GetOutputStream(), sep+"\n")
}

// isDirEmpty returns whether the directory is empty or not, ignoring ignoreFilesCount number of files
func isDirEmpty(directory string, ignoreFilesCount int) bool {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return false
	}
	return len(entries) == ignoreFilesCount
}

func CallVzBugReport(cmd *cobra.Command, vzHelper helpers.VZHelper, err error) error {
	autoBugReportFlag, errFlag := cmd.Flags().GetBool(constants.AutoBugReportFlag)
	if errFlag != nil {
		fmt.Fprintf(vzHelper.GetOutputStream(), "Error fetching flags: %s", errFlag.Error())
	}
	// if vz cli command returned an err and auto-bug-report is set to true
	if autoBugReportFlag {
		cmd2 := NewCmdBugReport(vzHelper)
		kubeconfigFlag, errFlag := cmd.Flags().GetString(constants.GlobalFlagKubeConfig)
		if errFlag != nil {
			fmt.Fprintf(vzHelper.GetOutputStream(), "Error fetching flags: %s", errFlag.Error())
		}
		contextFlag, errFlag2 := cmd.Flags().GetString(constants.GlobalFlagContext)
		if errFlag2 != nil {
			fmt.Fprintf(vzHelper.GetOutputStream(), "Error fetching flags: %s", errFlag2.Error())
		}
		cmd2.Flags().StringVar(&kubeconfigFlagValPointer, constants.GlobalFlagKubeConfig, "", constants.GlobalFlagKubeConfigHelp)
		cmd2.Flags().StringVar(&contextFlagValPointer, constants.GlobalFlagContext, "", constants.GlobalFlagContextHelp)
		cmd2.Flags().Set(constants.GlobalFlagKubeConfig, kubeconfigFlag)
		cmd2.Flags().Set(constants.GlobalFlagContext, contextFlag)
		bugReportErr := RunCmdBugReport(cmd2, vzHelper)
		if bugReportErr != nil {
			fmt.Fprintf(vzHelper.GetOutputStream(), "Error calling vz bug-report %s \n", bugReportErr.Error())
		}
	}
	// return original error from running vz command that was passed in
	return err
}
