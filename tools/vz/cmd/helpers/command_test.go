// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"fmt"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	testhelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
)

const (
	testUse              = "A sample usage help"
	testShort            = "A sample short help"
	testLong             = "A sample long help"
	flagNotDefinedErrFmt = "flag accessed but not defined: %s"
)

// TestNewCommand tests the functionality to create a new command based on the usage, short and log help.
func TestNewCommand(t *testing.T) {
	rc := testhelpers.NewFakeRootCmdContextWithFiles(t)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	assert.NotNil(t, NewCommand(rc, testUse, testShort, testLong))
}

// TestGetWaitTimeout tests the functionality that returns the wait timeout if set or returns the default timeout value.
func TestGetWaitTimeout(t *testing.T) {
	// GIVEN a command with no values provided for wait and timeout flags,
	// WHEN we get the wait timeout value,
	// THEN an error along with default timeout value of (0) is returned.
	timeout, err := GetWaitTimeout(getCommandWithoutFlags(), constants.TimeoutFlag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf(flagNotDefinedErrFmt, constants.WaitFlag))
	assert.NotNil(t, timeout)
	assert.Equal(t, time.Duration(0), timeout)

	// GIVEN a command with a value of true for wait and no value provided for timeout flag,
	// WHEN we get the wait timeout value,
	// THEN an error along with default timeout value of (0) is returned.
	cmdWithWaitFlagTrue := getCommandWithoutFlags()
	cmdWithWaitFlagTrue.PersistentFlags().Bool(constants.WaitFlag, true, "")
	timeout, err = GetWaitTimeout(cmdWithWaitFlagTrue, constants.TimeoutFlag)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf(flagNotDefinedErrFmt, constants.TimeoutFlag))
	assert.NotNil(t, timeout)
	assert.Equal(t, time.Duration(0), timeout)

	// GIVEN a command with a value of false for wait and no value provided for timeout flag,
	// WHEN we get the wait timeout value,
	// THEN an error along with default timeout value of (0) is returned.
	cmdWithWaitFlagFalse := getCommandWithoutFlags()
	cmdWithWaitFlagFalse.PersistentFlags().Bool(constants.WaitFlag, false, "")
	timeout, err = GetWaitTimeout(cmdWithWaitFlagFalse, constants.TimeoutFlag)
	assert.NoError(t, err)
	assert.NotNil(t, timeout)
	assert.Equal(t, time.Duration(0), timeout)

	// GIVEN a command with a value of true for wait and value provided for timeout flag,
	// WHEN we get the wait timeout value,
	// THEN an error along with default timeout value of (0) is returned.
	cmdWithWaitFlagAndTimeout := getCommandWithoutFlags()
	cmdWithWaitFlagAndTimeout.PersistentFlags().Bool(constants.WaitFlag, true, "")
	cmdWithWaitFlagAndTimeout.PersistentFlags().Duration(constants.TimeoutFlag, time.Duration(1), "")
	timeout, err = GetWaitTimeout(cmdWithWaitFlagAndTimeout, constants.TimeoutFlag)
	assert.NoError(t, err)
	assert.NotNil(t, timeout)
	assert.Equal(t, time.Duration(1), timeout)
}

// TestGetLogFormat tests the functionality that returns the log format that was set by the user.
func TestGetLogFormat(t *testing.T) {
	// GIVEN a command with no value provided for the log format flag,
	// WHEN we get the log format,
	// THEN the default log pattern is returned.
	logFormat, err := GetLogFormat(getCommandWithoutFlags())
	assert.NoError(t, err)
	assert.NotNil(t, logFormat)
	assert.Equal(t, LogFormatSimple, logFormat)

	// GIVEN a command with value custom provided for the log format flag,
	// WHEN we get the log format,
	// THEN the custom log pattern is returned.
	cmdWithLogFormat := getCommandWithoutFlags()
	cmdWithLogFormat.PersistentFlags().String(constants.LogFormatFlag, "custom", "")
	logFormat, err = GetLogFormat(cmdWithLogFormat)
	assert.NoError(t, err)
	assert.NotNil(t, logFormat)
	assert.Equal(t, LogFormat("custom"), logFormat)
}

