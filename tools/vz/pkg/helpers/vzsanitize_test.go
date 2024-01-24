// Copyright (c) 2022, 2024, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"encoding/csv"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/tools/vz/pkg/constants"
)

var (
	testIP               = "az0/:127.255.255.255l2/}"
	testIPToRemove       = "127.255.255.255"
	testOCID             = "az0/:ocid1.tenancy.oc1..a763cu5f3m7qpzwnvr2so2655cpzgxmglgtui3v7q/}az"
	testOCIDToRemove     = "ocid1.tenancy.oc1..a763cu5f3m7qpzwnvr2so2655cpzgxmglgtui3v7q"
	testSSHToRemove      = "ssh-rsa AAAAB3NzaCDo798PWwYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/ foo@foo-mac"
	testSSH              = "abcd/0: ssh-rsa AAAAB3NzaCDo798PWwYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/ foo@foo-mac"
	testUserData         = "az0:/\"user_data\": \"abcABC012=+\"az0:/"
	testUserDataToRemove = "\"user_data\": \"abcABC012=+\""
	testOPCID            = "\"message\": \"Request a service limit increase from the service limits page in the console. . http status code: 400. Opc request id:  a634bbc217b8188f263d98bc0b3d5c05/9AG80960E22B0EDFEFE506BA8D73DF3C/814906C375D7F4651B8A47987CCB4478\", xyz123"
	testOPCIDToRemove    = "  a634bbc217b8188f263d98bc0b3d5c05/9AG80960E22B0EDFEFE506BA8D73DF3C/814906C375D7F4651B8A47987CCB4478"

	// Specifies the location and name of the CSV file written to by WriteRedactionMapFile for these tests.
	redactMapFileLocation = os.TempDir()
	redactMapFilePath     = redactMapFileLocation + "/" + constants.RedactionMap
)

// TestSanitizeALine tests the SanitizeString function.
// GIVEN a variety of input strings,
// WHEN I call SanitizeString,
// THEN I expect the output strings to be properly sanitized.
func TestSanitizeALine(t *testing.T) {
	testRedactedValues := make(map[string]string)
	strictCheck := func(message string, toRemove string) {
		assert.Contains(t, message, toRemove, "The test case does not contain the expression to remove: "+toRemove)
		i := strings.Index(message, toRemove)
		sanitized := SanitizeString(message, testRedactedValues)
		assert.NotContains(t, sanitized, toRemove, "Failed to remove expression from string: "+toRemove)
		hashLength := (len(sanitized) - len(message)) + len(toRemove)
		reconstructed := sanitized[:i] + toRemove + sanitized[i+hashLength:]
		assert.Equal(t, message, reconstructed, "Extra character(s) removed from the message. Message: "+message+"\n reconstructed message: "+reconstructed)
	}
	strictCheck(testIP, testIPToRemove)
	strictCheck(testOCID, testOCIDToRemove)
	strictCheck(testSSH, testSSHToRemove)
	strictCheck(testUserData, testUserDataToRemove)
	strictCheck(testOPCID, testOPCIDToRemove)
}

// TestWriteRedactionMapFileEmpty tests that a CSV file is successfully created upon calling WriteRedactionMapFile.
// GIVEN zero calls to the redact function,
// WHEN I call WriteRedactionMapFile,
// THEN I expect it to still successfully create a CSV file.
func TestWriteRedactionMapFileEmpty(t *testing.T) {
	a := assert.New(t)
	testRedactedValues := make(map[string]string)
	err := WriteRedactionMapFile(redactMapFileLocation, testRedactedValues)
	a.Nil(err)
	_, err = os.Stat(redactMapFilePath)
	a.Nil(err, "redaction file %s does not exist", redactMapFilePath)
	err = os.Remove(redactMapFilePath)
	a.Nil(err)
}

// TestWriteRedactionMapFile tests that WriteRedactionMapFile correctly writes redacted string mapping to a CSV file.
// GIVEN a few calls to the redact function,
// WHEN I call WriteRedactionMapFile,
// THEN I expect it create a CSV file which contains mappings for all the previously redacted strings.
func TestWriteRedactionMapFile(t *testing.T) {
	a := assert.New(t)
	testRedactedValues := make(map[string]string)
	// redact a variety of inputs, as well as inputting a value more than once.
	testInputs := []string{testIP, testOCID, testSSH, testUserData, testOPCID, testIP}
	for _, input := range testInputs {
		redact(input, testRedactedValues)
	}
	numUniqueInputs := 5
	a.Len(testRedactedValues, numUniqueInputs)

	// write the redacted values to the CSV file
	err := WriteRedactionMapFile(redactMapFileLocation, testRedactedValues)
	a.Nil(err)

	// open the file
	f, err := os.Open(redactMapFilePath)
	a.Nil(err)
	defer f.Close()
	defer os.Remove(f.Name())

	// read the file line by line
	numLines := 0
	reader := csv.NewReader(f)
	record, err := reader.Read()
	for record != nil {
		numLines++
		a.Nil(err)

		// check that this line of the CSV file is as expected
		hash, found := testRedactedValues[record[1]]
		a.True(found)
		a.Equal(record[0], hash)

		record, err = reader.Read()
	}

	// expected number of lines in the csv
	a.Equal(numUniqueInputs, numLines)
}

// TestRedact tests the redact function.
// WHEN I call redact on various input strings,
// THEN the output redacted strings should follow a specific format.
func TestRedact(t *testing.T) {
	a := assert.New(t)
	testRedactedValues := make(map[string]string)

	// test that redacting the same value repeatedly returns the same value
	redactedIP := redact(testIP, testRedactedValues)
	a.Contains(redactedIP, constants.RedactionPrefix)
	a.NotContains(redactedIP, testIP)
	for i := 0; i < 2; i++ {
		r := redact(testIP, testRedactedValues)
		a.Equal(redactedIP, r)
	}

	// test a redacting a different value
	redactedSSH := redact(testSSH, testRedactedValues)
	a.Contains(redactedSSH, constants.RedactionPrefix)
	a.NotContains(redactedSSH, testSSH)
	a.NotEqual(redactedSSH, redactedIP)

	// test redacting the same value yet again returns the same value
	r := redact(testIP, testRedactedValues)
	a.Equal(redactedIP, r)
}
