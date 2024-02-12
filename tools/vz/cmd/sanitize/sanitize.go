// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package sanitize

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/files"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
)

const (
	CommandName = "sanitize"
	helpShort   = "Sanitize information from a directory or tar file"
	helpLong    = "This command sanitizes information from an existing directory or tar file and outputs it into a directory or tar.gz file of your choosing. The results of the sanitization should still be checked by the customer before sending them to support."
)

type flagValidation struct {
	inputDirectory  string
	inputTarFile    string
	outputDirectory string
	outputTarGZFile string
}

// NewCmdSanitize creates a sanitize command with the appropriate arguments and makes the command hidden
func NewCmdSanitize(vzHelper helpers.VZHelper) *cobra.Command {
	cmd := cmdhelpers.NewCommand(vzHelper, CommandName, helpShort, helpLong)
	cmd.Hidden = true

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runCmdSanitize(cmd, args, vzHelper)
	}

	cmd.PersistentFlags().String(constants.InputDirectoryFlagName, constants.InputDirectoryFlagValue, constants.InputDirectoryFlagUsage)
	cmd.PersistentFlags().String(constants.OutputDirectoryFlagName, constants.OutputDirectoryFlagValue, constants.OutputDirectoryFlagUsage)
	cmd.PersistentFlags().String(constants.InputTarFileFlagName, constants.InputTarFileFlagValue, constants.InputTarFileFlagUsage)
	cmd.PersistentFlags().String(constants.OutputTarGZFileFlagName, constants.OutputTarGZFileFlagValue, constants.OutputTarGZFileFlagUsage)
	cmd.PersistentFlags().String(constants.RedactedValuesFlagName, constants.RedactedValuesFlagValue, constants.RedactedValuesFlagUsage)

	// Verifies that the CLI args are not set at the creation of a command
	vzHelper.VerifyCLIArgsNil(cmd)

	return cmd
}

// runCmdSanitize runs a sanitize command which takes an input directory or tar file to sanitize and an output directory or tar.gz file to place the sanitized files
func runCmdSanitize(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	validatedStruct, err := parseInputAndOutputFlags(cmd, vzHelper)
	if err != nil {
		return err
	}
	if validatedStruct.inputDirectory == "" {
		//This is the case where only the tar string is specified, so a temporary directory is made to untar it into
		validatedStruct.inputDirectory, err = os.MkdirTemp("", constants.SanitizeDirInput)
		if err != nil {
			return err
		}
		defer os.RemoveAll(validatedStruct.inputDirectory)
		file, err := os.Open(validatedStruct.inputTarFile)
		defer file.Close()
		if err != nil {
			return fmt.Errorf("an error occurred when trying to open %s: %s", validatedStruct.inputTarFile, err.Error())
		}
		err = helpers.UntarArchive(validatedStruct.inputDirectory, file)
		if err != nil {
			return fmt.Errorf("an error occurred while trying to untar %s: %s", validatedStruct.inputTarFile, err.Error())
		}

	}
	// If an output directory is not specified, create a temporary output directory that can be used to tar the sanitize files
	if validatedStruct.outputDirectory == "" {
		validatedStruct.outputDirectory, err = os.MkdirTemp("", constants.SanitizeDirOutput)
		if err != nil {
			return err
		}
		defer os.RemoveAll(validatedStruct.outputDirectory)
	}
	if err = sanitizeDirectory(*validatedStruct); err != nil {
		return err
	}

	// Process the redacted values file flag.
	redactionFilePath, err := cmd.PersistentFlags().GetString(constants.RedactedValuesFlagName)
	if err != nil {
		return fmt.Errorf(constants.FlagErrorMessage, constants.RedactedValuesFlagName, err.Error())
	}
	if redactionFilePath != "" {
		// Create the redaction map file if the user provides a non-empty file path.
		if err := helpers.WriteRedactionMapFile(redactionFilePath, nil); err != nil {
			return fmt.Errorf(constants.RedactionMapCreationError, redactionFilePath, err.Error())
		}
	}
	return nil
}

