// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package delete

import (
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/handlers/common"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/module-operator/internal/handlerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/helm"
)

type HelmHandler struct {
	common.BaseHandler
}

var (
	_ handlerspi.StateMachineHandler = &HelmHandler{}
)

func NewHandler() handlerspi.StateMachineHandler {
	return &HelmHandler{}
}

// GetWorkName returns the work name
func (h HelmHandler) GetWorkName() string {
	return "uninstall"
}

// IsWorkNeeded returns true if install is needed
func (h HelmHandler) IsWorkNeeded(ctx handlerspi.HandlerContext) (bool, result.Result) {
	return true, result.NewResult()
}

// PreWorkUpdateStatus does the lifecycle pre-Work status update
func (h HelmHandler) PreWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// PreWork does the pre-work
func (h HelmHandler) PreWork(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// DoWorkUpdateStatus does the work status update
func (h HelmHandler) DoWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)
	return status.UpdateReadyConditionReconciling(ctx, module, moduleapi.ReadyReasonUninstallStarted)
}

// DoWork uninstalls the module using Helm
func (h HelmHandler) DoWork(ctx handlerspi.HandlerContext) result.Result {
	installed, err := helm.IsReleaseInstalled(ctx.HelmRelease.Name, ctx.HelmRelease.Namespace)
	if err != nil {
		ctx.Log.ErrorfThrottled("Error checking if Helm release installed for %s/%s", ctx.HelmRelease.Namespace, ctx.HelmRelease.Name)
		return result.NewResult()
	}
	if !installed {
		return result.NewResult()
	}

	err = helm.Uninstall(ctx.Log, ctx.HelmRelease.Name, ctx.HelmRelease.Namespace, ctx.DryRun)
	return result.NewResultShortRequeueDelayIfError(err)
}

// IsWorkDone Indicates whether a module is uninstalled
func (h HelmHandler) IsWorkDone(ctx handlerspi.HandlerContext) (bool, result.Result) {
	if ctx.DryRun {
		ctx.Log.Debugf("IsReady() dry run for %s", ctx.HelmRelease.Name)
		return true, result.NewResult()
	}

	deployed, err := helm.IsReleaseDeployed(ctx.HelmRelease.Name, ctx.HelmRelease.Namespace)
	if err != nil {
		ctx.Log.ErrorfThrottled("Error occurred checking release deployment: %v", err.Error())
		return false, result.NewResultShortRequeueDelayIfError(err)
	}

	return !deployed, result.NewResult()
}

// PostWorkUpdateStatus does the post-work status update
func (h HelmHandler) PostWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// PostWork does installation pre-work
func (h HelmHandler) PostWork(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// WorkCompletedUpdateStatus does the lifecycle completed Work status update
func (h HelmHandler) WorkCompletedUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)
	return status.UpdateReadyConditionSucceeded(ctx, module, moduleapi.ReadyReasonUninstallSucceeded)
}
