// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package factory

import (
	"github.com/verrazzano/verrazzano-modules/pkg/controller/handlerspi"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/module-integration/component-handler/delete"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/module-integration/component-handler/install"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/module-integration/component-handler/upgrade"
)

// NewModuleHandlerInfo creates a new ModuleHandlerInfo
func NewModuleHandlerInfo() handlerspi.ModuleHandlerInfo {
	return handlerspi.ModuleHandlerInfo{
		InstallActionHandler: install.NewHandler(),
		DeleteActionHandler:  delete.NewHandler(),

		// Update uses same handler as install
		UpdateActionHandler:  install.NewHandler(),
		UpgradeActionHandler: upgrade.NewHandler(),
	}
}
