// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package files handles searching
package files

import (
	"bufio"
	"errors"
	"fmt"
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

// TODO: May move to only functions which require pre-compiled regular expressions, and have the pre-compiled at
// compilation time rather than at runtime

// SearchMatches will search the list of TextMatch using a search expression and will return all that match
func SearchMatches(log *zap.SugaredLogger, matchesToSearch []TextMatch, searchMatchRe *regexp.Regexp) (matches []TextMatch, err error) {
	for _, matchToSearch := range matchesToSearch {
		if searchMatchRe.MatchString(matchToSearch.MatchedText) {
			matches = append(matches, matchToSearch)
		}
	}
	return matches, nil
}

// SearchFiles will search the list of files that are already known for text that matches
func SearchFiles(log *zap.SugaredLogger, rootDirectory string, files []string, searchMatchRe *regexp.Regexp) (matches []TextMatch, err error) {
	if searchMatchRe == nil {
		return nil, fmt.Errorf("SaerchFilesRe requires a regular expression")
	}

	if len(files) == 0 {
		log.Debugf("SearchFilesRe was not given any files, return nil")
		return nil, nil
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

// SearchFile search a file
func SearchFile(log *zap.SugaredLogger, fileName string, searchMatchRe *regexp.Regexp) (matches []TextMatch, err error) {
	if searchMatchRe == nil {
		return nil, fmt.Errorf("SearchFileRe requires a regular expression")
	}

	if len(fileName) == 0 {
		log.Debugf("SearchFileRe was not given a file, return nil")
		return nil, nil
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
func FindFilesAndSearch(log *zap.SugaredLogger, rootDirectory string, fileMatchRe *regexp.Regexp, searchMatchRe *regexp.Regexp) (matches []TextMatch, err error) {
	if len(rootDirectory) == 0 {
		return nil, errors.New("FindFilesAndSearch requires rootDirectory")
	}

	if fileMatchRe == nil {
		return nil, errors.New("FindFilesAndSearch requires fileMatch expression")
	}

	if searchMatchRe == nil {
		return nil, errors.New("FindFilesAndSearch requires a search expression be supplied")
	}

	// Get the list of files that match
	filesToSearch, err := GetMatchingFiles(log, rootDirectory, fileMatchRe)
	if err != nil {
		log.Debugf("FindFilesAndSearch failed", err)
		return nil, err
	}

	// Note that SearchFiles will detect if no files were found so just call it
	return SearchFiles(log, rootDirectory, filesToSearch, searchMatchRe)
}
