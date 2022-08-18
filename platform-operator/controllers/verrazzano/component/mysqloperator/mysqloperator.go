// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

// GetOverrides gets the install overrides
func GetOverrides(effectiveCR *vzapi.Verrazzano) []vzapi.Overrides {
	if effectiveCR.Spec.Components.CertManager != nil {
		return effectiveCR.Spec.Components.CertManager.ValueOverrides
	}
	return []vzapi.Overrides{}
}
