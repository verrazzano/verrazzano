// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package os

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestVzRootDir tests the env variable VZ_ROOT_DIR
// GIVEN a env variable VZ_ROOT_DIR
//  WHEN I call VzRootDir
//  THEN the value returned is either the contents of VZ_ROOT_DIR or default
func TestVzRootDir(t *testing.T) {
	assert := assert.New(t)

	assert.Equal("/verrazzano", VzRootDir(), "The VZ_ROOT_DIR is incorrect")

	// override env.go function to get env var
	getEnvFunc = func(string) string { return "testdir" }
	assert.Equal("testdir", VzRootDir(), "The VZ_ROOT_DIR is incorrect")
}
