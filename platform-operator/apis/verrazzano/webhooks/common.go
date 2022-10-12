// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"github.com/Jeffail/gabs/v2"
	"github.com/verrazzano/verrazzano/pkg/semver"
	"sigs.k8s.io/yaml"
)

// isMinVersion indicates whether the provide version is greater than the min version provided
func isMinVersion(vzVersion, minVersion string) bool {
	vzSemver, err := semver.NewSemVersion(vzVersion)
	if err != nil {
		return false
	}
	minSemver, err := semver.NewSemVersion(minVersion)
	if err != nil {
		return false
	}
	return !vzSemver.IsLessThan(minSemver)
}

// extractValueFromOverrideString extracts  a given value from override.
func extractValueFromOverrideString(overrideStr string, field string) (interface{}, error) {
	jsonConfig, err := yaml.YAMLToJSON([]byte(overrideStr))
	if err != nil {
		return nil, err
	}
	jsonString, err := gabs.ParseJSON(jsonConfig)
	if err != nil {
		return nil, err
	}
	return jsonString.Path(field).Data(), nil
}
