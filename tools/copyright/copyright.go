// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// This program will accept a list of files and directories and scan all of the files found therin to make sure that
// they have the correct Oracle copyright header and UPL license headers.
//
// Internally, we manage a list of file extensions and relative file/directory names to ignore.  We also load a list
// of ignore paths from the working directory of the program containing a list of paths relative to that working dir
// to explicitly ignore.

const (
	// ignoreFileDefaultName is the name of the special file that contains a list of files to ignore
	ignoreFileDefaultName = "ignore_copyright_check.txt"

	// maxLines is the maximum number of lines to read in a file before giving up
	maxLines = 5
)

var (
	// filesToSkip is a list of well-known filenames to skip while scanning, relative to the directory being scanned
	filesToSkip = []string{
		".gitlab-ci.yml",
		"go.mod",
		"go.sum",
		"LICENSE",
		"LICENSE.txt",
		"THIRD_PARTY_LICENSES.txt",
		"coverage.html",
		"clair-scanner",
	}

	// directoriesToShip is a list of well-known (sub)directories to skip while scanning, relative to the working
	// directory being scanned
	directoriesToSkip = []string{
		".git",
		"out",
		"bin",
		".settings",
		"thirdparty_licenses",
		"vendor",
		"_output",
		"_gen", "target",
		"node_modules",
	}

	// extensionsToSkip is a list of well-known file extensions that we will skip while scanning, including
	// binary files and file types that do not support comments (like json)
	extensionsToSkip = []string{
		".json",
		".png",
		".csv",
		".ico",
		".md",
		".jpeg",
		".jpg",
		".log",
		"-test-result.xml",
		".woff",
		".woff2",
		".ttf",
		".min.js",
		".min.css",
		".map",
		".cov",
		".iml",
		".jar",
		".zip",
		".gz",
		".test",
	}

	// copyrightRegex is the regular expression for recognizing correctly formatted copyright statements
	// Explanation of the regular expression
	// -------------------------------------
	// ^                           matches start of the line
	// (#|\/\/|<!--|\/\*)          matches either a # character, or two / characters or the literal string "<!--", or "/*"
	// Copyright                   matches the literal string " Copyright "
	// \([cC]\)                    matches "(c)" or "(C)"
	// ([1-2][0-9][0-9][0-9], )    matches a year in the range 1000-2999 followed by a comma and a space
	// ?([1-2][0-9][0-9][0-9], )   matches an OPTIONAL second year in the range 1000-2999 followed by a comma and a space
	// Oracle ... affiliates       matches that literal string
	// (\.|\. -->|\. \*\/|\. --%>) matches "." or ". -->" or ". */"
	// $                           matches the end of the line
	// the correct copyright line looks like this:
	// Copyright (c) 2020, Oracle and/or its affiliates.
	copyrightRegex = regexp.MustCompile(`^(#|\/\/|<!--|\/\*|<%--) Copyright \([cC]\) ([1-2][0-9][0-9][0-9], )?([1-2][0-9][0-9][0-9], )Oracle and\/or its affiliates(\.|\. -->|\. \*\/|\. --%>)$`)

	// uplRegex is the regular express for recognizing correctly formatted UPL license headers
	// Explanation of the regular expression
	// -------------------------------------
	// ^                           matches start of the line
	// (#|\/\/|<!--|\/\*|<%--)     matches either a # character, or two / characters or the literal string "<!--", "/*" or "<%--"
	// Licensed ... licenses\\/upl matches that literal string
	// (\.|\. -->|\. \*\/|\. --%>) matches "." or ". -->" or ". */" or ". --%>"
	// $                           matches the end of the line
	// the correct copyright line looks like this:
	// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
	uplRegex = regexp.MustCompile(`^(#|\/\/|<!--|\/\*|<%--) Licensed under the Universal Permissive License v 1\.0 as shown at https:\/\/oss\.oracle\.com\/licenses\/upl(\.|\. -->|\. \*\/|\. --%>)$`)

	// filesWithErrors Map to track files that failed the check with their error messages
	filesWithErrors map[string][]string

	// numFilesAnalyzed Total number of files analyzed
	numFilesAnalyzed uint

	// numFilesSkipped Total number of files skipped
	numFilesSkipped uint

	// numDirectoriesSkipped Total number of directories skipped
	numDirectoriesSkipped uint

	// filesToIgnore Files to ignore
	filesToIgnore = []string{}

	// directoriesToIgnore Directories to ignore
	directoriesToIgnore = []string{}

	// enforceCurrentYear Enforce that the current year is present in the copyright string (for modified files checks)
	enforceCurrentYear bool

	// currentYear Holds the current year string if we are enforcing that
	currentYear string

	// verbose If true enables verbose output
	verbose = false
)

