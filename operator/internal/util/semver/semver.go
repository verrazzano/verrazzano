// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package semver

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"regexp"
	"strconv"
	"sync"
)

const semverRegex = "^v(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)(?:-((?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\\.(?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\\+([0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))?$"

// SemVersion Implements a basic notion a semantic version (see https://semver.org/, test page: https://regex101.com/r/vkijKf/1/)
type SemVersion struct {
	Major         int64
	Minor         int64
	Patch         int64
	Prerelease    string
	Build         string
	VersionString string
}

var _regex *regexp.Regexp = nil
var _initRegEx sync.Mutex

func getRegex() *regexp.Regexp {
	if _regex != nil {
		return _regex
	}
	_initRegEx.Lock()
	defer _initRegEx.Unlock()
	var err error
	_regex, err = regexp.Compile(semverRegex)
	if err != nil {
		panic(fmt.Sprintf("Error compiling semver regex: %v", err))
	}
	return _regex
}

// NewSemVersion Create an instance of a SemVersion
func NewSemVersion(version string) (*SemVersion, error) {
	if len(version) == 0 {
		return nil, errors.New("SemVersion string can not be empty")
	}

	regex := getRegex()

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

	var patchVer int64 = 0
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
		Major:         majorVer,
		Minor:         minorVer,
		Patch:         patchVer,
		Prerelease:    prereleaseVer,
		Build:         buildVer,
		VersionString: version,
	}
	return &semVersion, nil
}

// CompareTo Compares the current version to another version
// - if from > this, -1 is returned
// - if from < this, 1 is returned
// - if they are equal, 0 is returned
func (to *SemVersion) CompareTo(from *SemVersion) int {
	var result int
	if result = compareVersion(from.Major, to.Major); result == 0 {
		if result = compareVersion(from.Minor, to.Minor); result == 0 {
			result = compareVersion(from.Patch, to.Patch)
			// Ignore pre-release/buildver fields for now
		}
	}
	return result
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