// TestGetVersion tests the functionality that returns the right Verrazzano version.
func TestGetVersion(t *testing.T) {
	// Create a fake VZ helper
	rc := testhelpers.NewFakeRootCmdContextWithFiles(t)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)

	// GIVEN a command with no values provided for version flags,
	// WHEN we get the version value,
	// THEN an error is returned.
	version, err := GetVersion(getCommandWithoutFlags(), rc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf(flagNotDefinedErrFmt, constants.VersionFlag))
	assert.NotNil(t, version)
	assert.Equal(t, "", version)

	// GIVEN a command with a default values provided for version flags,
	// WHEN we get the version value,
	// THEN no error is returned.
	cmdWithDefaultVZVersion := getCommandWithoutFlags()
	cmdWithDefaultVZVersion.PersistentFlags().String(constants.VersionFlag, constants.VersionFlagDefault, "")
	version, err = GetVersion(cmdWithDefaultVZVersion, rc)
	assert.NoError(t, err)
	assert.NotNil(t, version)

	// GIVEN a command with a valid version provided for version flags,
	// WHEN we get the version value,
	// THEN no error is returned.
	cmdWithDefaultVZVersion = getCommandWithoutFlags()
	cmdWithDefaultVZVersion.PersistentFlags().String(constants.VersionFlag, "v9.9.9", "")
	version, err = GetVersion(cmdWithDefaultVZVersion, rc)
	assert.NoError(t, err)
	assert.Equal(t, "v9.9.9", version)

	// GIVEN a command with a valid version provided for version flags,
	// WHEN we get the version value,
	// THEN no error is returned.
	cmdWithDefaultVZVersion = getCommandWithoutFlags()
	cmdWithDefaultVZVersion.PersistentFlags().String(constants.VersionFlag, "V9.9.9", "")
	version, err = GetVersion(cmdWithDefaultVZVersion, rc)
	assert.NoError(t, err)
	assert.Equal(t, "v9.9.9", version)

	// GIVEN a command with a valid version provided for version flags,
	// WHEN we get the version value,
	// THEN no error is returned.
	cmdWithDefaultVZVersion = getCommandWithoutFlags()
	cmdWithDefaultVZVersion.PersistentFlags().String(constants.VersionFlag, "9.9.9", "")
	version, err = GetVersion(cmdWithDefaultVZVersion, rc)
	assert.NoError(t, err)
	assert.Equal(t, "v9.9.9", version)

	// GIVEN a command with an invalid version provided for version flags,
	// WHEN we get the version value,
	// THEN an error is returned.
	cmdWithDefaultVZVersion = getCommandWithoutFlags()
	cmdWithDefaultVZVersion.PersistentFlags().String(constants.VersionFlag, "invalid", "")
	_, err = GetVersion(cmdWithDefaultVZVersion, rc)
	assert.EqualError(t, err, "Invalid version string: invalid (valid format is vn.n.n or n.n.n)")
}

