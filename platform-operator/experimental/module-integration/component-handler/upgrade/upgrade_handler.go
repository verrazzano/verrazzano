// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package upgrade

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

type ComponentHandler struct{}

var (
	_ handlerspi.StateMachineHandler = &ComponentHandler{}
)

func NewHandler() handlerspi.StateMachineHandler {
	return &ComponentHandler{}
}

// GetWorkName returns the work name
func (h ComponentHandler) GetWorkName() string {
	return "upgrade"
}

// IsWorkNeeded returns true if upgrade is needed
func (h ComponentHandler) IsWorkNeeded(ctx handlerspi.HandlerContext) (bool, result.Result) {
	compCtx, comp, err := common.GetComponentAndContext(ctx, constants.UpgradeOperation)
	if err != nil {
		return false, result.NewResultShortRequeueDelayWithError(err)
	}

	installed, err := comp.IsInstalled(compCtx)
	if err != nil {
		ctx.Log.ErrorfThrottled("Error checking if Helm release installed for %s/%s", ctx.HelmRelease.Namespace, ctx.HelmRelease.Name)
		return true, result.NewResult()
	}
	return installed, result.NewResult()
}

// PreWorkUpdateStatus updates the status for the pre-work state
func (h ComponentHandler) PreWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	// Update the module status
	module := ctx.CR.(*moduleapi.Module)
	res := modulestatus.UpdateReadyConditionReconciling(ctx, module, moduleapi.ReadyReasonUpgradeStarted)
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
		CondType: vzapi.CondUpgradeStarted,
		CompName: module.Spec.ModuleName,
		Msg:      string(vzapi.CondUpgradeStarted),
		Ready:    true,
	}
	return common.UpdateVerrazzanoComponentStatus(ctx, sd)
}

// PreWork does the pre-work
func (h ComponentHandler) PreWork(ctx handlerspi.HandlerContext) result.Result {
	compCtx, comp, err := common.GetComponentAndContext(ctx, constants.UpgradeOperation)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Wait for dependencies
	if !registry.ComponentDependenciesMet(comp, compCtx) {
		ctx.Log.Oncef("Component %s is waiting for dependenct components to be installed", comp.Name())
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Do the pre-upgrade
	if err := comp.PreUpgrade(compCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// DoWorkUpdateStatus updates the status for the work state
func (h ComponentHandler) DoWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// DoWork upgrades the module using Helm
func (h ComponentHandler) DoWork(ctx handlerspi.HandlerContext) result.Result {
	compCtx, comp, err := common.GetComponentAndContext(ctx, constants.UpgradeOperation)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if err := comp.Upgrade(compCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// IsWorkDone indicates whether a module is upgraded and ready
func (h ComponentHandler) IsWorkDone(ctx handlerspi.HandlerContext) (bool, result.Result) {
	compCtx, comp, err := common.GetComponentAndContext(ctx, constants.UpgradeOperation)
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

// PostWork does installation pre-work
func (h ComponentHandler) PostWork(ctx handlerspi.HandlerContext) result.Result {
	compCtx, comp, err := common.GetComponentAndContext(ctx, constants.UpgradeOperation)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	if err := comp.PostUpgrade(compCtx); err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// WorkCompletedUpdateStatus updates the status to completed
func (h ComponentHandler) WorkCompletedUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)
	res := modulestatus.UpdateReadyConditionSucceeded(ctx, module, moduleapi.ReadyReasonUpgradeSucceeded)
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
		CondType:    vzapi.CondUpgradeComplete,
		CompName:    module.Spec.ModuleName,
		CompVersion: module.Spec.Version,
		Msg:         string(vzapi.CondUpgradeComplete),
		Ready:       true,
	}
	return common.UpdateVerrazzanoComponentStatus(ctx, sd)
}
