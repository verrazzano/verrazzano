// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installupdate

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

type ActionType string

const (
	InstallAction ActionType = constants.InstallOperation
	UpdateAction  ActionType = constants.UpdateOperation
)

type ComponentHandler struct {
	action ActionType
}

var (
	_ handlerspi.StateMachineHandler = &ComponentHandler{}
)

func NewHandler(action ActionType) handlerspi.StateMachineHandler {
	return &ComponentHandler{action: action}
}

// GetWorkName returns the work name
func (h ComponentHandler) GetWorkName() string {
	return string(h.action)
}

// IsWorkNeeded returns true if install/update is needed
func (h ComponentHandler) IsWorkNeeded(ctx handlerspi.HandlerContext) (bool, result.Result) {
	return true, result.NewResult()
}

// CheckDependencies checks if the dependencies are ready
func (h ComponentHandler) CheckDependencies(ctx handlerspi.HandlerContext) result.Result {
	return common.CheckDependencies(ctx, string(h.action), h.getStartedReason())
}

// PreWorkUpdateStatus does the pre-Work status update
func (h ComponentHandler) PreWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	// Update the Verrazzano component status
	nsn, err := common.GetVerrazzanoNSN(ctx)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	sd := common.StatusData{
		Vznsn:    *nsn,
		CondType: vzapi.CondInstallStarted,
		CompName: module.Spec.ModuleName,
		Msg:      string(vzapi.CondInstallStarted),
		Ready:    false,
	}
	res := common.UpdateVerrazzanoComponentStatus(ctx, sd)
	if res.ShouldRequeue() {
		return res
	}

	// Update the module status
	return modulestatus.UpdateReadyConditionReconciling(ctx, module, h.getStartedReason())
}

// PreWork does the pre-work
func (h ComponentHandler) PreWork(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	compCtx, comp, err := common.GetComponentAndContext(ctx, string(h.action))
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Do the pre-install
	if err := comp.PreInstall(compCtx); err != nil {
		if !vzerrors.IsRetryableError(err) {
			modulestatus.UpdateReadyConditionFailed(ctx, module, h.getStartedReason(), err.Error())
		}
		return result.NewResultShortRequeueDelayWithError(err)
	}
	modulestatus.UpdateReadyConditionReconciling(ctx, module, h.getStartedReason())
	return result.NewResult()
}

// DoWorkUpdateStatus does th status update
func (h ComponentHandler) DoWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// DoWork installs the module using Helm
func (h ComponentHandler) DoWork(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	compCtx, comp, err := common.GetComponentAndContext(ctx, string(h.action))
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := comp.Install(compCtx); err != nil {
		if !vzerrors.IsRetryableError(err) {
			modulestatus.UpdateReadyConditionFailed(ctx, module, h.getStartedReason(), err.Error())
		}
		return result.NewResultShortRequeueDelayWithError(err)
	}
	modulestatus.UpdateReadyConditionReconciling(ctx, module, h.getStartedReason())
	return result.NewResult()
}

// IsWorkDone Indicates whether a module is installed and ready
func (h ComponentHandler) IsWorkDone(ctx handlerspi.HandlerContext) (bool, result.Result) {
	compCtx, comp, err := common.GetComponentAndContext(ctx, string(h.action))
	if err != nil {
		return false, result.NewResultShortRequeueDelayWithError(err)
	}

	ready := comp.IsReady(compCtx)
	return ready, result.NewResult()
}

// PostWorkUpdateStatus does the post-work status update
func (h ComponentHandler) PostWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// PostWork does installation post-work
func (h ComponentHandler) PostWork(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	compCtx, comp, err := common.GetComponentAndContext(ctx, string(h.action))
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if err := comp.PostInstall(compCtx); err != nil {
		if !vzerrors.IsRetryableError(err) {
			modulestatus.UpdateReadyConditionFailed(ctx, module, h.getStartedReason(), err.Error())
		}
		return result.NewResultShortRequeueDelayWithError(err)
	}
	modulestatus.UpdateReadyConditionReconciling(ctx, module, h.getStartedReason())
	return result.NewResult()
}

// WorkCompletedUpdateStatus updates the status to completed
func (h ComponentHandler) WorkCompletedUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)
	var reason moduleapi.ModuleConditionReason
	var cond vzapi.ConditionType

	if h.action == InstallAction {
		reason = moduleapi.ReadyReasonInstallSucceeded
		cond = vzapi.CondInstallComplete
	} else {
		reason = moduleapi.ReadyReasonUpdateSucceeded
		// Note, Verrazzano uses install condition for update
		cond = vzapi.CondInstallComplete
	}

	// Update the Verrazzano component status
	nsn, err := common.GetVerrazzanoNSN(ctx)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	sd := common.StatusData{
		Vznsn:       *nsn,
		CondType:    cond,
		CompName:    module.Spec.ModuleName,
		CompVersion: module.Spec.Version,
		Msg:         string(cond),
		Ready:       true,
	}
	res := common.UpdateVerrazzanoComponentStatus(ctx, sd)
	if res.ShouldRequeue() {
		return res
	}

	// Update the module status
	return modulestatus.UpdateReadyConditionSucceeded(ctx, module, reason)
}

func (h ComponentHandler) getStartedReason() moduleapi.ModuleConditionReason {
	if h.action == InstallAction {
		return moduleapi.ReadyReasonInstallStarted
	}
	return moduleapi.ReadyReasonUpdateStarted
}
