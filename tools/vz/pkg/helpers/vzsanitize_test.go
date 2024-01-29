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
	testIP                     = "az0/:127.255.255.255l2/}"
	testIPToRemove             = "127.255.255.255"
	testOCID                   = "az0/:ocid1.tenancy.oc1..a763cu5f3m7qpzwnvr2so2655cpzgxmglgtui3v7q/}az"
	testOCIDToRemove           = "ocid1.tenancy.oc1..a763cu5f3m7qpzwnvr2so2655cpzgxmglgtui3v7q"
	testSSHToRemove            = "ssh-NewKey-Format9@. AAAAB3NzaCDo798PWwYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/=foo @foo-mac z"
	testSSH_noComment          = "abcd/0: ssh-NewKey-Format9@. AAAAB3NzaCDo798PWwYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/=\n xYz123"
	testSSHToRemove_noComment  = "ssh-NewKey-Format9@. AAAAB3NzaCDo798PWwYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/="
	testSSH                    = "abcd/0: ssh-NewKey-Format9@. AAAAB3NzaCDo798PWwYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/=foo @foo-mac z\n xYz123"
	testSSHToRemove_RSA        = "ssh-rsa AAAAB3NzaCDo798PWwYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/=foo @foo-mac z"
	testSSH_RSA                = "abcd/0: ssh-rsa AAAAB3NzaCDo798PWwYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/=foo @foo-mac z\n xYz123"
	testSSHToRemove_sk25519    = "sk-ssh-ed25519@openssh.com AAAAW2XXFA0S2f2tHUFyEb6ktadcbfO2MczKg7z/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/== z@dxy fyz-ru"
	testSSH_sk25519            = "abcd/0: sk-ssh-ed25519@openssh.com AAAAW2XXFA0S2f2tHUFyEb6ktadcbfO2MczKg7z/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/== z@dxy fyz-ru\n xYz123"
	testSSHToRemove_sk_ecdsa   = "sk-ecdsa-sha2-nistp256@openssh.com AAAAInNrLWVjZHNYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/YniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/=== z@dxy fyz-ru"
	testSSH_sk_ecdsa           = "edcsa-abcd/0: sk-ecdsa-sha2-nistp256@openssh.com AAAAInNrLWVjZHNYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/YniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/=== z@dxy fyz-ru\n xYz123"
	testSSHToRemove_ecdsa_sha2 = "ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBADz1oA4gh3qZExdiS6krVOHXhh3KAMG9SHj1RqMXskDy2sTmO9mPF0P2HJfkm0OgCSMo3BZfvh2rh23fMfUI67gigAmOm41fGQ8B/K82sWj0LuskUR2TqRGQFwwWOVZYtUVtiboTg+XgL5fcGitxL+biT9LMTSOAiRw39cHmk6+B0kXBw== z@dxy fyz-ru"
	testSSH_ecdsa_sha2         = "ecdsa-abcd ecdsa-sha2-nistp521 AAAAE2VjZHNhLXNoYTItbmlzdHA1MjEAAAAIbmlzdHA1MjEAAACFBADz1oA4gh3qZExdiS6krVOHXhh3KAMG9SHj1RqMXskDy2sTmO9mPF0P2HJfkm0OgCSMo3BZfvh2rh23fMfUI67gigAmOm41fGQ8B/K82sWj0LuskUR2TqRGQFwwWOVZYtUVtiboTg+XgL5fcGitxL+biT9LMTSOAiRw39cHmk6+B0kXBw== z@dxy fyz-ru\n xYz123"
	testSSHToRemove_ed25519    = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIL/PsfEX91dggIwWL4edgvgVgn4FJdtZd9ZFXXXXXXXX z@dxy fyz-ru"
	testSSH_ed25519            = "ecdsa-abcd0 ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIL/PsfEX91dggIwWL4edgvgVgn4FJdtZd9ZFXXXXXXXX z@dxy fyz-ru\n xYz123"
	testSSHToRemove_dss        = "ssh-dss AAAAB3NzaC1kc3MAAACBAM6RtXQiwMnnreGmgpT9yinlWLFA8tycOT7or/7iXG06cp7BixJg65Xkl2zKXbq9/Sv+PAFfy7uK0ROSlya1IirMTDFWjMCXaOPwyHb+pM6uBA5UFQxQ9/I+KhWcfelqVVaGK36Xz7N8tCf+IwPvlkK4JeOnbmFfF0a3+nmlPsuXAAAAFQDFlq/WHwSVHlQXzBGRw6Kx7fbj6wAAAIAUZSEIPUFW7bKn8zQ7G7OXpIyMjxnWrpoDb38qTKyhcVrlMgH8cLb558SO/itTkLNRyNPLlSVuxM6qngm1jzPK0NzZbnVtxhTQjCbPmIml3nFjpXpJDUo7nXdR/Gzk15ffQTN/44cqkY/90x87ZgwqNLF8x44B1IUDyyG7NvTNcQAAAIEAy18U+7rh21k5pHBzOY0peZu/x/9/Cu7eJMFpmY7Za+XChjGHmuu2lw9xqebP3SQDIFyMQzXnV39bJXggAHPeGD+Rg2028PcF8w8veBh/+8OgQn+AyFinBRSir7huSApU223R+HvSMZsmXY9I9ycmVULOFy7/WLAcOXXXXX3ig1I= z@dxy fyz-ru"
	testSSH_dss                = "ssh-xyz ssh-dss AAAAB3NzaC1kc3MAAACBAM6RtXQiwMnnreGmgpT9yinlWLFA8tycOT7or/7iXG06cp7BixJg65Xkl2zKXbq9/Sv+PAFfy7uK0ROSlya1IirMTDFWjMCXaOPwyHb+pM6uBA5UFQxQ9/I+KhWcfelqVVaGK36Xz7N8tCf+IwPvlkK4JeOnbmFfF0a3+nmlPsuXAAAAFQDFlq/WHwSVHlQXzBGRw6Kx7fbj6wAAAIAUZSEIPUFW7bKn8zQ7G7OXpIyMjxnWrpoDb38qTKyhcVrlMgH8cLb558SO/itTkLNRyNPLlSVuxM6qngm1jzPK0NzZbnVtxhTQjCbPmIml3nFjpXpJDUo7nXdR/Gzk15ffQTN/44cqkY/90x87ZgwqNLF8x44B1IUDyyG7NvTNcQAAAIEAy18U+7rh21k5pHBzOY0peZu/x/9/Cu7eJMFpmY7Za+XChjGHmuu2lw9xqebP3SQDIFyMQzXnV39bJXggAHPeGD+Rg2028PcF8w8veBh/+8OgQn+AyFinBRSir7huSApU223R+HvSMZsmXY9I9ycmVULOFy7/WLAcOXXXXX3ig1I= z@dxy fyz-ru\n xYz123"
	testUserData               = "az0:/\"user_data\": \"abcABC012=+\"az0:/"
	testUserDataToRemove       = "\"user_data\": \"abcABC012=+\""
	testOPCID                  = "\"message\": \"Request a service limit increase from the service limits page in the console. . http status code: 400. Opc request id:  a634bbc217b8188f263d98bc0b3d5c05/9AG80960E22B0EDFEFE506BA8D73DF3C/814906C375D7F4651B8A47987CCB4478\", xyz123"
	testOPCIDToRemove          = "  a634bbc217b8188f263d98bc0b3d5c05/9AG80960E22B0EDFEFE506BA8D73DF3C/814906C375D7F4651B8A47987CCB4478"

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
	strictCheck(testSSH_noComment, testSSHToRemove_noComment)
	strictCheck(testSSH_RSA, testSSHToRemove_RSA)
	strictCheck(testSSH_sk25519, testSSHToRemove_sk25519)
	strictCheck(testSSH_sk_ecdsa, testSSHToRemove_sk_ecdsa)
	strictCheck(testSSH_ecdsa_sha2, testSSHToRemove_ecdsa_sha2)
	strictCheck(testSSH_ed25519, testSSHToRemove_ed25519)
	strictCheck(testSSH_dss, testSSHToRemove_dss)
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
