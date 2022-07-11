// Copyright (C) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	testIP = "127.255.255.255"
)

func TestSanitizeALine(t *testing.T) {
	assert.NotContains(t, SanitizeString(testIP), testIP)
	assert.Contains(t, SanitizeString("test.me.test.me"), "test")
}
