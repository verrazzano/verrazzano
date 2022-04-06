// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// HelmManagedResource provides an object type and name for a resource managed within a helm chart
type HelmManagedResource struct {
	Obj            controllerutil.Object
	NamespacedName types.NamespacedName
}
