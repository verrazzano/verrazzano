// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package install

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/handlers/common"
	"github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/module-operator/internal/handlerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/constants"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
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
	return "install"
}

// IsWorkNeeded returns true if install is needed
func (h HelmHandler) IsWorkNeeded(ctx handlerspi.HandlerContext) (bool, result.Result) {
	return true, result.NewResult()
}

// PreWorkUpdateStatus does the pre-Work status update
func (h HelmHandler) PreWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// PreWork does the pre-work
func (h HelmHandler) PreWork(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)

	// Create the target namespace (if it doesn't exist) and label it
	if module.Spec.TargetNamespace != "" {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: module.Spec.TargetNamespace}}
		_, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client, ns,
			func() error {
				if ns.Labels == nil {
					ns.Labels = map[string]string{}
				}
				ns.Labels[constants.VerrazzanoNamespaceLabel] = ns.Name
				return nil
			},
		)
		if err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}

	// Update the spec version if it is not set
	if len(module.Spec.Version) == 0 {
		// Update spec version to match chart, always requeue to get ModuleCR with version
		module.Spec.Version = ctx.ChartInfo.Version
		if err := ctx.Client.Update(context.TODO(), module); err != nil {
			return result.NewResultShortRequeueDelay()
		}
		// ALways reconcile so that we get a new tracker with the latest ModuleCR
		return result.NewResultShortRequeueDelay()
	}

	return result.NewResult()
}

// DoWorkUpdateStatus does th status update
func (h HelmHandler) DoWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)
	return status.UpdateReadyConditionReconciling(ctx, module, moduleapi.ReadyReasonInstallStarted)
}

// DoWork installs the module using Helm
func (h HelmHandler) DoWork(ctx handlerspi.HandlerContext) result.Result {
	return h.HelmUpgradeOrInstall(ctx)
}

// IsWorkDone Indicates whether a module is installed and ready
func (h HelmHandler) IsWorkDone(ctx handlerspi.HandlerContext) (bool, result.Result) {
	return h.CheckReleaseDeployedAndReady(ctx)
}

// PostWorkUpdateStatus does the post-work status update
func (h HelmHandler) PostWorkUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// PostWork does installation post-work
func (h HelmHandler) PostWork(ctx handlerspi.HandlerContext) result.Result {
	return result.NewResult()
}

// WorkCompletedUpdateStatus updates the status to completed
func (h HelmHandler) WorkCompletedUpdateStatus(ctx handlerspi.HandlerContext) result.Result {
	module := ctx.CR.(*moduleapi.Module)
	return status.UpdateReadyConditionSucceeded(ctx, module, moduleapi.ReadyReasonInstallSucceeded)
}
