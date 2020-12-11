// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"regexp"
	"strconv"
)

const semverRegex = "^v(0|[1-9]\\d*)\\.(0|[1-9]\\d*)\\.(0|[1-9]\\d*)(?:-((?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\\.(?:0|[1-9]\\d*|\\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\\+([0-9a-zA-Z-]+(?:\\.[0-9a-zA-Z-]+)*))?$"

// SemVersion Implements a basic notion a semantic version (see https://semver.org/)
type SemVersion struct {
	Major      int64
	Minor      int64
	Patch      int64
	Prerelease string
	Build      string
}

// NewSemVersion Create an instance of a SemVersion
func NewSemVersion(version string) (*SemVersion, error) {
	compile, err := regexp.Compile(semverRegex)
	if err != nil {
		return nil, err
	}
	versionComponents := compile.Split(version[1:], -1)

	majorVer, err := strconv.ParseInt(versionComponents[0], 10, 64)
	if err != nil {
		return nil, err
	}
	minorVer, err := strconv.ParseInt(versionComponents[0], 10, 64)
	if err != nil {
		return nil, err
	}
	patchVer, err := strconv.ParseInt(versionComponents[0], 10, 64)
	if err != nil {
		return nil, err
	}

	var prereleaseVer string
	if len(versionComponents) > 3 {
		prereleaseVer = versionComponents[3]
	}

	var buildVer string
	if len(versionComponents) > 4 {
		buildVer = versionComponents[4]
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

// Compare Compares
func (from *SemVersion) Compare(to *SemVersion) int {
	var result int
	if result = compareVersion(from.Major, to.Major); result == 0 {
		if result = compareVersion(from.Minor, to.Minor); result == 0 {
			result = compareVersion(from.Patch, to.Patch)
			// Ignore pre-release/buildver fields for now
		}
	}
	return result
}

func compareVersion(v1 int64, v2 int64) int {
	if v1 < v2 {
		return -1
	}
	if v1 > v2 {
		return 1
	}
	return 0
}
