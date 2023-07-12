// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testIP   = "127.255.255.255"
	testOCID = "ocid1.tenancy.oc1..a763cu5f3m7qpzwnvr2so2655cpzgxmglgtui3v7q"
	testSSH  = "ssh-rsa AAAAB3NzaCDo798PWwYniRpZ/DEKAapLQDfrHeR/OO59T4ZUr4ln/5EoUGYu1HRVWmvQx4wsKZRwl4u8pi9gYOW1pL/IYp3cumJef9Y99+/ foo@foo-mac"
)

func TestSanitizeALine(t *testing.T) {
	assert.NotContains(t, SanitizeString(testIP), testIP)
	assert.NotContains(t, SanitizeString(testOCID), testOCID)
	assert.NotContains(t, SanitizeString(testSSH), testSSH)
	assert.Contains(t, SanitizeString("test.me.test.me"), "test")
}
