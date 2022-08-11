// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package semver

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestValidSemver Tests the SemVersion parser for valid version strings
// GIVEN a set of valid version strings
// WHEN we try to create a SemVersion
// THEN no error is returned and a valid SemVersion object ref is returned
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
		if !hasPreRelease && !hasBuild {
			assert.Equal(t, fmt.Sprintf("%s.%s.%s", verComponents[0], verComponents[1], verComponents[2]), version.ToString())
		}
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
// GIVEN a set of valid inversion strings
// WHEN we try to create a SemVersion
// THEN an error is returned and nil is returned for the SemVersion object ref
func TestInValidSemver(t *testing.T) {
	invalidVersions := []string{
		"",
		"foo",
		"foo.1.0",
		"1.foo.0",
		"1.1.bar",
	}
	for _, verString := range invalidVersions {
		v, err := NewSemVersion(verString)
		assert.Error(t, err)
		assert.Nil(t, v)
	}
}

// TestCompareVersion Tests comparisons of version field values
// GIVEN a call to compareVersion
// WHEN v1 > v2, v1 < v2, and v1 == v2
// THEN -1 is returned when v1 > v2, 1 when > v1 < v2, and 0 when v1 == v2
func TestCompareVersion(t *testing.T) {
	assert.Equal(t, -1, compareVersion(2, 1))
	assert.Equal(t, 1, compareVersion(1, 2))
	assert.Equal(t, 0, compareVersion(2, 2))
}

// TestCompareTo Tests comparisons between SemVersion instances
// GIVEN a call to CompareTo with different SemVersion objects
// WHEN v1 > v2, v1 < v2, and v1 == v2
// THEN -1 is returned when v1 > v2, 1 when > v1 < v2, and 0 when v1 == v2
func TestCompareTo(t *testing.T) {

	v010, _ := NewSemVersion("v0.1.0")
	vn010, _ := NewSemVersion("0.1.0")
	v010_2, _ := NewSemVersion("v0.1.0")
	v011, _ := NewSemVersion("v0.1.1")

	v020, _ := NewSemVersion("v0.2.0")
	v100, _ := NewSemVersion("v1.0.0")

	assert.Equal(t, 0, v010.CompareTo(vn010))
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

	v010deva, _ := NewSemVersion("v0.1.0-dev+aaaaaaaa")
	v010deva2, _ := NewSemVersion("v0.1.0-dev+aaaaaaaa")
	v010devb, _ := NewSemVersion("v0.1.0-dev+bbbbbbbb")
	v010proda, _ := NewSemVersion("v0.1.0-prod+aaaaaaaa")
	v020deva, _ := NewSemVersion("v0.2.0-dev+aaaaaaaa")
	v020devb, _ := NewSemVersion("v0.2.0-dev+bbbbbbbb")

	assert.Equal(t, 0, v010deva.CompareTo(v010deva2))
	assert.Equal(t, 1, v010deva.CompareTo(v010devb))
	assert.Equal(t, 1, v010deva.CompareTo(v010proda))
	assert.Equal(t, 1, v010deva.CompareTo(v010))
	assert.Equal(t, -1, v010deva.CompareTo(v020deva))
	assert.Equal(t, -1, v010deva.CompareTo(v020devb))

	V100, err := NewSemVersion("V1.0.0")
	assert.NoError(t, err)
	assert.Equal(t, 0, V100.CompareTo(v100))
}

