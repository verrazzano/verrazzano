// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package installupdate

import (
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulestatus "github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/handlerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/experimental/module-integration/component-handler/common"
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
	return "install"
}

// IsWorkNeeded returns true if install/update is needed
func (h ComponentHandler) IsWorkNeeded(ctx handlerspi.HandlerContext) (bool, result.Result) {
	return true, result.NewResult()
}

// PreWorkUpdateStatus does the pre-Work status update
func (h ComponentHandler) PreWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	var reason moduleapi.ModuleConditionReason
	var cond vzapi.ConditionType

	if h.action == InstallAction {
		reason = moduleapi.ReadyReasonInstallStarted
		cond = vzapi.CondInstallStarted
	} else {
		reason = moduleapi.ReadyReasonInstallStarted
		cond = vzapi.CondInstallStarted
	}

	// Update the module status
	module := ctx.CR.(*moduleapi.Module)
	res := modulestatus.UpdateReadyConditionReconciling(ctx, module, reason)
	if res.ShouldRequeue() {
		return res
	}

	// Update the Verrazzano component status
	nsn, err := common.GetVerrazzanoNSN(ctx)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	sd := common.StatusData{
		Vznsn:    *nsn,
		CondType: cond,
		CompName: module.Spec.ModuleName,
		Msg:      string(cond),
		Ready:    true,
	}
	return common.UpdateComponentStatus(ctx, sd)
}

// PreWork does the pre-work
func (h ComponentHandler) PreWork(ctx handlerspi.HandlerContext) result.Result {
	compCtx, comp, err := common.GetComponentAndContext(ctx, string(h.action))
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Wait for dependencies
	if !registry.ComponentDependenciesMet(comp, compCtx) {
		ctx.Log.Oncef("Component %s is waiting for dependenct components to be installed", comp.Name())
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Do the pre-install
	if err := comp.PreInstall(compCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// DoWorkUpdateStatus does th status update
func (h ComponentHandler) DoWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// DoWork installs the module using Helm
func (h ComponentHandler) DoWork(ctx handlerspi.HandlerContext) result.Result {
	compCtx, comp, err := common.GetComponentAndContext(ctx, string(h.action))
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := comp.Install(compCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
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
	compCtx, comp, err := common.GetComponentAndContext(ctx, string(h.action))
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if err := comp.PostInstall(compCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// WorkCompletedUpdateStatus updates the status to completed
func (h ComponentHandler) WorkCompletedUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
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

	module := ctx.CR.(*moduleapi.Module)
	res := modulestatus.UpdateReadyConditionSucceeded(ctx, module, reason)
	if res.ShouldRequeue() {
		return res
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
	return common.UpdateComponentStatus(ctx, sd)
}
