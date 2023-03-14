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