// TestIsEqualTo Tests IsEqualTo for various combinations of SemVersion objects
// GIVEN a call to IsEqualTo with different SemVersion objects
// WHEN v > arg, v < arg, and v == arg
// THEN True v == arg, false otherwise
func TestIsEqualTo(t *testing.T) {
	v010, _ := NewSemVersion("v0.1.0")
	v010_2, _ := NewSemVersion("v0.1.0")
	v020, _ := NewSemVersion("v0.2.0")
	v011, _ := NewSemVersion("v0.1.1")
	v100, _ := NewSemVersion("v1.0.0")

	assert.True(t, v010.IsEqualTo(v010))
	assert.True(t, v010.IsEqualTo(v010_2))
	assert.False(t, v010.IsEqualTo(v020))
	assert.False(t, v010.IsEqualTo(v011))
	assert.False(t, v010.IsEqualTo(v100))

	assert.False(t, v020.IsEqualTo(v010))
	assert.False(t, v020.IsEqualTo(v010_2))
	assert.True(t, v020.IsEqualTo(v020))
	assert.False(t, v020.IsEqualTo(v011))
	assert.False(t, v020.IsEqualTo(v100))

	assert.False(t, v011.IsEqualTo(v010))
	assert.False(t, v011.IsEqualTo(v010_2))
	assert.False(t, v011.IsEqualTo(v020))
	assert.True(t, v011.IsEqualTo(v011))
	assert.False(t, v011.IsEqualTo(v100))

	assert.False(t, v100.IsEqualTo(v010))
	assert.False(t, v100.IsEqualTo(v010_2))
	assert.False(t, v100.IsEqualTo(v020))
	assert.False(t, v100.IsEqualTo(v011))
	assert.True(t, v100.IsEqualTo(v100))

	v009, _ := NewSemVersion("v0.0.9")
	v009_2, _ := NewSemVersion("v0.0.9")
	v0010, _ := NewSemVersion("v0.0.10")
	assert.True(t, v009.IsEqualTo(v009_2))
	assert.False(t, v009.IsEqualTo(v0010))
}

// TestIsLessThan Tests IsLessThan for various combinations of SemVersion objects
// GIVEN a call to IsLessThan with different SemVersion objects
// WHEN v > arg, v < arg, and v == arg
// THEN True v < arg, false otherwise
func TestIsLessThan(t *testing.T) {
	v010, _ := NewSemVersion("v0.1.0")
	v010_2, _ := NewSemVersion("v0.1.0")
	v020, _ := NewSemVersion("v0.2.0")
	v011, _ := NewSemVersion("v0.1.1")
	v100, _ := NewSemVersion("v1.0.0")
	v200, _ := NewSemVersion("v2.0.0")

	assert.False(t, v010.IsLessThan(v010))
	assert.False(t, v010.IsLessThan(v010_2))
	assert.True(t, v010.IsLessThan(v020))
	assert.True(t, v010.IsLessThan(v011))
	assert.True(t, v010.IsLessThan(v100))

	assert.False(t, v020.IsLessThan(v010))
	assert.False(t, v020.IsLessThan(v010_2))
	assert.False(t, v020.IsLessThan(v020))
	assert.False(t, v020.IsLessThan(v011))
	assert.True(t, v020.IsLessThan(v100))

	assert.False(t, v011.IsLessThan(v010))
	assert.False(t, v011.IsLessThan(v010_2))
	assert.True(t, v011.IsLessThan(v020))
	assert.False(t, v011.IsLessThan(v011))
	assert.True(t, v011.IsLessThan(v100))

	assert.False(t, v100.IsLessThan(v010))
	assert.False(t, v100.IsLessThan(v010_2))
	assert.False(t, v100.IsLessThan(v020))
	assert.False(t, v100.IsLessThan(v011))
	assert.False(t, v100.IsLessThan(v100))
	assert.True(t, v100.IsLessThan(v200))

	v009, _ := NewSemVersion("v0.0.9")
	v009_2, _ := NewSemVersion("v0.0.9")
	v0010, _ := NewSemVersion("v0.0.10")
	assert.False(t, v009.IsLessThan(v009_2))
	assert.True(t, v009.IsLessThan(v0010))
	assert.False(t, v0010.IsLessThan(v009))
}

