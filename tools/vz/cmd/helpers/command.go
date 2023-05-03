// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/verrazzano/verrazzano/pkg/semver"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

// NewCommand - utility method to create cobra commands
func NewCommand(vzHelper helpers.VZHelper, use string, short string, long string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
	}

	// Configure the IO streams
	cmd.SetOut(vzHelper.GetOutputStream())
	cmd.SetErr(vzHelper.GetErrorStream())
	cmd.SetIn(vzHelper.GetInputStream())

	// Disable usage output on errors
	cmd.SilenceUsage = true
	return cmd
}

// GetWaitTimeout returns the time to wait for a command to complete
func GetWaitTimeout(cmd *cobra.Command, timeoutFlag string) (time.Duration, error) {
	// Get the wait value from the command line
	wait, err := cmd.PersistentFlags().GetBool(constants.WaitFlag)
	if err != nil {
		return time.Duration(0), err
	}
	if wait {
		timeout, err := cmd.PersistentFlags().GetDuration(timeoutFlag)
		if err != nil {
			return time.Duration(0), err
		}
		return timeout, nil
	}

	// Return duration of zero since --wait=false was specified
	return time.Duration(0), nil
}

// GetLogFormat returns the format type for streaming log output
func GetLogFormat(cmd *cobra.Command) (LogFormat, error) {
	// Get the log format value from the command line
	logFormat := cmd.PersistentFlags().Lookup(constants.LogFormatFlag)
	if logFormat == nil {
		return LogFormatSimple, nil
	}

	return LogFormat(logFormat.Value.String()), nil
}

// GetVersion returns the version of Verrazzano to install/upgrade
func GetVersion(cmd *cobra.Command, vzHelper helpers.VZHelper) (string, error) {
	// Get the version from the command line
	version, err := cmd.PersistentFlags().GetString(constants.VersionFlag)
	if err != nil {
		return "", err
	}
	if version == constants.VersionFlagDefault {
		// Find the latest release version of Verrazzano
		version, err = helpers.GetLatestReleaseVersion(vzHelper.GetHTTPClient())
		if err != nil {
			return version, err
		}
	} else {
		// Validate the version string
		installVersion, err := semver.NewSemVersion(version)
		if err != nil {
			return "", err
		}
		version = fmt.Sprintf("v%s", installVersion.ToString())
	}
	return version, nil
}

// ConfirmWithUser asks the user a yes/no question and returns true if the user answered yes, false
// otherwise.
func ConfirmWithUser(questionText string, skipQuestion bool) (bool, error) {
	if skipQuestion {
		return true, nil
	}
	var response string
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Printf("%s [Y/n]: ", questionText)
	if scanner.Scan() {
		response = scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	if response == "y" || response == "Y" {
		return true, nil
	}
	return false, nil
}

// getOperatorFileFromFlag returns the value for the operator-file option
func getOperatorFileFromFlag(cmd *cobra.Command) (string, error) {
	// Get the value from the command line
	operatorFile, err := cmd.PersistentFlags().GetString(constants.OperatorFileFlag)
	if err != nil {
		return "", fmt.Errorf("Failed to parse the command line option %s: %s", constants.OperatorFileFlag, err.Error())
	}
	return operatorFile, nil
}
