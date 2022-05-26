// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package modules

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

const ControllerLabel = "verrazzano.io/module"

type DelegateReconciler interface {
	ReconcileModule(ctx spi.ComponentContext) error
	SetStatusWriter(statusWriter clipkg.StatusWriter)
}
