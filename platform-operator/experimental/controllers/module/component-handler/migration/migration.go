// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package migration

import (
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulestatus "github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/handlerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/controllers/module/component-handler/common"
)

type migrationHandler struct {
}

var (
	_ handlerspi.MigrationHandler = &migrationHandler{}
)

func NewHandler() handlerspi.MigrationHandler {
	return &migrationHandler{}
}

// UpdateStatusIfAlreadyInstalled handles the case where Verrazzano has already installed the component without modules, but not using modules.
// If that is the case, then the module.Status must get updated with installed component condition, version, etc.,
// so that it appears that it was installed by the module controller.
func (h migrationHandler) UpdateStatusIfAlreadyInstalled(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)
	vzcr, err := common.GetVerrazzanoCR(ctx)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	// If conditions exist then the module is being or has been reconciled.
	if module.Status.Conditions != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Check if component is installed.  If not then status doesn't need to be updated.
	_, comp, err := common.GetComponentAndContext(ctx, string(constants.InstallOperation))
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	compStatus, ok := vzcr.Status.Components[comp.Name()]
	if !ok {
		return result.NewResult()
	}
	var installed bool
	for _, compCond := range compStatus.Conditions {
		if compCond.Type == vzapi.CondInstallComplete {
			installed = true
			break
		}
	}
	if !installed {
		return result.NewResult()
	}

	// Set the module status condition, installed generation and installed version
	return modulestatus.UpdateModuleStatusToInstalled(ctx, module, vzcr.Status.Version, compStatus.LastReconciledGeneration)
}
