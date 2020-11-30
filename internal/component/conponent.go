// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	installv1alpha1 "github.com/verrazzano/verrazzano/api/v1alpha1"
)

// Component interface defines the methods implemented by components
type Component interface {
	Name() string
	Upgrade(cr *installv1alpha1.Verrazzano) error
}

// GetComponents returns the list of components that are installable and upgradeable.
// The components will be processed in the order items in the array
func GetComponents() []Component {
	return []Component{Verrazzano{}}
}
