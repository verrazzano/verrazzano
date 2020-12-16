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
		assert.NoError(t, err)
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
		assert.Error(t, err)
	}
}

// TestCompareVersion Tests comparisons of version field values
func TestCompareVersion(t *testing.T) {
	assert.Equal(t, -1, compareVersion(2, 1))
	assert.Equal(t, 1, compareVersion(1, 2))
	assert.Equal(t, 0, compareVersion(2, 2))
}

// TestCompareTo Tests comparisons between SemVersion instances
func TestCompareTo(t *testing.T) {

	v010, _ := NewSemVersion("v0.1.0")
	v010_2, _ := NewSemVersion("v0.1.0")
	v011, _ := NewSemVersion("v0.1.1")

	v020, _ := NewSemVersion("v0.2.0")
	v100, _ := NewSemVersion("v1.0.0")

	assert.Equal(t, 0, v010.CompareTo(v010_2))
	assert.Equal(t, -1, v010.CompareTo(v011))
	assert.Equal(t, -1, v010.CompareTo(v020))
	assert.Equal(t, 1, v020.CompareTo(v010))
	assert.Equal(t, 1, v020.CompareTo(v011))
	assert.Equal(t, -1, v020.CompareTo(v100))
	assert.Equal(t, 1, v100.CompareTo(v020))

	v0_0_9, _ := NewSemVersion("v0.0.9")
	v0_0_10, _ := NewSemVersion("v0.0.10")
	assert.Equal(t, 1, v0_0_10.CompareTo(v0_0_9))

	V100, err := NewSemVersion("V1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, 0, V100.CompareTo(v100))
}
