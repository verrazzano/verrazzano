// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package semver

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

// TestValidSemver Tests the SemVersion parser for valid version strings
func TestValidSemver(t *testing.T) {
	testVersions := [][]string{
		{"0", "0", "4"},
		{"1", "2", "3"},
		{"10", "20", "30"},
		{"1", "1", "2", "prerelease", "meta"},
		{"1", "1", "2", "", "meta"},
		{"1", "1", "2", "", "meta-valid"},
		{"1", "0", "0", "alpha", ""},
		{"1", "0", "0", "beta", ""},
		{"1", "0", "0", "alpha.beta", ""},
		{"1", "0", "0", "alpha-a.b-c-somethinglong", "build.1-aef.1-its-okay"},
	}
	for _, verComponents := range testVersions {
		verString := fmt.Sprintf("v%s.%s.%s", verComponents[0], verComponents[1], verComponents[2])
		hasPreRelease := len(verComponents) > 3 && len(verComponents[3]) > 0
		if hasPreRelease {
			verString = fmt.Sprintf("%s-%s", verString, verComponents[3])
		}
		hasBuild := len(verComponents) > 4 && len(verComponents[4]) > 0
		if hasBuild {
			verString = fmt.Sprintf("%s+%s", verString, verComponents[4])
		}

		version, err := NewSemVersion(verString)
		assert.Nil(t, err)
		assert.NotNil(t, version)
		assert.Equal(t, verString, version.VersionString)
		expectedMajor, _ := strconv.ParseInt(verComponents[0], 10, 64)
		assert.Equal(t, expectedMajor, version.Major)
		expectedMinor, _ := strconv.ParseInt(verComponents[1], 10, 64)
		assert.Equal(t, expectedMinor, version.Minor)
		expectedPatch, _ := strconv.ParseInt(verComponents[2], 10, 64)
		assert.Equal(t, expectedPatch, version.Patch)
		if hasPreRelease {
			assert.Equal(t, verComponents[3], version.Prerelease)
		} else {
			assert.Equal(t, "", version.Prerelease)
		}
		if hasBuild {
			assert.Equal(t, verComponents[4], version.Build)
		} else {
			assert.Equal(t, "", version.Build)
		}
	}
}

// TestInValidSemver Tests the SemVersion parser for valid version strings
func TestInValidSemver(t *testing.T) {
	invalidVersions := []string{
		"",
		"foo",
		"foo.1.0",
		"1.foo.0",
		"1.1.bar",
	}
	for _, verString := range invalidVersions {
		_, err := NewSemVersion(verString)
		assert.NotNil(t, err)
	}
}
