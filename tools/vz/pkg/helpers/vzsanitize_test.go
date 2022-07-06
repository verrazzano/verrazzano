package helpers

import (
	"github.com/stretchr/testify/assert"
	"os"
	"regexp"
	"testing"
)

var testDir = "../../pkg/analysis/test/files"

const testIP = "132.23.234.24"

func TestSanitizeDirectoryMatch(t *testing.T) {
	SanitizeDirectory(testDir, false)
	testFile := testDir + "/sanity_test.txt"
	testFiletmp := testFile + "_tmpfoo"
	file, err := os.ReadFile(testFiletmp)
	assert.Nil(t, err)
	//assert.Contains(t, string(file), "REDACT")
	assert.NotContains(t, string(file), testIP)
	testCleanup(testDir)
}

func TestSanitizeDirectoryNoMatch(t *testing.T) {
	SanitizeDirectory(testDir, false)
	testFile := testDir + "/sanity_test_no_match.txt"
	testFiletmp := testFile + "_tmpfoo"
	file, err := os.ReadFile(testFiletmp)
	assert.Nil(t, err)
	//assert.NotContains(t, string(file), "REDACT")
	assert.Contains(t, string(file), "2134.46.75689.235464356768")
	testCleanup(testDir)
}

func testCleanup(path string) error {
	files, err := GetMatchingFiles(path, regexp.MustCompile(".*_tmpfoo"))
	if err != nil {
		return err
	}
	for _, eachFile := range files {
		println("deleting temp file: ", eachFile)
		os.Remove(eachFile)
	}
	return nil
}

func TestSanitizeALine(t *testing.T) {
	assert.NotContains(t, SanitizeALine(testIP), testIP)
	assert.Contains(t, SanitizeALine("test.me.test.me"), "test")
	testCleanup(testDir)
}
