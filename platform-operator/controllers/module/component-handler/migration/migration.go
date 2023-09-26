// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package migration

import (
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulestatus "github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/handlerspi"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/module/component-handler/common"
	"time"
)

type migrationHandler struct {
}

var (
	_ handlerspi.MigrationHandler = &migrationHandler{}
)

func NewHandler() handlerspi.MigrationHandler {
	return &migrationHandler{}
}

// UpdateStatusIfAlreadyInstalled handles the case where Verrazzano has already installed the component without modules,
// but not using module CRs. This happens when updating from 1.4, 1.5, 1.6 to 2.0. If that is the case, then the
// module.Status must get updated with installed component condition, version, etc.,
// so that it appears that it was installed by the module controller.
func (h migrationHandler) UpdateStatusIfAlreadyInstalled(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)
	vzcr, err := common.GetVerrazzanoCR(ctx)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	// If conditions exist then the module is being or has been reconciled.
	if module.Status.Conditions != nil {
		// no status update needed
		return result.NewResult()
	}

	// Check if component is installed.  If not then status doesn't need to be updated.
	_, comp, err := common.GetComponentAndContext(ctx, string(constants.InstallOperation))
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	compStatus, ok := vzcr.Status.Components[comp.Name()]
	if !ok {
		// This is a new component being installed, No status update needed
		return result.NewResult()
	}

	// If VZ status indicates that the component was deleted, then don't update the
	// status.  This can happen in special cases, such as VMO 1.5 -> 1.6 upgrade where
	// a module CR is created because a component is installed even though the VZ component
	// API says it is disabled.
	// Check both disabled and the uninstalled/uninstalling condition to see if is deleted.
	if compStatus.State == vzapi.CompStateDisabled {
		return result.NewResult()
	}
	cond := getLatestVzComponentCondition(ctx, compStatus)
	if cond == nil || cond.Type == vzapi.CondUninstallComplete || cond.Type == vzapi.CondUninstallStarted {
		return result.NewResult()
	}

	// Set the module status condition, installed generation and installed version
	return modulestatus.UpdateModuleStatusToInstalled(ctx, module, "v0.0.0", 0)
}

// Get the latest Verrazzano component condition based on the time stamp
func getLatestVzComponentCondition(ctx handlerspi.HandlerContext, compStatus *vzapi.ComponentStatusDetails) *vzapi.Condition {
	var latestCond *vzapi.Condition
	var latestTime *time.Time
	for i, cond := range compStatus.Conditions {
		// Sample vz cond time layout "2006-01-02T15:04:15Z"
		condTime, err := time.Parse(time.RFC3339, cond.LastTransitionTime)
		if err != nil {
			// nothing we can do in the case, return nil so an install is triggered
			ctx.Log.Oncef("Failed parsing Verrazzano condition time %s for component %s", cond.LastTransitionTime, compStatus.Name)
			return nil
		}
		if latestTime == nil {
			latestTime = &condTime
		}
		if latestCond == nil || condTime.After(*latestTime) {
			latestCond = &compStatus.Conditions[i]
			latestTime = &condTime
		}
	}
	return latestCond
}