// parseInputAndOutputFlags validates the directory and tar file flags along with checking that the directory flag and the tar file are not both specified
func parseInputAndOutputFlags(cmd *cobra.Command, vzHelper helpers.VZHelper) (*flagValidation, error) {
	inputDirectory, err := cmd.PersistentFlags().GetString(constants.InputDirectoryFlagName)
	if err != nil {
		return nil, fmt.Errorf(constants.FlagErrorMessage, constants.InputDirectoryFlagName, err.Error())
	}
	inputTarFileString, err := cmd.PersistentFlags().GetString(constants.InputTarFileFlagName)
	if err != nil {
		return nil, fmt.Errorf(constants.FlagErrorMessage, constants.InputTarFileFlagName, err.Error())
	}
	if inputDirectory != "" && inputTarFileString != "" {
		return nil, fmt.Errorf("an input directory and an input tar file cannot be both specified")
	}
	if inputDirectory == "" && inputTarFileString == "" {
		return nil, fmt.Errorf("an input directory or an input tar file must be specified")
	}
	outputDirectory, err := cmd.PersistentFlags().GetString(constants.OutputDirectoryFlagName)
	if err != nil {
		return nil, fmt.Errorf(constants.FlagErrorMessage, constants.OutputDirectoryFlagName, err.Error())
	}
	outputTarGZFileString, err := cmd.PersistentFlags().GetString(constants.OutputTarGZFileFlagName)
	if err != nil {
		return nil, fmt.Errorf(constants.FlagErrorMessage, constants.OutputTarGZFileFlagName, err.Error())
	}
	if outputDirectory != "" && outputTarGZFileString != "" {
		return nil, fmt.Errorf("an output directory and an output tar.gz file cannot be specified")
	}
	if outputDirectory == "" && outputTarGZFileString == "" {
		return nil, fmt.Errorf("an output directory or an output tar.gz file must be specified")
	}
	return &flagValidation{inputDirectory: inputDirectory, inputTarFile: inputTarFileString, outputDirectory: outputDirectory, outputTarGZFile: outputTarGZFileString}, nil
}

// sanitizeDirectory sanitizes all the files in a directory, outputs the sanitized files to a separate directory, and tars the sanitized directory if necessary
func sanitizeDirectory(validation flagValidation) error {
	listOfFilesToSanitize, err := files.GetAllDirectoriesAndFiles(validation.inputDirectory)
	if !(strings.HasSuffix(validation.inputDirectory, string(os.PathSeparator))) {
		validation.inputDirectory = validation.inputDirectory + string(os.PathSeparator)
	}
	if !(strings.HasSuffix(validation.outputDirectory, string(os.PathSeparator))) {
		validation.outputDirectory = validation.outputDirectory + string(os.PathSeparator)
	}
	if _, err := os.Stat(validation.outputDirectory); errors.Is(err, os.ErrNotExist) {
		os.Mkdir(validation.outputDirectory, 0700)
	}
	if err != nil {
		return err
	}
	for i := range listOfFilesToSanitize {
		fileInfo, err := os.Stat(listOfFilesToSanitize[i])
		if err != nil {
			return err
		}
		fileMode := fileInfo.Mode()
		isDir := fileInfo.IsDir()
		if isMetadataFile(fileInfo.Name(), isDir) {
			continue
		}
		err = sanitizeFileAndWriteItToOutput(validation, isDir, listOfFilesToSanitize[i], fileMode)
		if err != nil {
			return err
		}

	}
	if validation.outputTarGZFile != "" {
		tarGZFileForOutput, err := os.Create(validation.outputTarGZFile)
		defer tarGZFileForOutput.Close()
		if err != nil {
			return err
		}
		if err = helpers.CreateReportArchive(validation.outputDirectory, tarGZFileForOutput, false); err != nil {
			return err
		}
	}
	return nil
}

// sanitizeFileAndWriteItToOutput sanitizes a file and writes the sanitized file to its corresponding path in a separate directory
func sanitizeFileAndWriteItToOutput(validation flagValidation, isDirectory bool, fileToSanitizePath string, fileMode fs.FileMode) error {
	pathOfSanitizedFileOrDirectoryForOutput := strings.ReplaceAll(fileToSanitizePath, validation.inputDirectory, validation.outputDirectory)
	if isDirectory {
		err := os.MkdirAll(pathOfSanitizedFileOrDirectoryForOutput, fileMode)
		return err
	}
	unsanitizedFileBytes, err := os.ReadFile(fileToSanitizePath)
	if err != nil {
		return err
	}
	notSanitizedFileString := string(unsanitizedFileBytes)
	sanitizedFileString := helpers.SanitizeString(notSanitizedFileString, nil)
	sanitizedFileStringAsBytes := []byte(sanitizedFileString)

	err = os.WriteFile(pathOfSanitizedFileOrDirectoryForOutput, sanitizedFileStringAsBytes, fileMode)
	return err
}

// isMetadataFile determines if a file in a tar archive fits the format of Mac Metadata
func isMetadataFile(fileBaseName string, isDir bool) bool {
	return strings.HasPrefix(fileBaseName, "._") && !isDir
}
