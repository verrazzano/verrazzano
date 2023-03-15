// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package semver

import "github.com/Masterminds/semver/v3"

func MatchesConstraint(version string, versionConstraint string) (bool, error) {
	v, err := semver.NewVersion(version)
	if err != nil {
		return false, err
	}
	constraint, err := semver.NewConstraint(versionConstraint)
	if err != nil {
		return false, err
	}
	if !constraint.Check(v) {
		return false, nil
	}
	return true, nil
}