func main() {

	help := false

	flag.BoolVar(&enforceCurrentYear, "enforce-current", false, "Enforce the current year is present")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.BoolVar(&help, "help", false, "Display usage help")
	flag.Parse()

	if help {
		printUsage()
		os.Exit(0)
	}

	os.Exit(runScan(flag.Args()))
}

// runScan Execute the scan against the provided targets
func runScan(args []string) int {

	if len(args) < 1 {
		fmt.Printf("\nNo pathnames provided for scan, exiting.\n")
		printUsage()
		return 1
	}

	year, _, _ := time.Now().Date()
	currentYear = strconv.Itoa(year) + ", "

	if enforceCurrentYear {
		fmt.Println("Enforcing current year in copyright string")
	}

	if err := loadIgnoreFile(); err != nil {
		fmt.Printf("Error updating ingore files list: %v\n", err)
		return 1
	}

	filesWithErrors = make(map[string][]string, 10)

	// Arguments are a list of directories and/or files.  Iterate through each one and
	// - if it's a file,scan it
	// - if it's a dir, walk it and scan it recursively
	for _, arg := range args {
		fmt.Printf("Scanning target %s\n", arg)
		argInfo, err := os.Stat(arg)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Printf("WARNING: %s does not exist, skipping\n", arg)
				continue
			}
			fmt.Printf("Error getting file info for %s: %v", arg, err.Error())
			return 1
		}
		if argInfo.IsDir() {
			err = filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					if skipOrIgnoreDir(info.Name(), path) {
						if verbose {
							fmt.Printf("Skipping directory %s and all its contents\n", path)
						}
						return filepath.SkipDir
					}
					return nil
				}
				err = checkFile(path, info)
				if err != nil {
					return err
				}
				return nil
			})
		} else {
			err = checkFile(arg, argInfo)
		}
		if err != nil {
			fmt.Printf("Error processing %s: %v", arg, err.Error())
			return 1
		}
	}
	printScanReport()
	if len(filesWithErrors) > 0 {
		return 1
	}
	return 0
}

// checkFile Scans the specified file if it does not match the ignore criteria
func checkFile(path string, info os.FileInfo) error {
	// Ignore the file if
	// - the extension matches one in the global set of ignored extensions
	// - the name matches one in the global set of ignored relative file names
	// - it is in the global ignores list read from disk
	if skipFile(path, info) {
		numFilesSkipped++
		if verbose {
			fmt.Printf("Skipping file %s/n", path)
		}
		return nil
	}

	fileErrors, err := checkCopyrightAndLicense(path)
	if err != nil {
		return err
	}
	numFilesAnalyzed++
	if verbose {
		fmt.Printf("Scanning %s\n", path)
	}
	if len(fileErrors) > 0 {
		filesWithErrors[path] = fileErrors
	}
	return nil
}

// checkCopyrightAndLicense returns true if the file has a valid/correct copyright notice
func checkCopyrightAndLicense(path string) (fileErrors []string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return fileErrors, err
	}
	reader := bufio.NewScanner(file)
	reader.Split(bufio.ScanLines)
	defer file.Close()

	foundCopyright := false
	foundLicense := false

	linesRead := 0
	for reader.Scan() && linesRead < maxLines {
		line := reader.Text()
		if copyrightRegex.MatchString(line) {
			foundCopyright = true
			if enforceCurrentYear && !strings.Contains(line, currentYear) {
				fileErrors = append(fileErrors, "Copyright does not contain current year")
			}
		}
		if uplRegex.MatchString(line) {
			foundLicense = true
		}
		if foundCopyright && foundLicense {
			break
		}
		linesRead++
	}
	if !foundCopyright {
		fileErrors = append(fileErrors, "Copyright not found")
	}
	if !foundLicense {
		fileErrors = append(fileErrors, "License not found")
	}
	return fileErrors, nil
}

