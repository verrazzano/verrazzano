// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test to check if the browser opening function is supported on current platform
func TestOpenURLInBrowser(t *testing.T) {
	var (
		tests = []struct {
			args []string
		}{
			{
				[]string{"https://github.com"},
			},
			{
				[]string{"https://google.com"},
			},
		}
	)
	asserts := assert.New(t)
	for _, test := range tests {
		err := OpenURLInBrowser(test.args[0])
		asserts.NoError(err)
	}
}
