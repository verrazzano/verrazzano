// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"

	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
)

// IsUpgradeRequired Returns true if the BOM version is greather than CR spec or status version
func IsUpgradeRequired(actualCR *vzapi.Verrazzano) (bool, error) {
	if actualCR == nil {
		return false, fmt.Errorf("no Verrazzano CR provided")
	}
	bomVersion, err := validators.GetCurrentBomVersion()
	if err != nil {
		return false, err
	}

	if len(actualCR.Spec.Version) > 0 {
		specVersion, err := semver.NewSemVersion(actualCR.Spec.Version)
		if err != nil {
			return false, err
		}
		return bomVersion.IsGreatherThan(specVersion), nil
	}
	if len(actualCR.Status.Version) > 0 {
		statusVersion, err := semver.NewSemVersion(actualCR.Status.Version)
		if err != nil {
			return false, err
		}
		return bomVersion.IsGreatherThan(statusVersion), nil
	}
	return false, nil
}
