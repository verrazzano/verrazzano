// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
	testOPCID            = "\"message\": \"Request a service limit increase from the service limits page in the console. . http status code: 400. Opc request id: a634bbc217b8188f263d98bc0b3d5c05/9AG80960E22B0EDFEFE506BA8D73DF3C/814906C375D7F4651B8A47987CCB4478\", xyz123"
	testOPCIDToRemove    = " a634bbc217b8188f263d98bc0b3d5c05/9AG80960E22B0EDFEFE506BA8D73DF3C/814906C375D7F4651B8A47987CCB4478"
)

func TestSanitizeALine(t *testing.T) {
	strictCheck := func(message string, toRemove string) {
		assert.Contains(t, message, toRemove, "The test case does not contain the expression to remove: "+toRemove)
		i := strings.Index(message, toRemove)
		sanitized := SanitizeString(message)
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
