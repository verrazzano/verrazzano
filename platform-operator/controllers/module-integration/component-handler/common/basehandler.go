// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/common/handlerspi"
	"github.com/verrazzano/verrazzano-modules/common/pkg/controller/util"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type BaseHandler struct {
	Config       handlerspi.StateMachineHandlerConfig
	ModuleCR     *moduleapi.Module
	Action       moduleapi.ModuleLifecycleActionType
	MlcName      string
	MlcNamespace string
}

func (h *BaseHandler) GetActionName() string {
	//TODO implement me
	return "unknown"
}

// Init initializes the handler with Helm chart information
func (h *BaseHandler) Init(_ handlerspi.HandlerContext, config handlerspi.StateMachineHandlerConfig, action moduleapi.ModuleLifecycleActionType) (ctrl.Result, error) {
	h.Config = config
	h.ModuleCR = config.CR.(*moduleapi.Module)
	h.MlcName = DeriveModuleLifeCycleName(h.ModuleCR.Name, moduleapi.HelmLifecycleClass, action)
	h.MlcNamespace = h.ModuleCR.Namespace
	h.Action = action
	return ctrl.Result{}, nil
}

// IsActionNeeded returns true if install is needed
func (h BaseHandler) IsActionNeeded(ctx handlerspi.HandlerContext) (bool, ctrl.Result, error) {
	return true, ctrl.Result{}, nil
}

// PreActionUpdateStatus does the lifecycle pre-Action status update
func (h *BaseHandler) PreActionUpdateStatus(context handlerspi.HandlerContext) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// PreAction does the pre-action
func (h BaseHandler) PreAction(ctx handlerspi.HandlerContext) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// IsPreActionDone returns true if pre-action done
func (h BaseHandler) IsPreActionDone(ctx handlerspi.HandlerContext) (bool, ctrl.Result, error) {
	return true, ctrl.Result{}, nil
}

// ActionUpdateStatus does the lifecycle Action status update
func (h *BaseHandler) ActionUpdateStatus(context handlerspi.HandlerContext) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// DoAction installs the component using Helm
func (h BaseHandler) DoAction(ctx handlerspi.HandlerContext) (ctrl.Result, error) {
	// Create ModuleLifecycle
	mlc := moduleapi.ModuleLifecycle{
		ObjectMeta: metav1.ObjectMeta{
			Name:      h.MlcName,
			Namespace: h.ModuleCR.Namespace,
		},
	}
	_, err := controllerutil.CreateOrUpdate(context.TODO(), ctx.Client, &mlc, func() error {
		err := h.mutateMLC(&mlc)
		if err != nil {
			return err
		}
		return controllerutil.SetControllerReference(h.ModuleCR, &mlc, h.Config.Scheme)
	})

	return ctrl.Result{}, err
}

func (h BaseHandler) mutateMLC(mlc *moduleapi.ModuleLifecycle) error {
	mlc.Spec.LifecycleClassName = moduleapi.LifecycleClassType(h.ModuleCR.Spec.ModuleName)
	mlc.Spec.Action = h.Action
	mlc.Spec.Version = h.ModuleCR.Spec.Version
	mlc.Spec.Installer.HelmRelease = h.Config.HelmInfo.HelmRelease
	mlc.Spec.Installer.HelmRelease.Overrides = h.ModuleCR.Spec.Overrides
	return nil
}

// IsActionDone returns true if the module action is done
func (h BaseHandler) IsActionDone(ctx handlerspi.HandlerContext) (bool, ctrl.Result, error) {
	if ctx.DryRun {
		return true, ctrl.Result{}, nil
	}

	mlc, err := h.GetModuleLifecycle(ctx)
	if err != nil {
		return false, util.NewRequeueWithShortDelay(), nil
	}
	if mlc.Status.State == moduleapi.StateCompleted || mlc.Status.State == moduleapi.StateNotNeeded {
		return true, ctrl.Result{}, nil
	}
	ctx.Log.Progressf("Waiting for ModuleLifecycle %s to be completed", h.MlcName)
	return false, ctrl.Result{}, nil
}

// PostAction does post-action
func (h BaseHandler) PostAction(ctx handlerspi.HandlerContext) (ctrl.Result, error) {
	if ctx.DryRun {
		return ctrl.Result{}, nil
	}

	if err := h.DeleteModuleLifecycle(ctx); err != nil {
		return util.NewRequeueWithShortDelay(), nil
	}
	return ctrl.Result{}, nil
}

// UpdateDoneStatus does the lifecycle status update with the action is done
func (h BaseHandler) UpdateDoneStatus(ctx handlerspi.HandlerContext, cond moduleapi.LifecycleCondition, state moduleapi.ModuleStateType, version string) (ctrl.Result, error) {
	AppendCondition(h.ModuleCR, string(cond), cond)
	h.ModuleCR.Status.State = state
	h.ModuleCR.Status.ObservedGeneration = h.ModuleCR.Generation
	if len(version) > 0 {
		h.ModuleCR.Status.Version = version
	}
	if err := ctx.Client.Status().Update(context.TODO(), h.ModuleCR); err != nil {
		return util.NewRequeueWithShortDelay(), nil
	}
	return ctrl.Result{}, nil
}

// UpdateStatus does the lifecycle pre-Action status update
func (h BaseHandler) UpdateStatus(ctx handlerspi.HandlerContext, cond moduleapi.LifecycleCondition, state moduleapi.ModuleStateType) (ctrl.Result, error) {
	AppendCondition(h.ModuleCR, string(cond), cond)
	h.ModuleCR.Status.State = state
	if err := ctx.Client.Status().Update(context.TODO(), h.ModuleCR); err != nil {
		return util.NewRequeueWithShortDelay(), nil
	}
	return ctrl.Result{}, nil
}

// PostActionUpdateStatue does installation post-action status update
func (h BaseHandler) PostActionUpdateStatus(ctx handlerspi.HandlerContext) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

// IsPostActionDone returns true if post-action done
func (h BaseHandler) IsPostActionDone(ctx handlerspi.HandlerContext) (bool, ctrl.Result, error) {
	return true, ctrl.Result{}, nil
}

// CompletedActionUpdateStatus does the lifecycle completed Action status update
func (h *BaseHandler) CompletedActionUpdateStatus(context handlerspi.HandlerContext) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}