// TestGetVersionWithManifests tests the GetVersion function when the user supplies the manifests flag.
func TestGetVersionWithManifests(t *testing.T) {
	// Create a fake VZ helper
	rc := testhelpers.NewFakeRootCmdContextWithFiles(t)
	defer testhelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)

	// GIVEN a command with a specific version and a manifests file with a version that does not match,
	// WHEN we get the version value,
	// THEN an error is returned.
	cmd := getCommandWithoutFlags()
	cmd.PersistentFlags().String(constants.VersionFlag, "", "")
	cmd.PersistentFlags().Set(constants.VersionFlag, "9.9.9")
	cmd.PersistentFlags().String(constants.ManifestsFlag, "", "")
	cmd.PersistentFlags().Set(constants.ManifestsFlag, "../../test/testdata/operator-file-fake.yaml")
	_, err := GetVersion(cmd, rc)
	assert.ErrorContains(t, err, "Requested version '9.9.9' does not match manifests version 'v1.5.2'")

	// GIVEN a command with a specific version and a manifests file with a version that matches,
	// WHEN we get the version value,
	// THEN no error is returned and the returned version is the version specified in the version flag (prefixed with a "v").
	cmd = getCommandWithoutFlags()
	cmd.PersistentFlags().String(constants.VersionFlag, "", "")
	cmd.PersistentFlags().Set(constants.VersionFlag, "1.5.2")
	cmd.PersistentFlags().String(constants.ManifestsFlag, "", "")
	cmd.PersistentFlags().Set(constants.ManifestsFlag, "../../test/testdata/operator-file-fake.yaml")
	version, err := GetVersion(cmd, rc)
	assert.NoError(t, err)
	assert.Equal(t, "v1.5.2", version)

	// GIVEN a command with a specific version that includes a suffix and a manifests file with a major/minor/patch version that matches,
	// WHEN we get the version value,
	// THEN no error is returned and the returned version is the version specified in the version flag (prefixed with a "v").
	cmd = getCommandWithoutFlags()
	cmd.PersistentFlags().String(constants.VersionFlag, "", "")
	cmd.PersistentFlags().Set(constants.VersionFlag, "1.5.2-1234+xyz")
	cmd.PersistentFlags().String(constants.ManifestsFlag, "", "")
	cmd.PersistentFlags().Set(constants.ManifestsFlag, "../../test/testdata/operator-file-fake.yaml")
	version, err = GetVersion(cmd, rc)
	assert.NoError(t, err)
	assert.Equal(t, "v1.5.2-1234+xyz", version)

	// GIVEN the manifests flag has a value and the version flag is not provided,
	// WHEN we get the version value,
	// THEN no error is returned and the returned version is the version from the VPO deployment in the manifests.
	cmd = getCommandWithoutFlags()
	cmd.PersistentFlags().String(constants.VersionFlag, constants.VersionFlagDefault, "")
	cmd.PersistentFlags().String(constants.ManifestsFlag, "", "")
	cmd.PersistentFlags().Set(constants.ManifestsFlag, "../../test/testdata/operator-file-fake.yaml")
	version, err = GetVersion(cmd, rc)
	assert.NoError(t, err)
	assert.Equal(t, "v1.5.2", version)
}

// TestGetOperatorFile tests the functionality to return the right operator file.
func TestGetOperatorFile(t *testing.T) {
	// GIVEN a command with no value provided for the operator file flag,
	// WHEN we get the operator file,
	// THEN the default value of operator file is returned.
	operatorFile, err := getOperatorFileFromFlag(getCommandWithoutFlags())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), fmt.Sprintf(flagNotDefinedErrFmt, constants.ManifestsFlag))
	assert.NotNil(t, operatorFile)
	assert.Equal(t, "", operatorFile)

	// GIVEN a command with no value provided for the operator file flag,
	// WHEN we get the operator file,
	// THEN the default value of operator file is returned.
	cmdWithOperatorFile := getCommandWithoutFlags()
	cmdWithOperatorFile.PersistentFlags().String(constants.ManifestsFlag, "/tmp/operator.yaml", "")
	operatorFile, err = getOperatorFileFromFlag(cmdWithOperatorFile)
	assert.NoError(t, err)
	assert.NotNil(t, operatorFile)
	assert.Equal(t, "/tmp/operator.yaml", operatorFile)

}

// getCommandWithoutFlags creates a dummy test command with no flags
func getCommandWithoutFlags() *cobra.Command {
	return &cobra.Command{
		Use:   testUse,
		Short: testShort,
		Long:  testLong,
	}
}
