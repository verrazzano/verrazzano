// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
)

// isUpgradeRequired Returns true if we detect that an upgrade is required but not (at least) in progress:
//   - if the Spec version IS NOT empty is less than the BOM version, an upgrade is required
//   - if the Spec version IS empty the Status version is less than the BOM, then an upgrade is required (upgrade of initial install scenario)
//
// If we return true here, it means we should stop reconciling until an upgrade has been requested
func (r Reconciler) isUpgradeRequired(actualCR *vzv1alpha1.Verrazzano) (bool, error) {
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
		return specVersion.IsLessThan(bomVersion), nil
	}
	if len(actualCR.Status.Version) > 0 {
		statusVersion, err := semver.NewSemVersion(actualCR.Status.Version)
		if err != nil {
			return false, err
		}
		return statusVersion.IsLessThan(bomVersion), nil
	}
	return false, nil
}
