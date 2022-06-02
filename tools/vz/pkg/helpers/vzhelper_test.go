// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetLatestReleaseVersion
// GIVEN a list of release versions
//  WHEN I call this function
//  THEN expect it to return the latest version string
func TestGetLatestReleaseVersion(t *testing.T) {

	releases := []string{"v0.1.0", "v1.2.1", "v1.3.1"}
	latestRelease, err := getLatestReleaseVersion(releases)
	assert.NoError(t, err)
	assert.Equal(t, latestRelease, "v1.3.1")
}
