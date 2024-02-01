// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

// Package files handles searching
package files

import (
	"bufio"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"regexp"
	"time"
)

// TimeRange is used when searching requires time bounded matches
// Handling of a TimeRange:
//
//	if not specified, match is included
//	if StartTime is supplied (not zero) matches at/after that time are included
//	if EndTime is supplied (not zero) matches at/before that time are included
type TimeRange struct {
	StartTime metav1.Time
	EndTime   metav1.Time
}

// TextMatch supplies information about the matched text
type TextMatch struct {
	FileName    string
	FileLine    int
	Timestamp   metav1.Time
	MatchedText string
}

var ZeroTime = metav1.NewTime(time.Time{})

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
func SearchFiles(log *zap.SugaredLogger, rootDirectory string, files []string, searchMatchRe *regexp.Regexp, timeRange *TimeRange) (matches []TextMatch, err error) {
	if searchMatchRe == nil {
		return nil, fmt.Errorf("SaerchFilesRe requires a regular expression")
	}

	if len(files) == 0 {
		log.Debugf("SearchFilesRe was not given any files, return nil")
		return nil, nil
	}

	for _, fileName := range files {
		matchesFromFile, err := SearchFile(log, fileName, searchMatchRe, timeRange)
		if err != nil {
			log.Debugf("failure opening %s", fileName, err)
			return nil, err
		}
		if len(matchesFromFile) > 0 {
			matches = append(matches, matchesFromFile...)
		}
	}
	return matches, nil
}

// SearchFile search a file
func SearchFile(log *zap.SugaredLogger, fileName string, searchMatchRe *regexp.Regexp, timeRange *TimeRange) (matches []TextMatch, err error) {
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

	// Had issues with token too large using the scanner, so using a reader instead
	reader := bufio.NewReader(file)
	lineNumber := 0
	var match TextMatch
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			// If we had an unexpected failure we fail
			log.Debugf("failure reading file %s", fileName, readErr)
			return nil, readErr
		}
		if len(line) > 0 {
			// See if we have a match
			lineNumber++
			matched := searchMatchRe.Find([]byte(line))
			if len(matched) > 0 {
				timestamp := ExtractTimeIfPresent(line)
				if IsInTimeRange(timestamp, timeRange) {
					match.Timestamp = timestamp
					match.FileLine = lineNumber
					match.FileName = fileName
					match.MatchedText = line
					matches = append(matches, match)
				}
			}
		}
		// If we hit EOF, we're done
		if readErr == io.EOF {
			break
		}
	}

	return matches, nil
}

// FindFilesAndSearch will search across files that match a specified expression
func FindFilesAndSearch(log *zap.SugaredLogger, rootDirectory string, fileMatchRe *regexp.Regexp, searchMatchRe *regexp.Regexp, timeRange *TimeRange) (matches []TextMatch, err error) {
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
	return SearchFiles(log, rootDirectory, filesToSearch, searchMatchRe, timeRange)
}

// ExtractTimeIfPresent determines if the text matches a known pattern which has a timestamp in it (such as known log formats)
// and will extract the timestamp into a wrappered metav1.Time. If there is no timestamp found it will return a zero time value
func ExtractTimeIfPresent(inputText string) metav1.Time {
	// TODO: Add known log timestamp patterns, and parse out the times
	return ZeroTime
}

// IsInTimeRange will check if a specified time is within the specified range
// It will return true if:
//   - there is no time range specified
//   - a time range is specified and the time specified is in that range (see TimeRange type)
//
// Otherwise it will return false:
//   - if the time is zero and there is a range specified, it can't determine it
//   - if the time is not within the range specified
func IsInTimeRange(timeToCheck metav1.Time, timeRange *TimeRange) bool {
	// If there is no time range, then all times would match it
	if timeRange == nil || (timeRange.StartTime.IsZero() && timeRange.EndTime.IsZero()) {
		return true
	}

	// We know there is some time range specified, so if there is no input time to check
	// we can't determine if it is in the range, so return false
	if timeToCheck.IsZero() {
		return false
	}

	// if the start/end times in a range are zero, they don't limit the range
	isAfterStart := timeRange.StartTime.IsZero() || !timeToCheck.Before(&timeRange.StartTime)
	isBeforeEnd := timeRange.EndTime.IsZero() || timeToCheck.Before(&timeRange.EndTime) || timeToCheck.Equal(&timeRange.EndTime)
	return isAfterStart && isBeforeEnd
}