// TestIsGreatherThan Tests IsGreatherThan for various combinations of SemVersion objects
// GIVEN a call to IsGreatherThan with different SemVersion objects
// WHEN v > arg, v < arg, and v == arg
// THEN True v > arg, false otherwise
func TestIsGreatherThan(t *testing.T) {
	v010, _ := NewSemVersion("v0.1.0")
	v010_2, _ := NewSemVersion("v0.1.0")
	v020, _ := NewSemVersion("v0.2.0")
	v011, _ := NewSemVersion("v0.1.1")
	v100, _ := NewSemVersion("v1.0.0")
	v200, _ := NewSemVersion("v2.0.0")

	assert.False(t, v010.IsGreatherThan(v010))
	assert.False(t, v010.IsGreatherThan(v010_2))
	assert.False(t, v010.IsGreatherThan(v020))
	assert.False(t, v010.IsGreatherThan(v011))
	assert.False(t, v010.IsGreatherThan(v100))

	assert.True(t, v020.IsGreatherThan(v010))
	assert.True(t, v020.IsGreatherThan(v010_2))
	assert.False(t, v020.IsGreatherThan(v020))
	assert.True(t, v020.IsGreatherThan(v011))
	assert.False(t, v020.IsGreatherThan(v100))

	assert.True(t, v011.IsGreatherThan(v010))
	assert.True(t, v011.IsGreatherThan(v010_2))
	assert.False(t, v011.IsGreatherThan(v020))
	assert.False(t, v011.IsGreatherThan(v011))
	assert.False(t, v011.IsGreatherThan(v100))

	assert.True(t, v100.IsGreatherThan(v010))
	assert.True(t, v100.IsGreatherThan(v010_2))
	assert.True(t, v100.IsGreatherThan(v020))
	assert.True(t, v100.IsGreatherThan(v011))
	assert.False(t, v100.IsGreatherThan(v100))
	assert.False(t, v100.IsGreatherThan(v200))

	v009, _ := NewSemVersion("v0.0.9")
	v009_2, _ := NewSemVersion("v0.0.9")
	v0010, _ := NewSemVersion("v0.0.10")
	assert.False(t, v009.IsGreatherThan(v009_2))
	assert.False(t, v009.IsGreatherThan(v0010))
	assert.True(t, v0010.IsGreatherThan(v009))
}

// TestToString Tests ToString function which converts a Semver object to its string representation
// GIVEN a call to ToString with different valid SemVersion objects
// WHEN the ToString returns the correct string
// THEN return Equal
func TestToString(t *testing.T) {
	v010, _ := NewSemVersion("v0.1.0")
	v010dev, _ := NewSemVersion("v0.1.0-dev")
	v010a, _ := NewSemVersion("v0.1.0+aaaaaaaa")
	v010deva, _ := NewSemVersion("v0.1.0-dev+aaaaaaaa")
	v010devaa, _ := NewSemVersion("v0.1.0-dev+aaaaaaaa-aaaaaaaa")
	v010aa, _ := NewSemVersion("v0.1.0+aaaaaaaa-aaaaaaaa")

	assert.Equal(t, "0.1.0", v010.ToString())
	assert.Equal(t, "0.1.0-dev", v010dev.ToString())
	assert.Equal(t, "0.1.0+aaaaaaaa", v010a.ToString())
	assert.Equal(t, "0.1.0-dev+aaaaaaaa", v010deva.ToString())
	assert.Equal(t, "0.1.0-dev+aaaaaaaa-aaaaaaaa", v010devaa.ToString())
	assert.Equal(t, "0.1.0+aaaaaaaa-aaaaaaaa", v010aa.ToString())
}

// TestCompareVersionSubstring Tests CompareVersionSubstring function which converts a Semver object to its string representation
// GIVEN a call to CompareVersionSubstring with two strings
// WHEN the Semver field is equivalent to the passed in string
// THEN return Equal
func TestCompareVersionSubstring(t *testing.T) {
	v010, _ := NewSemVersion("v0.1.0")
	v010dev, _ := NewSemVersion("v0.1.0-dev")
	v010a, _ := NewSemVersion("v0.1.0+aaaaaaaa")
	v010deva, _ := NewSemVersion("v0.1.0-dev+aaaaaaaa")
	v010devaa, _ := NewSemVersion("v0.1.0-dev+aaaaaaaa-aaaaaaaa")

	assert.Equal(t, 0, compareVersionSubstring(v010.Prerelease, ""))
	assert.Equal(t, 0, compareVersionSubstring(v010.Build, ""))
	assert.Equal(t, 1, compareVersionSubstring(v010dev.Prerelease, ""))
	assert.Equal(t, 0, compareVersionSubstring(v010dev.Prerelease, "dev"))
	assert.Equal(t, 0, compareVersionSubstring(v010dev.Build, ""))
	assert.Equal(t, 0, compareVersionSubstring(v010a.Prerelease, ""))
	assert.Equal(t, 1, compareVersionSubstring(v010a.Prerelease, "aaaaaaaa"))
	assert.Equal(t, 0, compareVersionSubstring(v010a.Build, "aaaaaaaa"))
	assert.Equal(t, 0, compareVersionSubstring(v010deva.Prerelease, "dev"))
	assert.Equal(t, 1, compareVersionSubstring(v010deva.Build, ""))
	assert.Equal(t, 0, compareVersionSubstring(v010deva.Build, "aaaaaaaa"))
	assert.Equal(t, 1, compareVersionSubstring(v010devaa.Build, "aaaaaaaa"))
	assert.Equal(t, 1, compareVersionSubstring(v010devaa.Prerelease, "aaaaaaaa"))
	assert.Equal(t, 0, compareVersionSubstring(v010devaa.Build, "aaaaaaaa-aaaaaaaa"))
}

