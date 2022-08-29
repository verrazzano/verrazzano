// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package v1beta1

import (
	"fmt"
)

// ValidateInstallOverrides checks that the overrides slice has only one override type per slice item
func ValidateInstallOverrides(Overrides []Overrides) error {
	overridePerItem := 0
	for _, override := range Overrides {
		if override.ConfigMapRef != nil {
			overridePerItem++
		}
		if override.SecretRef != nil {
			overridePerItem++
		}
		if override.Values != nil {
			overridePerItem++
		}
		if overridePerItem > 1 {
			return fmt.Errorf("Invalid install overrides. Cannot specify more than one override type in the same list element")
		}
		if overridePerItem == 0 {
			return fmt.Errorf("Invalid install overrides. No override specified")
		}
		overridePerItem = 0
	}
	return nil
}