// printScanReport Dump the scan to stdout
func printScanReport() {
	fmt.Printf("\nResults of scan:\n\tFiles analyzed: %d\n\tFiles with error: %d\n\tFiles skipped: %d\n\tDirectories skipped: %d\n",
		numFilesAnalyzed, len(filesWithErrors), numFilesSkipped, numDirectoriesSkipped)

	if len(filesWithErrors) > 0 {
		fmt.Printf("\nThe following files have errors:\n")

		// Sort the keys so the files are grouped lexicographically in the output,
		// instead of randomized by just walking the map
		keys := make([]string, 0, len(filesWithErrors))
		for key := range filesWithErrors {
			if len(key) > 0 {
				keys = append(keys, key)
			}
		}
		sort.Strings(keys)

		for _, key := range keys {
			errors := filesWithErrors[key]
			buff := new(bytes.Buffer)
			writer := csv.NewWriter(buff)
			writer.Write(errors)
			writer.Flush()

			fmt.Printf("\tFile: %s, Errors: %s\n", key, buff.String())
		}

		fmt.Println("\nExamples of valid comments:")
		fmt.Println("With forward slash (Java-style):")
		fmt.Println("// Copyright (c) 2021, Oracle and/or its affiliates.")
		fmt.Println("// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.")
		fmt.Println("With dash (For SQL files for example):")
		fmt.Println("-- Copyright (c) 2021, Oracle and/or its affiliates.")
		fmt.Println("-- Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.")
		fmt.Println("XML comments:")
		fmt.Println("<!-- Copyright (c) 2021, Oracle and/or its affiliates. -->")
		fmt.Println("<!-- Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl. -->")
		fmt.Println("With #:")
		fmt.Println("# Copyright (c) 2021, Oracle and/or its affiliates.")
		fmt.Println("# Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.")
	}
}

// loadIgnoreFile Loads the set of user-specified ignore files/paths
func loadIgnoreFile() error {
	ignoreFileName := os.Getenv("COPYRIGHT_INGOREFILE_PATH")
	if len(ignoreFileName) == 0 {
		ignoreFileName = ignoreFileDefaultName
	}

	ignoreFile, err := os.Open(ignoreFileName)
	if err != nil {
		return err
	}
	reader := bufio.NewScanner(ignoreFile)
	reader.Split(bufio.ScanLines)
	defer ignoreFile.Close()

	// ignoreFileList Contents of ignore file
	ignoreFileList := []string{}

	for reader.Scan() {
		line := strings.TrimSpace(reader.Text())
		// skip empty lines - otherwise the code below will end up skipping entire
		if len(line) == 0 {
			continue
		}
		// ignore lines starting with "#"
		if strings.HasPrefix(line, "#") {
			continue
		}
		ignoreFileList = append(ignoreFileList, line)
	}

	for _, ignoreLine := range ignoreFileList {
		info, err := os.Stat(ignoreLine)
		if err != nil {
			continue
		}
		if info.IsDir() {
			// if the path points to an existing directory, add it to directories to ignore
			directoriesToIgnore = append(directoriesToIgnore, ignoreLine)
		} else {
			filesToIgnore = append(filesToIgnore, ignoreLine)
		}
	}

	fmt.Printf("Files to ignore: %v\n", filesToIgnore)
	fmt.Printf("Directories to ignore: %v\n", directoriesToIgnore)
	fmt.Println()
	return nil
}

// skipOrIgnoreDir Returns true if a directory matches the skip or ignore lists
func skipOrIgnoreDir(relativeName string, path string) bool {
	if contains(directoriesToSkip, relativeName) || contains(directoriesToIgnore, path) {
		numDirectoriesSkipped++
		return true
	}
	return false
}

// skipFile Returns true if the file should be ignored/skipped
func skipFile(pathToFile string, info os.FileInfo) bool {
	return contains(filesToSkip, info.Name()) ||
		contains(extensionsToSkip, filepath.Ext(info.Name())) ||
		contains(filesToIgnore, pathToFile) ||
		isFileOnIgnoredPath(pathToFile)
}

// isFileOnIgnoredPath Returns true if the file is under one of the dirs specified in the ignore file
func isFileOnIgnoredPath(filepath string) bool {
	for index := range directoriesToIgnore {
		if strings.Contains(filepath, directoriesToIgnore[index]) {
			return true
		}
	}
	return false
}

// contains Search a list of strings for a value
func contains(strings []string, value string) bool {
	for i := range strings {
		if value == strings[i] {
			return true
		}
	}
	return false
}

// printUsage Prints the help for this program
func printUsage() {
	usageString := `

go run copyright.go [options] path1 [path2 path3 ...]

Options:
	--enforce-current   Enforce that files provided to the tool have the current year in the copyright
	--verbose           Verbose output

`
	fmt.Println(usageString)
}