// TestMissingPrefix Tests if a SemVersion is the same if the prefix v is missing
// GIVEN a call to IsEqualTo with SemVersion objects where the only difference is the prefix
// WHEN the comparison functions are called
// THEN the correct result is returned
func TestMissingPrefix(t *testing.T) {
	v010v, _ := NewSemVersion("v0.1.0")
	v010, _ := NewSemVersion("0.1.0")
	v011, _ := NewSemVersion("v0.1.1")
	v011v, _ := NewSemVersion("0.1.1")

	assert.True(t, v010.IsEqualTo(v010v))
	assert.True(t, v010.IsLessThan(v011v))
	assert.True(t, v010.IsLessThan(v011))
	assert.True(t, v010v.IsLessThan(v011v))
	assert.True(t, v010v.IsLessThan(v011))
	assert.True(t, v011.IsGreatherThan(v010v))
	assert.True(t, v011.IsGreatherThan(v010))
	assert.True(t, v011v.IsGreatherThan(v010))
	assert.True(t, v011v.IsGreatherThan(v010))
}

// TestIsGreatherThanOrEqualTo Tests IsGreatherThanOrEqualTo for various combinations of SemVersion objects
// GIVEN a call to IsGreatherThanOrEqualTo with different SemVersion objects
// WHEN v > arg, v < arg, and v == arg
// THEN True v >= arg, false otherwise
func TestIsGreatherThanOrEqualTo(t *testing.T) {
	v010, _ := NewSemVersion("v0.1.0")
	v010_2, _ := NewSemVersion("v0.1.0")
	v020, _ := NewSemVersion("v0.2.0")
	v011, _ := NewSemVersion("v0.1.1")
	v100, _ := NewSemVersion("v1.0.0")
	v200, _ := NewSemVersion("v2.0.0")

	assert.True(t, v010.IsGreaterThanOrEqualTo(v010))
	assert.True(t, v010.IsGreaterThanOrEqualTo(v010_2))
	assert.False(t, v010.IsGreaterThanOrEqualTo(v020))
	assert.False(t, v010.IsGreaterThanOrEqualTo(v011))
	assert.False(t, v010.IsGreaterThanOrEqualTo(v100))

	assert.True(t, v020.IsGreaterThanOrEqualTo(v010))
	assert.True(t, v020.IsGreaterThanOrEqualTo(v010_2))
	assert.True(t, v020.IsGreaterThanOrEqualTo(v020))
	assert.True(t, v020.IsGreaterThanOrEqualTo(v011))
	assert.False(t, v020.IsGreaterThanOrEqualTo(v100))

	assert.True(t, v011.IsGreaterThanOrEqualTo(v010))
	assert.True(t, v011.IsGreaterThanOrEqualTo(v010_2))
	assert.False(t, v011.IsGreaterThanOrEqualTo(v020))
	assert.True(t, v011.IsGreaterThanOrEqualTo(v011))
	assert.False(t, v011.IsGreaterThanOrEqualTo(v100))

	assert.True(t, v100.IsGreaterThanOrEqualTo(v010))
	assert.True(t, v100.IsGreaterThanOrEqualTo(v010_2))
	assert.True(t, v100.IsGreaterThanOrEqualTo(v020))
	assert.True(t, v100.IsGreaterThanOrEqualTo(v011))
	assert.True(t, v100.IsGreaterThanOrEqualTo(v100))
	assert.False(t, v100.IsGreaterThanOrEqualTo(v200))

	v009, _ := NewSemVersion("v0.0.9")
	v009_2, _ := NewSemVersion("v0.0.9")
	v0010, _ := NewSemVersion("v0.0.10")
	assert.True(t, v009.IsGreaterThanOrEqualTo(v009_2))
	assert.False(t, v009.IsGreaterThanOrEqualTo(v0010))
	assert.True(t, v0010.IsGreaterThanOrEqualTo(v009))
	assert.True(t, v009.IsGreaterThanOrEqualTo(v009))
}
