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
