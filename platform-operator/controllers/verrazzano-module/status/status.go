// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package status

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/module-operator/internal/handlerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

// readyConditionMessages defines the condition messages for the Ready type condition
var readyConditionMessages = map[moduleapi.ModuleConditionReason]string{
	moduleapi.ReadyReasonInstallStarted:     "Started installing Module `%s` as Helm release `%s/%s`",
	moduleapi.ReadyReasonInstallSucceeded:   "Successfully installed Module `%s` as Helm release `%s/%s`",
	moduleapi.ReadyReasonInstallFailed:      "Failed installing Module `%s` as Helm release `%s%s`: %v",
	moduleapi.ReadyReasonUninstallStarted:   "Started uninstalling Module `%s` as Helm release `%s/%s`",
	moduleapi.ReadyReasonUninstallSucceeded: "Successfully uninstalled Module `%s` as Helm release `%s/%s`",
	moduleapi.ReadyReasonUninstallFailed:    "Failed uninstalling Module `%s` as Helm release `%s/%s`: %v",
	moduleapi.ReadyReasonUpdateStarted:      "Started updating Module `%s` as Helm release `%s/%s`",
	moduleapi.ReadyReasonUpdateSucceeded:    "Successfully updated Module `%s` as Helm release `%s/%s`",
	moduleapi.ReadyReasonUpdateFailed:       "Failed updating Module `%s` as Helm release `%s/%s`: %v",
	moduleapi.ReadyReasonUpgradeStarted:     "Started upgrading Module `%s` as Helm release `%s/%s`",
	moduleapi.ReadyReasonUpgradeSucceeded:   "Successfully upgraded Module `%s` as Helm release `%s/%s`",
	moduleapi.ReadyReasonUpgradeFailed:      "Failed upgrading Module `%s` as Helm release `%s/%s`: %v",
}

// UpdateReadyConditionSucceeded updates the Ready condition when the module has succeeded
func UpdateReadyConditionSucceeded(ctx handlerspi.HandlerContext, module *moduleapi.Module, reason moduleapi.ModuleConditionReason) result.Result {
	module.Status.LastSuccessfulVersion = module.Spec.Version
	module.Status.LastSuccessfulGeneration = module.Generation

	msgTemplate := readyConditionMessages[reason]
	msg := fmt.Sprintf(msgTemplate, module.Name, module.Spec.TargetNamespace, ctx.HelmRelease.Name)
	return updateReadyCondition(ctx, module, reason, corev1.ConditionTrue, msg)
}

// UpdateReadyConditionReconciling updates the Ready condition when the module is reconciling
func UpdateReadyConditionReconciling(ctx handlerspi.HandlerContext, module *moduleapi.Module, reason moduleapi.ModuleConditionReason) result.Result {
	msgTemplate := readyConditionMessages[reason]
	msg := fmt.Sprintf(msgTemplate, module.Name, module.Spec.TargetNamespace, ctx.HelmRelease.Name)

	return updateReadyCondition(ctx, module, reason, corev1.ConditionFalse, msg)
}

// UpdateReadyConditionFailed updates the Ready condition when the module has failed
func UpdateReadyConditionFailed(ctx handlerspi.HandlerContext, module *moduleapi.Module, reason moduleapi.ModuleConditionReason, msgDetail string) result.Result {
	msgTemplate := readyConditionMessages[reason]
	msg := fmt.Sprintf(msgTemplate, module.Name, module.Spec.TargetNamespace, ctx.HelmRelease.Name, msgDetail)

	return updateReadyCondition(ctx, module, reason, corev1.ConditionFalse, msg)
}

// updateReadyCondition updates the Ready condition
func updateReadyCondition(ctx handlerspi.HandlerContext, module *moduleapi.Module, reason moduleapi.ModuleConditionReason, status corev1.ConditionStatus, msg string) result.Result {
	// Always get the latest module from the controller-runtime cache to try and avoid conflict error
	latestModule := &moduleapi.Module{}
	if err := ctx.Client.Get(context.TODO(), types.NamespacedName{Namespace: module.Namespace, Name: module.Name}, latestModule); err != nil {
		return result.NewResultShortRequeueDelay()
	}
	latestModule.Status.LastSuccessfulVersion = module.Status.LastSuccessfulVersion
	latestModule.Status.LastSuccessfulGeneration = module.Status.LastSuccessfulGeneration

	cond := moduleapi.ModuleCondition{
		Type:    moduleapi.ModuleConditionReady,
		Reason:  reason,
		Status:  status,
		Message: msg,
	}
	appendCondition(latestModule, cond)

	if err := ctx.Client.Status().Update(context.TODO(), latestModule); err != nil {
		return result.NewResultShortRequeueDelay()
	}
	return result.NewResult()
}

// appendCondition appends the condition to the list of conditions
func appendCondition(module *moduleapi.Module, cond moduleapi.ModuleCondition) {
	cond.LastTransitionTime = getTransitionTime()

	// Copy conditions that have a different type than the input condition into a new list
	var newConditions []moduleapi.ModuleCondition
	for i, existing := range module.Status.Conditions {
		if existing.Type != cond.Type {
			newConditions = append(newConditions, module.Status.Conditions[i])
		}
	}
	module.Status.Conditions = append(newConditions, cond)
}

// IsInstalled checks if the modules is installed
func IsInstalled(cr *moduleapi.Module) bool {
	cond := GetReadyCondition(cr)
	if cond == nil {
		return false
	}

	// If the reason is not install started or failed, then assume installed.
	switch cond.Reason {
	case moduleapi.ReadyReasonInstallStarted:
		return false
	case moduleapi.ReadyReasonInstallFailed:
		return false
	}

	return true
}

// GetReadyCondition gets the Ready condition type
func GetReadyCondition(cr *moduleapi.Module) *moduleapi.ModuleCondition {
	for i, cond := range cr.Status.Conditions {
		if cond.Type == moduleapi.ModuleConditionReady {
			return &cr.Status.Conditions[i]
		}
	}
	return nil
}

func getTransitionTime() string {
	t := time.Now().UTC()
	return fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second())
}
