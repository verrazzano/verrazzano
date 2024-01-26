package sanitize

// Copyright (c) 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/verrazzano/verrazzano/pkg/files"
	cmdhelpers "github.com/verrazzano/verrazzano/tools/vz/cmd/helpers"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/helpers"
	"os"
	"regexp"
)

const (
	flagErrorStr = "error fetching flag: %s"
	CommandName  = "sanitize"
	helpShort    = "Sanitize information from an existing cluster snapshot"
	helpLong     = "sanitize function"
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
	validatedStruct, err := parseInputAndOutputFlags(cmd, vzHelper, constants.InputDirectoryFlagName, constants.InputTarFileFlagName, constants.OutputDirectoryFlagName, constants.OutputTarGZFileFlagName)
	if err != nil {
		return err
	}
	if validatedStruct.inputDirectory != "" {
		validatedStruct.inputDirectory, err = os.MkdirTemp("", constants.SanitizeDir)
		if err != nil {
			return err
		}
		defer os.RemoveAll(validatedStruct.inputDirectory)
		if validatedStruct.inputTarFile != "" {
			//This is the case where only the tar string is specified
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

	}
	return nil

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
	// This regular expression will match all filenames and it assumes that file names won't have new line characters
	regExpForAllFiles, err := regexp.Compile(".*")
	if err != nil {
		return err
	}
	listOfFilesToSanitize, err := files.GetMatchingFiles(inputDirectory, regExpForAllFiles)
	if err != nil {
		return err
	}
	for i, _ := range listOfFilesToSanitize {
		fileToSanitize := listOfFilesToSanitize[i]
		unsanitizedFileBytes, err := os.ReadFile(fileToSanitize)
		if err != nil {
			return err
		}
		notSanitizedFileString := string(unsanitizedFileBytes)
		// Pick up tomorrow to work on output directory
		sanitizedFileString := helpers.SanitizeString(notSanitizedFileString, nil)
		fmt.Println(sanitizedFileString)

	}
	return nil
}
