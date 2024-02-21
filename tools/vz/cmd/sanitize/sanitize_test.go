// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package sanitize

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	testHelpers "github.com/verrazzano/verrazzano/tools/vz/test/helpers"
)

const (
	testCattleSystemPodsDirectory = "../../pkg/internal/test/cluster/testCattleSystempods"
	ipAddressRedactionDirectory   = "../../pkg/internal/test/sanitization/ip-address-redaction"
	ipToSanitize                  = "127.0.0.0"
)

// TestNewCmdSanitize
// GIVEN a VZ Helper
// WHEN I call NewCmdSanitize
// THEN expect the command to successfully create a new Sanitize command
func TestNewCmdSanitize(t *testing.T) {
	rc, err := testHelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testHelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	cmd := NewCmdSanitize(rc)
	assert.NotNil(t, cmd)
}

// TestNoArgsIntoSanitize
// GIVEN an Sanitize command
// WHEN I call cmd.Execute() with no flags passed in
// THEN expect the command to return an error
func TestNoArgsIntoSanitize(t *testing.T) {
	rc, err := testHelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testHelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	cmd := NewCmdSanitize(rc)
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	assert.NotNil(t, err)
}

// TestTwoInputArgsIntoSanitize
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an input directory and an input tar file specified
// THEN expect the command to return an error
func TestTwoInputArgsIntoSanitize(t *testing.T) {
	rc, err := testHelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testHelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputDirectoryFlagName, "input-directory")
	cmd.PersistentFlags().Set(constants.InputTarFileFlagName, "input.tar")
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	assert.NotNil(t, err)
}

// TestTwoOutputArgsIntoSanitize
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an output directory and an output tar.gz file specified
// THEN expect the command to return an error
func TestTwoOutputArgsIntoSanitize(t *testing.T) {
	rc, err := testHelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testHelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.OutputDirectoryFlagName, constants.TestDirectory)
	cmd.PersistentFlags().Set(constants.OutputTarGZFileFlagName, "output.tar.gz")
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	assert.NotNil(t, err)
}

// TestSanitizeCommandWithInputDirectoryAndOutputTarGZFile
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an input directory and an output tar.gz file specified
// THEN expect the command to not return an error and to create the specified tar.gz file
func TestSanitizeCommandWithInputDirectoryAndOutputTarGZFile(t *testing.T) {
	rc, err := testHelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testHelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputDirectoryFlagName, testCattleSystemPodsDirectory)
	cmd.PersistentFlags().Set(constants.OutputTarGZFileFlagName, constants.OutputTarGZFile)
	defer os.Remove(constants.OutputTarGZFile)
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	assert.Nil(t, err)
}

// TestInputDirectoryAndOutputDirectorySlashCombinations
// GIVEN a Sanitize command
// When I call cmd.Execute() with all possible combinations of the input directory and output directory containing or not containing a slash at the end of its path
// Then I expect the commands to not return an error and create the specified directory in the correct format
func TestInputDirectoryAndOutputDirectorySlashCombinations(t *testing.T) {
	tests := []struct {
		inputDirectory  string
		outputDirectory string
	}{
		{inputDirectory: testCattleSystemPodsDirectory + "/", outputDirectory: constants.TestDirectory + "/"},
		{inputDirectory: testCattleSystemPodsDirectory + "/", outputDirectory: constants.TestDirectory},
		{inputDirectory: testCattleSystemPodsDirectory, outputDirectory: constants.TestDirectory + "/"},
		{inputDirectory: testCattleSystemPodsDirectory, outputDirectory: constants.TestDirectory},
	}
	for i, tt := range tests {
		t.Run("Test "+fmt.Sprint(i+1), func(t *testing.T) {
			rc, err := testHelpers.NewFakeRootCmdContextWithFiles()
			assert.Nil(t, err)
			defer testHelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
			cmd := NewCmdSanitize(rc)
			cmd.PersistentFlags().Set(constants.InputDirectoryFlagName, tt.inputDirectory)
			cmd.PersistentFlags().Set(constants.OutputDirectoryFlagName, tt.outputDirectory)
			defer os.RemoveAll(constants.TestDirectory)
			assert.NotNil(t, cmd)
			err = cmd.Execute()
			assert.Nil(t, err)
			_, err = os.Stat(constants.TestDirectory + "/cluster-snapshot")
			assert.Nil(t, err)
		})
	}
}

// TestSanitizeCommandWithInputTarAndOutputTarGZFile
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an input tar file and an output .tar.gz file specified
// THEN expect the command to not return an error and to create the specified directory
func TestSanitizeCommandWithInputTarAndOutputTarGZFile(t *testing.T) {
	rc, err := testHelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testHelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputTarFileFlagName, "../../pkg/internal/test/cluster/istio-ingress-ip-not-found-test.tar")
	cmd.PersistentFlags().Set(constants.OutputTarGZFileFlagName, constants.OutputTarGZFile)
	defer os.Remove(constants.OutputTarGZFile)
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	assert.Nil(t, err)
}

// TestSanitizeCommandWithInputTarAndOutputDirectoryFile
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with both an input tar file and an output directory file specified
// THEN expect the command to not return an error and to create the specified directory
func TestSanitizeCommandWithInputTarAndOutputDirectoryFile(t *testing.T) {
	rc, err := testHelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testHelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputTarFileFlagName, "../../pkg/internal/test/cluster/istio-ingress-ip-not-found-test.tar")
	cmd.PersistentFlags().Set(constants.OutputDirectoryFlagName, constants.TestDirectory)
	defer os.RemoveAll(constants.TestDirectory)
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	assert.Nil(t, err)
}

