// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package main

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// TestGetInstallReleaseWithoutMajorVersion Tests the getInstallRelease function
// WHEN with git release tags major version change does not exist
// THEN install release version with two minor release difference is expected
func TestGetInstallReleaseWithoutMajorVersion(t *testing.T) {
	pwd, _ := os.Getwd()
	parseCliArgs([]string{pwd, "install-version"})
	releaseTags := []string{"v1.0.0", "v1.0.1", "v1.0.2", "v1.0.3", "v1.0.4", "v1.1.0", "v1.1.1", "v1.1.2", "v1.2.0", "v1.2.1", "v1.2.2", "v1.3.0", "v1.3.1", "v1.3.2", "v1.3.3", "v1.3.4", "v1.3.5", "v1.3.6", "v1.3.7", "v1.3.8", "v1.4.0", "v1.4.1", "v1.4.2"}
	assert.Equal(t, "v1.2.2\n", getInstallRelease(releaseTags))
}

// TestGetInstallReleaseWithMajorRelease Tests the getInstallRelease function
// WHEN with git release tags major version change exists
// THEN install release version with two minor release difference is expected
func TestGetInstallReleaseWithMajorRelease(t *testing.T) {
	pwd, _ := os.Getwd()
	parseCliArgs([]string{pwd, "install-version"})
	releaseTags := []string{"v1.0.0", "v1.0.1", "v1.0.2", "v1.0.3", "v1.0.4", "v1.1.0", "v1.1.1", "v1.1.2", "v1.2.0", "v1.2.1", "v1.2.2", "v1.3.0", "v1.3.1", "v1.3.2", "v1.3.3", "v1.3.4", "v1.3.5", "v1.3.6", "v1.3.7", "v1.3.8", "v1.4.0", "v1.4.1", "v1.4.2", "v1.5.0", "v2.0.0", "v2.1.0"}
	assert.Equal(t, "v1.5.0\n", getInstallRelease(releaseTags))
}

// TestGetInterimReleaseWithoutMajorVersion Tests the getInterimRelease function
// WHEN with git release tags major version change does not exist
// THEN interim release version with one minor release difference is expected
func TestGetInterimReleaseWithoutMajorVersion(t *testing.T) {
	pwd, _ := os.Getwd()
	parseCliArgs([]string{pwd, "interim-version"})
	releaseTags := []string{"v1.0.0", "v1.0.1", "v1.0.2", "v1.0.3", "v1.0.4", "v1.1.0", "v1.1.1", "v1.1.2", "v1.2.0", "v1.2.1", "v1.2.2", "v1.3.0", "v1.3.1", "v1.3.2", "v1.3.3", "v1.3.4", "v1.3.5", "v1.3.6", "v1.3.7", "v1.3.8", "v1.4.0", "v1.4.1", "v1.4.2"}
	assert.Equal(t, "v1.3.8\n", getInterimRelease(releaseTags))
}

// TestGetInterimReleaseWithMajorVersion Tests the getInterimRelease function
// WHEN with git release tags major version change exists
// THEN interim release version with one minor release difference is expected
func TestGetInterimReleaseWithMajorVersion(t *testing.T) {
	pwd, _ := os.Getwd()
	parseCliArgs([]string{pwd, "interim-version"})
	releaseTags := []string{"v1.0.0", "v1.0.1", "v1.0.2", "v1.0.3", "v1.0.4", "v1.1.0", "v1.1.1", "v1.1.2", "v1.2.0", "v1.2.1", "v1.2.2", "v1.3.0", "v1.3.1", "v1.3.2", "v1.3.3", "v1.3.4", "v1.3.5", "v1.3.6", "v1.3.7", "v1.3.8", "v1.4.0", "v1.4.1", "v1.4.2", "v1.5.0", "2.0.0"}
	assert.Equal(t, "v1.5.0\n", getInterimRelease(releaseTags))
}
