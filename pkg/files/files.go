// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package files

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// GetMatchingFiles returns the filenames for files that match a regular expression.
func GetMatchingFiles(rootDirectory string, fileMatchRe *regexp.Regexp) (fileMatches []string, err error) {
	if len(rootDirectory) == 0 {
		return nil, errors.New("GetMatchingFiles requires a rootDirectory")
	}

	if fileMatchRe == nil {
		return nil, fmt.Errorf("GetMatchingFiles requires a regular expression")
	}

	walkFunc := func(fileName string, fileInfo os.FileInfo, err error) error {
		if !fileMatchRe.MatchString(fileName) {
			return nil
		}
		if !fileInfo.IsDir() {
			fileMatches = append(fileMatches, fileName)
		}
		return nil
	}

	err = filepath.Walk(rootDirectory, walkFunc)
	if err != nil {
		return nil, err
	}
	return fileMatches, err
}

// GetAllDirectoriesAndFiles returns the directories and filenames for all directories and files within a root directory
func GetAllDirectoriesAndFiles(rootDirectory string) (fileMatches []string, err error) {
	if len(rootDirectory) == 0 {
		return nil, errors.New("GetMatchingFiles requires a rootDirectory")
	}
	walkFunc := func(fileName string, fileInfo os.FileInfo, err error) error {
		if fileName == rootDirectory {
			return nil
		}
		fileMatches = append(fileMatches, fileName)
		return nil
	}

	err = filepath.Walk(rootDirectory, walkFunc)
	return fileMatches, err
}
