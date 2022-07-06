//Copyright (C) 2022, Oracle and/or its affiliates.
//Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
)

var regexToReplacementList = []string{}

const ipv4Regex = "[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}\\.[[:digit:]]{1,3}"

// InitRegexToReplacementMap Initialize the regex string to replacement string map
// Append to this map for any future additions
func InitRegexToReplacementMap() {
	regexToReplacementList = append(regexToReplacementList, ipv4Regex)
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

// SanitizeDirectory will iterate over the given root directory to sanitize each file
// ReplaceFile is to whether replace the file with sanitized content or create a new file with _tmpfoo suffix
func SanitizeDirectory(rootDirectory string, replaceFile bool) {
	InitRegexToReplacementMap()
	fileMatches, err := GetMatchingFiles(rootDirectory, regexp.MustCompile(".*"))
	check(err)
	for _, eachFile := range fileMatches {
		SanitizeFile(eachFile, replaceFile)
	}
}

// SanitizeFile the given file, replaceFile boolean is to whether inplace replacement or to a new file
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

		sanitizedLine := SanitizeALine(string(l))
		outFile.WriteString(sanitizedLine + "\n")
	}

	if replaceFile {
		check(os.Remove(path))
		check(os.Rename(path+"_tmpfoo", path))
	}

	return nil
}

// SanitizeALine sanitizes each line in a given file,
// Sanitizes based on the regex map initialized above
func SanitizeALine(l string) string {
	if len(regexToReplacementList) == 0 {
		InitRegexToReplacementMap()
	}
	for _, eachRegex := range regexToReplacementList {
		l = regexp.MustCompile(eachRegex).ReplaceAllString(l, getMd5Hash(l))
	}
	return l
}

func check(e error) error {
	if e != nil {
		return e
	}
	return nil
}

// getMd5Hash generates the one way hash for the input string
func getMd5Hash(line string) string {
	data := []byte(line)
	hashedVal := md5.Sum(data)
	hexString := hex.EncodeToString(hashedVal[:])
	return hexString
}
