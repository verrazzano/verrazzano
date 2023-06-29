// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package factory

import (
	"github.com/verrazzano/verrazzano-modules/common/handlerspi"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/handlers/delete"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/handlers/reconcile"
)

// NewModuleHandlerInfo creates a new NewModuleHandlerInfo
func NewModuleHandlerInfo() handlerspi.ModuleHandlerInfo {
	return handlerspi.ModuleHandlerInfo{
		ReconcileActionHandler: reconcile.NewHandler(),
		DeleteActionHandler:    delete.NewHandler(),
	}
}
