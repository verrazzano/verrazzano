package sanitize

// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/files"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"io/fs"
	"os"
	"strings"
)

const (
	CommandName = "sanitize"
	helpShort   = "Sanitize information from an existing cluster snapshot"
	helpLong    = "This command sanitizes information from an existing dire"
)

type flagValidation struct {
	inputDirectory  string
	inputTarFile    string
	outputDirectory string
	outputTarGZFile string
}

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

	// Verifies that the CLI args are not set at the creation of a command
	vzHelper.VerifyCLIArgsNil(cmd)

	return cmd
}
func runCmdSanitize(cmd *cobra.Command, args []string, vzHelper helpers.VZHelper) error {
	validatedStruct, err := parseInputAndOutputFlags(cmd, vzHelper, constants.InputDirectoryFlagName, constants.OutputDirectoryFlagName, constants.InputTarFileFlagName, constants.OutputTarGZFileFlagName)
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
	err = sanitizeDirectory(*validatedStruct)
	return err

}

// This function validates the directory and tar file flags along with checking that the directory flag and the tar file are not both specified
func parseInputAndOutputFlags(cmd *cobra.Command, vzHelper helpers.VZHelper, inputDirectoryFlagValue string, outputDirectoryFlagValue string, inputTarFileFlagValue string, outputTarGZFileFlagValue string) (*flagValidation, error) {
	inputDirectory, err := cmd.PersistentFlags().GetString(inputDirectoryFlagValue)
	if err != nil {
		return nil, fmt.Errorf(constants.FlagErrorMessage, constants.InputDirectoryFlagName, err.Error())
	}
	inputTarFileString, err := cmd.PersistentFlags().GetString(inputTarFileFlagValue)
	if err != nil {
		return nil, fmt.Errorf(constants.FlagErrorMessage, constants.InputTarFileFlagName, err.Error())
	}
	if inputDirectory != "" && inputTarFileString != "" {
		return nil, fmt.Errorf("an input directory and an input tar file cannot be both specified")
	}
	if inputDirectory == "" && inputTarFileString == "" {
		return nil, fmt.Errorf("an input directory or an input tar file must be specified")
	}
	outputDirectory, err := cmd.PersistentFlags().GetString(outputDirectoryFlagValue)
	if err != nil {
		return nil, fmt.Errorf(constants.FlagErrorMessage, constants.OutputDirectoryFlagName, err.Error())
	}
	outputTarGZFileString, err := cmd.PersistentFlags().GetString(outputTarGZFileFlagValue)
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

func sanitizeDirectory(validation flagValidation) error {
	inputDirectory := validation.inputDirectory
	listOfFilesToSanitize, err := files.GetAllDirectoriesAndFiles(inputDirectory)
	if _, err := os.Stat(validation.outputDirectory); errors.Is(err, os.ErrNotExist) {
		os.Mkdir(validation.outputDirectory, 0700)
	}
	if err != nil {
		return err
	}
	for i, _ := range listOfFilesToSanitize {
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

// The function isMetadataFile determines if a file in a tar archive fits the format of Mac Metadata
func isMetadataFile(fileBaseName string, isDir bool) bool {
	return strings.HasPrefix(fileBaseName, "._") && !isDir
}
