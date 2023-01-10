// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestHandleFlags tests the handleFlags function
func TestHandleFlags(t *testing.T) {
	asserts := assert.New(t)

	// GIVEN command line arguments
	// WHEN  the handleFlags function is called
	// THEN  the command line flags are parsed correctly
	const testCertDir = "/tmp/unit-test"
	os.Args = []string{"cmd", "--cert-dir=" + testCertDir}
	handleFlags()

	asserts.Equal(testCertDir, certDir)
}
