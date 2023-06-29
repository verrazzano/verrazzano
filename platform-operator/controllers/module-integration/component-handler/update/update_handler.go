// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package update

import (
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/handlers/common"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/handlerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
)

type ComponentHandler struct {
	common.BaseHandler
}

var (
	_ handlerspi.StateMachineHandler = &ComponentHandler{}
)

func NewHandler() handlerspi.StateMachineHandler {
	return &ComponentHandler{}
}

// GetWorkName returns the work name
func (h ComponentHandler) GetWorkName() string {
	return "update"
}

// IsWorkNeeded returns true if update is needed
func (h ComponentHandler) IsWorkNeeded(ctx handlerspi.HandlerContext) (bool, result.Result) {
	return true, result.NewResult()
}

// PreWorkUpdateStatus updates the status for the pre-work state
func (h ComponentHandler) PreWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// PreWork does the pre-work
func (h ComponentHandler) PreWork(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// DoWorkUpdateStatus updates the status for the work state
func (h ComponentHandler) DoWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)
	return status.UpdateReadyConditionReconciling(ctx, module, moduleapi.ReadyReasonUpdateStarted)
}

// DoWork updates the module using Helm
func (h ComponentHandler) DoWork(ctx handlerspi.HandlerContext) result.Result {
	return h.HelmUpgradeOrInstall(ctx)
}

// IsWorkDone Indicates whether a module is updated and ready
func (h ComponentHandler) IsWorkDone(ctx handlerspi.HandlerContext) (bool, result.Result) {
	return h.CheckReleaseDeployedAndReady(ctx)
}

// PostWorkUpdateStatus does the post-work status update
func (h ComponentHandler) PostWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// PostWork does installation pre-work
func (h ComponentHandler) PostWork(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// WorkCompletedUpdateStatus updates the status to completed
func (h ComponentHandler) WorkCompletedUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)
	return status.UpdateReadyConditionSucceeded(ctx, module, moduleapi.ReadyReasonUpdateSucceeded)
}
