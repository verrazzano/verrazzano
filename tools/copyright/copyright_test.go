// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// TestRunScan Tests the main scan driver function
// GIVEN a call to runScan
// WHEN with a directory to scan
// THEN the scan results are as expected
func TestRunScan(t *testing.T) {
	verbose = true
	ec := runScan([]string{"test"})
	assert.Equal(t, 0, ec)
	assert.Equal(t, uint(6), numFilesAnalyzed)
	assert.Equal(t, 5, len(filesWithErrors))
	assert.Equal(t, uint(1), numFilesSkipped)
	assert.Equal(t, uint(1), numDirectoriesSkipped)
}

// TestRunScanNoArgs Tests the main scan driver function
// GIVEN a call to runScan
// WHEN no args are provided
// THEN a non-zero code is returned
func TestRunScanNoArgs(t *testing.T) {
	assert.Equal(t, 1, runScan([]string{}))
}

// TestRunScanNoArgs Tests the main scan driver function
// GIVEN a call to runScan
// WHEN a non-existent path is provided
// THEN a zero error code is returned (the path is ignored)
func TestRunScanPathDoesNotExist(t *testing.T) {
	assert.Equal(t, 0, runScan([]string{"foo"}))
}

// TestContains test the contains() utility fn
// GIVEN a list of strings
// WHEN contains is called with valid and invalid strings
// THEN true is returned when the item is present, false otherwise
func TestContains(t *testing.T) {
	assert.True(t, contains([]string{"foo", "bar", "thud"}, "foo"))
	assert.True(t, contains([]string{"foo", "bar", "thud"}, "bar"))
	assert.True(t, contains([]string{"foo", "bar", "thud"}, "thud"))
	assert.False(t, contains([]string{"foo", "bar", "thud"}, "thwack"))
	assert.False(t, contains([]string{}, "foo"))
	assert.False(t, contains([]string{"foo", "bar", "thud"}, ""))
}

// TestPrintScanReport Keep code coverage happy if we need to
func TestPrintScanReport(t *testing.T) {
	filesWithErrors = make(map[string][]string, 10)
	filesWithErrors["test1"] = []string{"Some kind of error"}
	printScanReport()
	printUsage()
}

// TestSkipOrIgnoreDir tests the skip/ignore dir logic
// GIVEN a call to skipOrIgnoreDir
// WHEN paths are passed that do and do not match the skip criteria
// THEN true is returned if the dir is to be skipped, false otherwise
func TestSkipOrIgnoreDir(t *testing.T) {

	saveIgnores := directoriesToIgnore
	defer func() {
		directoriesToIgnore = saveIgnores
	}()

	directoriesToIgnore = append(directoriesToIgnore, "someDirToIgnore")
	skippedCountBefore := numDirectoriesSkipped
	assert.True(t, skipOrIgnoreDir(directoriesToSkip[0], "foo"))
	assert.True(t, skipOrIgnoreDir("foo", directoriesToIgnore[0]))
	assert.False(t, skipOrIgnoreDir("foo", "bar"))
	assert.Equal(t, skippedCountBefore+2, numDirectoriesSkipped)
}

// TestSkipOrIgnoreDir tests the skip/ignore dir logic
// GIVEN a call to skipOrIgnoreDir
// WHEN paths are passed that do and do not match the skip criteria
// THEN true is returned if the dir is to be skipped, false otherwise
func TestLoadIgnoreFile(t *testing.T) {

	defer os.Unsetenv("COPYRIGHT_INGOREFILE_PATH")

	os.Setenv("COPYRIGHT_INGOREFILE_PATH", "./thud.txt")
	err := loadIgnoreFile()
	assert.NotNil(t, err)

	os.Setenv("COPYRIGHT_INGOREFILE_PATH", "./"+ignoreFileDefaultName)
	err = loadIgnoreFile()
	assert.Nil(t, err)
	assert.True(t, len(directoriesToIgnore) > 0)
	assert.True(t, len(filesToIgnore) > 0)
}

// TestCheckFileNoLicense tests checkFile with a file with no license string
// GIVEN a call to checkFile
// WHEN with a file with valid license and copyright info
// THEN No errors are reported
func TestCheckFile(t *testing.T) {
	runCheckFileTest(t, "test/include/good.txt", false)
}

// TestCheckFileNoLicense tests checkFile with a file with no license string
// GIVEN a call to checkFile
// WHEN with a file with no license string
// THEN Errors are reported
func TestCheckFileNoLicense(t *testing.T) {
	runCheckFileTest(t, "test/include/nolicense.txt", true)
}

// TestCheckFileNoCopyright tests checkFile with a file with no license string
// GIVEN a call to checkFile
// WHEN with a file with no copyright string
// THEN Errors are reported
func TestCheckFileNoCopyright(t *testing.T) {
	runCheckFileTest(t, "test/include/nocopyright.txt", true)
}

// TestCheckFileBadCopyright tests checkFile with a file
// GIVEN a call to checkFile
// WHEN with a file with a bad copyright string
// THEN Errors are reported
func TestCheckFileBadCopyright(t *testing.T) {
	runCheckFileTest(t, "test/include/badcopyright.txt", true)
}

// TestCheckFileBadLicense tests checkFile with a file
// GIVEN a call to checkFile
// WHEN with a file with a bad license string
// THEN Errors are reported
func TestCheckFileBadLicense(t *testing.T) {
	runCheckFileTest(t, "test/include/badlicense.txt", true)
}

// TestCheckFileBuriedInfo tests checkFile with a file
// GIVEN a call to checkFile
// WHEN with a file with valid license/copyright info buried too far down (over 5 lines)
// THEN Errors are reported
func TestCheckFileBuriedInfo(t *testing.T) {
	runCheckFileTest(t, "test/include/buried.txt", true)
}

// TestCheckFileFileOnIgnoreList tests checkFile with a file
// GIVEN a call to checkFile
// WHEN with a file on the ignore list
// THEN No errors are reported and the skip count increments
func TestCheckFileFileOnIgnoreList(t *testing.T) {
	loadIgnoreFile()
	defer func() {
		filesToIgnore = []string{}
		directoriesToIgnore = []string{}
	}()
	beforeTestSkippedCount := numFilesSkipped
	runCheckFileTest(t, "test/include/ignore.txt", false)
	assert.Equal(t, beforeTestSkippedCount+1, numFilesSkipped)
}

func runCheckFileTest(t *testing.T, fileName string, expectErrors bool) {
	filesWithErrors = make(map[string][]string, 10)
	verbose = true

	numErrors := 0
	if expectErrors {
		numErrors = 1
	}

	info, err := os.Stat(fileName)
	assert.Nil(t, err)
	assert.Nil(t, checkFile(fileName, info))
	assert.True(t, len(filesWithErrors) == numErrors)
	_, ok := filesWithErrors[fileName]
	assert.Equal(t, expectErrors, ok)
	printScanReport()
}
