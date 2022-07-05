//Copyright (C) 2022, Oracle and/or its affiliates.
//Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var regexList = []string{}
var regexToReplacementMap = make(map[string]string)

const redactIpAddress = "REDACTED-IP4-ADDRESS"
const ipv4Regex = "[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}"

// Initialize the regex string to replacement string map
// Append to this map for any future additions
func initRegexToReplacementMap() {
	regexToReplacementMap[ipv4Regex] = redactIpAddress
}

// GetMatchingFiles returns the filenames for files that match a regular expression.
func GetMatchingFiles(rootDirectory string, fileMatchRe *regexp.Regexp) (fileMatches []string, err error) {
	if len(rootDirectory) == 0 {
		return nil, fmt.Errorf("GetMatchingFiles requires a rootDirectory")
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

// This will iterate over the given root directory to sanitize each file
// replaceFile is to whether replace the file with sanitized content or create a new file with _tmpfoo suffix
func SanitizeDirectory(rootDirectory string, replaceFile bool) {
	initRegexToReplacementMap()
	fileMatches, err := GetMatchingFiles(rootDirectory, regexp.MustCompile(".*"))
	check(err)
	for _, eachFile := range fileMatches {
		SanitizeFile(eachFile, replaceFile)
	}
}

//Sanitize the given file, replaceFile boolean is to whether inplace replacement or to a new file
func SanitizeFile(path string, replaceFile bool) error {
	fmt.Println("path provided is: ", path)
	input, err := os.Open(path)
	check(err)
	defer input.Close()

	outFile, err := os.Create(path + "_tmpfoo")
	check(err)
	defer outFile.Close()

	br := bufio.NewReader(input)
	for {
		l, _, err := br.ReadLine()

		if err == io.EOF || l == nil {
			break
		}
		check(err)

		sanitizedLine := sanitizeEachLine(string(l))
		outFile.WriteString(sanitizedLine + "\n")
	}

	//originalFile, err := os.ReadFile(path)
	//check(err)
	//fmt.Println("Original file is: \n", string(originalFile))

	//modifiedFile, err := os.ReadFile(path + "_tmpfoo")
	//check(err)
	//fmt.Println("Modified file is: \n", string(modifiedFile))

	if replaceFile == true {
		check(os.Remove(path))
		check(os.Rename(path+"_tmpfoo", path))
	}

	return nil
}

// Sanitize each line in a given file,
// Sanitizes based on the regex map initialized above
func sanitizeEachLine(l string) string {
	for k, v := range regexToReplacementMap {
		l = regexp.MustCompile(k).ReplaceAllString(l, v)
	}
	return l
}

func check(e error) error {
	if e != nil {
		fmt.Errorf(e.Error())
		return e
	}
	return nil
}
