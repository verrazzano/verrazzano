// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	testIP       = "127.255.255.255"
	testHostName = "www.oracle.com.io.nip"
)

func TestSanitizeALine(t *testing.T) {
	assert.NotContains(t, SanitizeString(testIP), testIP)
	assert.Contains(t, SanitizeString("test.me.test.me 123"), "123")
	assert.NotContains(t, SanitizeString(testHostName), testHostName)
	assert.Contains(t, SanitizeString("Not a hostname of www.google.com"), "Not a hostname of ")
}
