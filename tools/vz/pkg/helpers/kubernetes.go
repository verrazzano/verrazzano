// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helpers

import (
	v80clientset "github.com/verrazzano/verrazzano/platform-operator/clients/verrazzano/clientset/versioned"
)

type Kubernetes interface {
	NewVerrazzanoClientSet() (v80clientset.Interface, error)
}
