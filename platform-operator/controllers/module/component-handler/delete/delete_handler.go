// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package delete

import (
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulestatus "github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/handlerspi"
	vzerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/module/component-handler/common"
)

type ComponentHandler struct{}

var (
	_ handlerspi.StateMachineHandler = &ComponentHandler{}
)

func NewHandler() handlerspi.StateMachineHandler {
	return &ComponentHandler{}
}

// GetWorkName returns the work name
func (h ComponentHandler) GetWorkName() string {
	return "uninstall"
}

// IsWorkNeeded returns true if uninstall is needed
func (h ComponentHandler) IsWorkNeeded(ctx handlerspi.HandlerContext) (bool, result.Result) {
	// Always return true so that the post-uninstall can run in the case that the VPO
	// was restarted
	return true, result.NewResult()
}

// CheckDependencies checks if the dependencies are ready
func (h ComponentHandler) CheckDependencies(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// PreWorkUpdateStatus does the lifecycle pre-Work status update
func (h ComponentHandler) PreWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	// Update the Verrazzano component status
	nsn, err := common.GetVerrazzanoNSN(ctx)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	sd := common.StatusData{
		Vznsn:    *nsn,
		CondType: vzapi.CondUninstallStarted,
		CompName: module.Spec.ModuleName,
		Msg:      string(vzapi.CondUninstallStarted),
		Ready:    false,
	}
	res := common.UpdateVerrazzanoComponentStatus(ctx, sd)
	if res.ShouldRequeue() {
		return res
	}

	// Update the module status
	return modulestatus.UpdateReadyConditionReconciling(ctx, module, moduleapi.ReadyReasonUninstallStarted)
}

// PreWork does the pre-work
func (h ComponentHandler) PreWork(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	compCtx, comp, err := common.GetComponentAndContext(ctx, constants.UninstallOperation)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Do the pre-delete
	if err := comp.PreUninstall(compCtx); err != nil {
		if !vzerrors.IsRetryableError(err) {
			modulestatus.UpdateReadyConditionFailed(ctx, module, moduleapi.ReadyReasonUninstallStarted, err.Error())
		}
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// DoWorkUpdateStatus does the work status update
func (h ComponentHandler) DoWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// DoWork uninstalls the module using Helm
func (h ComponentHandler) DoWork(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	compCtx, comp, err := common.GetComponentAndContext(ctx, constants.UninstallOperation)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := comp.Uninstall(compCtx); err != nil {
		if !vzerrors.IsRetryableError(err) {
			modulestatus.UpdateReadyConditionFailed(ctx, module, moduleapi.ReadyReasonUninstallStarted, err.Error())
		}
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// IsWorkDone Indicates whether a module is uninstalled
func (h ComponentHandler) IsWorkDone(ctx handlerspi.HandlerContext) (bool, result.Result) {
	compCtx, comp, err := common.GetComponentAndContext(ctx, constants.UninstallOperation)
	if err != nil {
		return false, result.NewResultShortRequeueDelayWithError(err)
	}

	exists, err := comp.Exists(compCtx)
	if err != nil {
		ctx.Log.ErrorfThrottled("Error checking if Helm release exists for %s/%s", ctx.HelmRelease.Namespace, ctx.HelmRelease.Name)
		return false, result.NewResultShortRequeueDelayWithError(err)
	}
	return !exists, result.NewResult()
}

// PostWorkUpdateStatus does the post-work status update
func (h ComponentHandler) PostWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// PostWork does installation pre-work
func (h ComponentHandler) PostWork(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	compCtx, comp, err := common.GetComponentAndContext(ctx, constants.UninstallOperation)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if err := comp.PostUninstall(compCtx); err != nil {
		if !vzerrors.IsRetryableError(err) {
			modulestatus.UpdateReadyConditionFailed(ctx, module, moduleapi.ReadyReasonUninstallStarted, err.Error())
		}
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// WorkCompletedUpdateStatus does the lifecycle completed Work status update
func (h ComponentHandler) WorkCompletedUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	// Update the Verrazzano component status to disabled
	vzNSN, err := common.GetVerrazzanoNSN(ctx)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	res := common.UpdateVerrazzanoComponentStatusToDisabled(ctx, *vzNSN, module.Spec.ModuleName)
	if res.ShouldRequeue() {
		return res
	}

	// Update the module status
	return modulestatus.UpdateReadyConditionSucceeded(ctx, module, moduleapi.ReadyReasonUninstallSucceeded)
}
