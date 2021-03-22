// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package files handles searching
package files

import (
	"bufio"
	"errors"
	"go.uber.org/zap"
	"os"
	"regexp"
)

// TextMatch supplies information about the matched text
type TextMatch struct {
	FileName    string
	FileLine    int
	MatchedText string
}

// SearchMatches will search the list of TextMatch using a search expression and will return all that match
// This is handy for looking at
func SearchMatches(log *zap.SugaredLogger, matchesToSearch[]TextMatch, searchExpression string) (matches []TextMatch, err error) {
	if len(searchExpression) == 0 {
		return nil, errors.New("SearchMatches requires a search expression")
	}

	searchMatchRe, err := regexp.Compile(searchExpression)
	if err != nil {
		log.Debugf("Failed to compile regular expression: %s", searchExpression, err)
		return nil, err
	}

	for _, matchToSearch := range matchesToSearch {
		if searchMatchRe.MatchString(matchToSearch.MatchedText) {
			matches = append(matches, matchToSearch)
		}
	}
	return matches, nil
}

// SearchFiles will search the list of files that are already known for text that matches
func SearchFiles(log *zap.SugaredLogger, rootDirectory string, files []string, searchExpression string) (matches []TextMatch, err error) {
	if len(searchExpression) == 0 {
		return nil, errors.New("SearchFiles requires a search expression")
	}
	if len(files) == 0 {
		log.Debugf("SearchFiles was not given any files, return nil")
		return nil, nil
	}

	searchMatchRe, err := regexp.Compile(searchExpression)
	if err != nil {
		log.Debugf("Failed to compile regular expression: %s", searchExpression, err)
		return nil, err
	}

	for _, fileName := range files {
		file, err := os.Open(fileName)
		if err != nil {
			log.Debugf("failure opening %s", fileName, err)
			return nil, err
		}
		defer file.Close()

		fileStat, err := file.Stat()
		if err != nil {
			log.Debugf("failure getting stat for %s", fileName, err)
			return nil, err
		}
		if fileStat.IsDir() {
			log.Debugf("Skipping directory in search %s", fileName)
			continue
		}
		if !fileStat.Mode().IsRegular() {
			log.Debugf("Skipping non-regular file in search %s", fileName)
			continue
		}

		scanner := bufio.NewScanner(file)
		lineNumber := 0
		var match TextMatch
		for scanner.Scan() {
			lineNumber++
			matched := searchMatchRe.Find(scanner.Bytes())
			if len(matched) > 0 {
				match.FileLine = lineNumber
				match.FileName = fileName
				match.MatchedText = scanner.Text()
				matches = append(matches, match)
			}
		}
		err = scanner.Err()
		if err != nil {
			log.Debugf("failure scanning file %s", fileName, err)
			return nil, err
		}
	}
	return matches, nil
}

// SearchFile searches a file
func SearchFile(log *zap.SugaredLogger, fileName string, searchExpression string) (matches []TextMatch, err error) {
	if len(searchExpression) == 0 {
		return nil, errors.New("SearchFile requires a search expression")
	}

	if len(fileName) == 0 {
		log.Debugf("SearchFile was not given a files, return nil")
		return nil, nil
	}

	searchMatchRe, err := regexp.Compile(searchExpression)
	if err != nil {
		log.Debugf("Failed to compile regular expression: %s", searchExpression, err)
		return nil, err
	}

	file, err := os.Open(fileName)
	if err != nil {
		log.Debugf("failure opening %s", fileName, err)
		return nil, err
	}
	defer file.Close()

	fileStat, err := file.Stat()
	if err != nil {
		log.Debugf("failure getting stat for %s", fileName, err)
		return nil, err
	}
	if fileStat.IsDir() {
		log.Debugf("Skipping directory in search %s", fileName)
		return nil, nil
	}
	if !fileStat.Mode().IsRegular() {
		log.Debugf("Skipping non-regular file in search %s", fileName)
		return nil, nil
	}

	scanner := bufio.NewScanner(file)
	lineNumber := 0
	var match TextMatch
	for scanner.Scan() {
		lineNumber++
		matched := searchMatchRe.Find(scanner.Bytes())
		if len(matched) > 0 {
			match.FileLine = lineNumber
			match.FileName = fileName
			match.MatchedText = scanner.Text()
			matches = append(matches, match)
		}
	}
	err = scanner.Err()
	if err != nil {
		log.Debugf("failure scanning file %s", fileName, err)
		return nil, err
	}

	return matches, nil
}

// FindFilesAndSearch will search across files that match a specified expression
func FindFilesAndSearch(log *zap.SugaredLogger, rootDirectory string, fileMatch string, searchExpression string) (matches []TextMatch, err error) {
	if len(rootDirectory) == 0 {
		return nil, errors.New("FindFilesAndSearch requires rootDirectory")
	}

	if len(fileMatch) == 0 {
		return nil, errors.New("FindFilesAndSearch requires fileMatch expression")
	}

	if len(searchExpression) == 0 {
		return nil, errors.New("FindFilesAndSearch requires a search expression be supplied")
	}

	// Get the list of files that match
	filesToSearch, err := GetMatchingFiles(log, rootDirectory, fileMatch)
	if err != nil {
		log.Debugf("FindFilesAndSearch failed", err)
		return nil, err
	}

	// Note that SearchFiles will detect if no files were found so just call it
	return SearchFiles(log, rootDirectory, filesToSearch, searchExpression)
}

// GetAllMatches wrappers the Compile and FindAll so we can debug log any compile failures (just reduce the code
// on the calling side a little)
func GetAllMatches(log *zap.SugaredLogger, inputString []byte, matchEx string, number int) (value [][]byte, err error) {
	if len(inputString) == 0 {
		return nil, nil
	}
	matcher, err := regexp.Compile(matchEx)
	if err != nil {
		log.Debugf("Regular expression %s compile failed", matchEx, err)
		return nil, err
	}
	return matcher.FindAll(inputString, number), nil
}
