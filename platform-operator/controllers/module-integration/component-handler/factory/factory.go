// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package factory

import (
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/handlers/helm/delete"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/handlers/helm/install"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/handlers/helm/update"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/handlers/helm/upgrade"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/handlerspi"
)

// NewModuleHandlerInfo creates a new ModuleHandlerInfo
func NewModuleHandlerInfo() handlerspi.ModuleHandlerInfo {
	return handlerspi.ModuleHandlerInfo{
		InstallActionHandler: install.NewHandler(),
		DeleteActionHandler:  delete.NewHandler(),
		UpdateActionHandler:  update.NewHandler(),
		UpgradeActionHandler: upgrade.NewHandler(),
	}
}
