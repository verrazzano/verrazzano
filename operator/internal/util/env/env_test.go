// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package env

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// TestVzRootDir tests the env variable VZ_ROOT_DIR
// GIVEN a env variable VZ_ROOT_DIR
//  WHEN I call VzRootDir
//  THEN the value returned is either the contents of VZ_ROOT_DIR or default
func TestVzRootDir(t *testing.T) {
	defer func() { getEnvFunc = os.Getenv }()
	assert := assert.New(t)
	getEnvFunc = os.Getenv
	assert.Equal("/verrazzano", VzRootDir(), "The VZ_ROOT_DIR is incorrect")

	// override env.go function to get env var
	getEnvFunc = func(string) string { return "testdir" }
	assert.Equal("testdir", VzRootDir(), "The VZ_ROOT_DIR is incorrect")
}

// TestVzChartDir tests getting the chart directory
// GIVEN a env variable VZ_ROOT_DIR
//  WHEN I call VzChartDir
//  THEN the value returned is either based on VZ_ROOT_DIR or default
func TestVzChartDir(t *testing.T) {
	defer func() { getEnvFunc = os.Getenv }()
	assert := assert.New(t)
	getEnvFunc = os.Getenv
	assert.Equal("/verrazzano/install/chart", VzChartDir(), "The chart directory is incorrect")

	// override env.go function to get env var
	getEnvFunc = func(string) string { return "/testdir" }
	assert.Equal("/testdir/operator/scripts/install/chart", VzChartDir(), "The chart directory is incorrect")
}

func TestIsCheckVersionRequired(t *testing.T) {
	defer func() { getEnvFunc = os.Getenv }()
	getEnvFunc = func(string) string { return "false" }
	assert.False(t, IsCheckVersionRequired())
	getEnvFunc = func(string) string { return "" }
	assert.True(t, IsCheckVersionRequired())
	getEnvFunc = func(string) string { return "true" }
	assert.True(t, IsCheckVersionRequired())
}
