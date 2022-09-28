// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package semver

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

const semverRegex = "^[v|V](0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)(?:-((?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\\.(?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\\+([0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))?$"

// SemVersion Implements a basic notion a semantic version (see https://semver.org/, test page: https://regex101.com/r/vkijKf/1/)
type SemVersion struct {
	Major      int64
	Minor      int64
	Patch      int64
	Prerelease string
	Build      string
}

var compiledRegEx *regexp.Regexp

func getRegex() (*regexp.Regexp, error) {
	if compiledRegEx != nil {
		return compiledRegEx, nil
	}
	var err error
	compiledRegEx, err = regexp.Compile(semverRegex)
	if err != nil {
		return nil, err
	}
	return compiledRegEx, nil
}

// NewSemVersion Create an instance of a SemVersion
func NewSemVersion(version string) (*SemVersion, error) {
	if len(version) == 0 {
		return nil, errors.New("SemVersion string cannot be empty")
	}
	if !strings.HasPrefix(version, "v") && !strings.HasPrefix(version, "V") {
		version = "v" + version
	}
	regex, err := getRegex()
	if err != nil {
		return nil, err
	}

	allMatches := regex.FindAllStringSubmatch(version, -1)
	zap.S().Debugf("allMatches: %v", allMatches)
	if len(allMatches) == 0 {
		return nil, fmt.Errorf("Invalid version string %s", version)
	}

	versionComponents := allMatches[0]
	zap.S().Debugf("components: %v", versionComponents)
	numComponents := len(versionComponents)
	if numComponents < 3 {
		return nil, fmt.Errorf("Invalid version string %s", version)
	}
	majorVer, err := strconv.ParseInt(versionComponents[1], 10, 64)
	if err != nil {
		return nil, err
	}
	minorVer, err := strconv.ParseInt(versionComponents[2], 10, 64)
	if err != nil {
		return nil, err
	}

	var patchVer int64
	if numComponents > 3 {
		patchVer, err = strconv.ParseInt(versionComponents[3], 10, 64)
		if err != nil {
			return nil, err
		}
	}

	var prereleaseVer string
	if numComponents > 4 {
		prereleaseVer = versionComponents[4]
	}

	var buildVer string
	if numComponents > 5 {
		buildVer = versionComponents[5]
	}
	semVersion := SemVersion{
		Major:      majorVer,
		Minor:      minorVer,
		Patch:      patchVer,
		Prerelease: prereleaseVer,
		Build:      buildVer,
	}
	return &semVersion, nil
}

// CompareTo Compares the current version to another version
// - if from > this, -1 is returned
// - if from < this, 1 is returned
// - if they are equal, 0 is returned
func (v *SemVersion) CompareTo(from *SemVersion) int {
	var result int
	if result = compareVersion(from.Major, v.Major); result == 0 {
		if result = compareVersion(from.Minor, v.Minor); result == 0 {
			if result = compareVersion(from.Patch, v.Patch); result == 0 {
				if result = compareVersionSubstring(from.Prerelease, v.Prerelease); result == 0 {
					result = compareVersionSubstring(from.Build, v.Build)
				}
			}
		}
	}
	return result
}

// IsEqualTo Returns true if to == from
func (v *SemVersion) IsEqualTo(from *SemVersion) bool {
	return v.CompareTo(from) == 0
}

// IsGreatherThan Returns true if to > from
func (v *SemVersion) IsGreatherThan(from *SemVersion) bool {
	return v.CompareTo(from) > 0
}

// IsLessThan Returns true if to < from
func (v *SemVersion) IsLessThan(from *SemVersion) bool {
	return v.CompareTo(from) < 0
}

// ToString Convert to a valid semver string representation
func (v *SemVersion) ToString() string {
	if v.Build != "" && v.Prerelease != "" {
		return fmt.Sprintf("%v.%v.%v-%v+%v", v.Major, v.Minor, v.Patch, v.Prerelease, v.Build)
	} else if v.Build == "" && v.Prerelease != "" {
		return fmt.Sprintf("%v.%v.%v-%v", v.Major, v.Minor, v.Patch, v.Prerelease)
	} else if v.Build != "" && v.Prerelease == "" {
		return fmt.Sprintf("%v.%v.%v+%v", v.Major, v.Minor, v.Patch, v.Build)
	} else {
		return fmt.Sprintf("%v.%v.%v", v.Major, v.Minor, v.Patch)
	}
}

// Returns
// - 1 if v2 > v1
// - -1 if v1 > v2
// - 0 of v1 == v2
func compareVersion(v1 int64, v2 int64) int {
	if v1 < v2 {
		return 1
	}
	if v1 > v2 {
		return -1
	}
	return 0
}

// Returns 0 if the strings are equal, or 1 if not
func compareVersionSubstring(v1 string, v2 string) int {
	if strings.Compare(v1, v2) == 0 {
		return 0
	}
	return 1
}

// IsGreaterThanOrEqualTo Returns true if to >= from
func (v *SemVersion) IsGreaterThanOrEqualTo(from *SemVersion) bool {
	return v.IsGreatherThan(from) || v.IsEqualTo(from)
}
