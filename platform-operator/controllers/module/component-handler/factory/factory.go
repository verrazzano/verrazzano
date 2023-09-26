// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package factory

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/handlerspi"
	delete2 "github.com/verrazzano/verrazzano/platform-operator/controllers/module/component-handler/delete"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/module/component-handler/installupdate"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/module/component-handler/migration"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/module/component-handler/upgrade"
)

// NewModuleHandlerInfo creates a new ModuleHandlerInfo
func NewModuleHandlerInfo() handlerspi.ModuleHandlerInfo {
	return handlerspi.ModuleHandlerInfo{
		DeleteActionHandler:  delete2.NewHandler(),
		InstallActionHandler: installupdate.NewHandler(installupdate.InstallAction),
		UpdateActionHandler:  installupdate.NewHandler(installupdate.UpdateAction),
		UpgradeActionHandler: upgrade.NewHandler(),
		MigrationHandler:     migration.NewHandler(),
	}
}