// TestSanitizeCommandCorrectlyObscuresInput
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with a directory that contains files that meet the criteria to be sanitized
// THEN expect the command to not return an error and to output a directory with those files correctly sanitized
func TestSanitizeCommandCorrectlyObscuresInput(t *testing.T) {
	rc, err := testHelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testHelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputDirectoryFlagName, ipAddressRedactionDirectory)
	cmd.PersistentFlags().Set(constants.OutputDirectoryFlagName, constants.TestDirectory)
	defer os.RemoveAll(constants.TestDirectory)
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	assert.Nil(t, err)
	sanitizedFileBytes, err := os.ReadFile(constants.TestDirectory + string(os.PathSeparator) + "ip-address-not-sanitized.txt")
	assert.Nil(t, err)
	sanitizedFileString := string(sanitizedFileBytes)
	assert.Contains(t, sanitizedFileString, helpers.SanitizeString(ipToSanitize, nil))
}

// TestSanitizeRedactedValuesMap
// GIVEN a Sanitize command
// WHEN I call cmd.Execute() with the --redacted-values-file flag set
// THEN expect the command to not return an error and create a redacted values file
func TestSanitizeRedactedValuesFile(t *testing.T) {
	redactedValuesTestFile := filepath.Join(os.TempDir(), "test-map.csv")
	rc, err := testHelpers.NewFakeRootCmdContextWithFiles()
	assert.Nil(t, err)
	defer testHelpers.CleanUpNewFakeRootCmdContextWithFiles(rc)
	cmd := NewCmdSanitize(rc)
	cmd.PersistentFlags().Set(constants.InputDirectoryFlagName, ipAddressRedactionDirectory)
	cmd.PersistentFlags().Set(constants.OutputDirectoryFlagName, constants.TestDirectory)
	cmd.PersistentFlags().Set(constants.RedactedValuesFlagName, redactedValuesTestFile)
	defer os.RemoveAll(constants.TestDirectory)
	defer os.Remove(redactedValuesTestFile)
	assert.NotNil(t, cmd)
	err = cmd.Execute()
	assert.Nil(t, err)

	// read the redacted values CSV file
	f, err := os.Open(redactedValuesTestFile)
	assert.Nil(t, err)
	reader := csv.NewReader(f)
	mapContents, err := reader.ReadAll()
	assert.Nil(t, err)
	assert.Len(t, mapContents, 1)
	redactedPair := mapContents[0]
	assert.Equal(t, redactedPair[0], helpers.SanitizeString(ipToSanitize, nil))
	assert.Equal(t, redactedPair[1], ipToSanitize)
}

// TestIsMetadataFile
// GIVEN a call to isMetadataFile
// WHEN I call isMetadataFile with an input that meets the criteria for a Mac metadata file in a tar file and inputs that do not
// THEN expect the function to correctly return whether the inputs meet the criteria or do not meet the criteria
func TestIsMetadataFile(t *testing.T) {
	assert.True(t, isMetadataFile("._cluster-snapshot", false))
	assert.False(t, isMetadataFile("cluster-snapshot", true))
	assert.False(t, isMetadataFile("cluster-snapshot", false))
}

// TestSanitizeDirectory
// GIVEN a call to sanitizeDirectory
// WHEN I call sanitizeDirectory with a structure that contains an input directory that contains files that need to be sanitized
// THEN expect the function to correctly create the expected output directory containing the properly sanitized file
func TestSanitizeDirectory(t *testing.T) {
	validationForTest := flagValidation{inputDirectory: "../../pkg/internal/test/sanitization/ocid-redaction", inputTarFile: "", outputTarGZFile: "", outputDirectory: constants.TestDirectory}
	defer os.RemoveAll(constants.TestDirectory)
	err := sanitizeDirectory(validationForTest)
	assert.Nil(t, err)
	sanitizedFileBytes, err := os.ReadFile(constants.TestDirectory + string(os.PathSeparator) + "ocid-redaction-not-sanitized.txt")
	assert.Nil(t, err)
	sanitizedFileString := string(sanitizedFileBytes)
	assert.Contains(t, sanitizedFileString, helpers.SanitizeString("ocid1.tenancy.oc1..a763cu5f3m7qpzwnvr2so2655cpzgxmglgtui3v7q", nil))
}

// TestSanitizeFileAndWriteItToOutput
// GIVEN a call to SanitizeFileAndWriteItToOutput
// WHEN I call sanitizeFileAndWriteItToOutput with the appropriate arguments that references a file that does not need to be altered
// THEN expect the function to create the new file, which should be identical to the old file
func TestSanitizeFileAndWriteItToOutput(t *testing.T) {
	validationForTest := flagValidation{inputDirectory: "../../pkg/internal/test/sanitization/no-redaction", inputTarFile: "", outputTarGZFile: "", outputDirectory: constants.TestDirectory}
	os.Mkdir("test-directory", 0700)
	defer os.RemoveAll(constants.TestDirectory)
	err := sanitizeFileAndWriteItToOutput(validationForTest, false, "../../pkg/internal/test/sanitization/no-redaction/no-redaction-needed.txt", 0600)
	assert.Nil(t, err)
	sanitizedFileBytes, err := os.ReadFile(constants.TestDirectory + string(os.PathSeparator) + "no-redaction-needed.txt")
	assert.Nil(t, err)
	unsanitizedFileBytes, err := os.ReadFile("../../pkg/internal/test/sanitization/no-redaction/no-redaction-needed.txt")
	assert.Nil(t, err)
	assert.Equal(t, sanitizedFileBytes, unsanitizedFileBytes)
}
