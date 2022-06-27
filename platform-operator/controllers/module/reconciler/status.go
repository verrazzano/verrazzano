// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconciler

import (
	"context"
	"fmt"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	modulesv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/modules/v1alpha1"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"time"
)

//UpdateStatus configures the Module's status based on the passed in state and then updates the Module on the cluster
func (r *Reconciler) UpdateStatus(ctx spi.ComponentContext, condition modulesv1alpha1.ModuleCondition) error {
	module := ctx.Module()
	state := modulesv1alpha1.State(condition)
	// Update the Module's State
	module.SetState(state)
	// Append a new condition, if applicable
	appendCondition(module, string(state), condition)

	// update the components status in the VZ CR
	if err := r.updateComponentStatus(ctx, string(state), convertModuleConditiontoCondition(condition)); err != nil {
		return err
	}

	// Update the module status
	return r.doStatusUpdate(ctx)
}

func NeedsReconcile(ctx spi.ComponentContext) bool {
	return ctx.Module().Status.ObservedGeneration != ctx.Module().Generation
}

func NewCondition(message string, condition modulesv1alpha1.ModuleCondition) modulesv1alpha1.Condition {
	t := time.Now().UTC()
	return modulesv1alpha1.Condition{
		Type:    condition,
		Message: message,
		Status:  corev1.ConditionTrue,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}
}

func (r *Reconciler) doStatusUpdate(ctx spi.ComponentContext) error {
	module := ctx.Module()
	err := r.StatusWriter.Update(context.TODO(), module)
	if err == nil {
		return err
	}
	if k8serrors.IsConflict(err) {
		ctx.Log().Debugf("Update conflict for Module %s: %v", module.Name, err)
	} else {
		ctx.Log().Errorf("Failed to update Module %s :v", module.Name, err)
	}
	// Return error so that reconcile gets called again
	return err
}

func appendCondition(module *modulesv1alpha1.Module, message string, condition modulesv1alpha1.ModuleCondition) {
	conditions := module.Status.Conditions
	newCondition := NewCondition(message, condition)
	var lastCondition *modulesv1alpha1.Condition
	if len(conditions) > 0 {
		lastCondition = &conditions[len(conditions)-1]
	}

	// Only update the conditions if there is a notable change between the last update
	if needsConditionUpdate(lastCondition, &newCondition) {
		// Delete oldest condition if at tracking limit
		if len(conditions) > modulesv1alpha1.ConditionArrayLimit {
			conditions = conditions[1:]
		}
		module.Status.Conditions = append(conditions, newCondition)
	}
}

//needsConditionUpdate checks if the condition needs an update
func needsConditionUpdate(last, new *modulesv1alpha1.Condition) bool {
	if last == nil {
		return true
	}
	return last.Type != new.Type && last.Message != new.Message
}

// updateComponentStatus updates the component status in the VZ CR
func (r *Reconciler) updateComponentStatus(compContext spi.ComponentContext, message string, conditionType installv1alpha1.ConditionType) error {
	t := time.Now().UTC()
	condition := installv1alpha1.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}

	componentName := compContext.GetComponent()
	cr := compContext.ActualCR()
	log := compContext.Log()

	if cr.Status.Components == nil {
		cr.Status.Components = make(map[string]*installv1alpha1.ComponentStatusDetails)
	}
	componentStatus := cr.Status.Components[componentName]
	if componentStatus == nil {
		componentStatus = &installv1alpha1.ComponentStatusDetails{
			Name: componentName,
		}
		cr.Status.Components[componentName] = componentStatus
	}

	if conditionType == installv1alpha1.CondInstallComplete {
		if componentStatus.ReconcilingGeneration > 0 {
			componentStatus.LastReconciledGeneration = componentStatus.ReconcilingGeneration
			componentStatus.ReconcilingGeneration = 0
		} else {
			componentStatus.LastReconciledGeneration = cr.Generation
		}
	} else {
		if componentStatus.ReconcilingGeneration == 0 {
			componentStatus.ReconcilingGeneration = cr.Generation
		}
	}
	componentStatus.Conditions = appendConditionIfNecessary(log, componentStatus, condition)

	// Set the state of resource
	componentStatus.State = checkCondtitionType(conditionType)

	// Update the status
	return r.updateVerrazzanoStatus(log, cr)
}

func checkCondtitionType(currentCondition installv1alpha1.ConditionType) installv1alpha1.CompStateType {
	switch currentCondition {
	case installv1alpha1.CondPreInstall:
		return installv1alpha1.CompStatePreInstalling
	case installv1alpha1.CondInstallStarted:
		return installv1alpha1.CompStateInstalling
	case installv1alpha1.CondUninstallStarted:
		return installv1alpha1.CompStateUninstalling
	case installv1alpha1.CondUpgradeStarted:
		return installv1alpha1.CompStateUpgrading
	case installv1alpha1.CondUpgradePaused:
		return installv1alpha1.CompStateUpgrading
	case installv1alpha1.CondUninstallComplete:
		return installv1alpha1.CompStateReady
	case installv1alpha1.CondInstallFailed, installv1alpha1.CondUpgradeFailed, installv1alpha1.CondUninstallFailed:
		return installv1alpha1.CompStateFailed
	}
	// Return ready for installv1alpha1.CondInstallComplete, installv1alpha1.CondUpgradeComplete
	return installv1alpha1.CompStateReady
}

func appendConditionIfNecessary(log vzlog.VerrazzanoLogger, compStatus *installv1alpha1.ComponentStatusDetails, newCondition installv1alpha1.Condition) []installv1alpha1.Condition {
	for _, existingCondition := range compStatus.Conditions {
		if existingCondition.Type == newCondition.Type {
			return compStatus.Conditions
		}
	}
	log.Debugf("Adding %s resource newCondition: %v", compStatus.Name, newCondition.Type)
	return append(compStatus.Conditions, newCondition)
}

func (r *Reconciler) updateVerrazzanoStatus(log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	err := r.StatusWriter.Update(context.TODO(), vz)
	if err == nil {
		return nil
	}
	if ctrlerrors.IsUpdateConflict(err) {
		log.Debugf("Requeuing to get a fresh copy of the Verrazzano resource since the current one is outdated.")
	} else {
		log.Errorf("Failed to update Verrazzano resource :v", err)
	}
	// Return error so that reconcile gets called again
	return err
}

// convertModuleConditiontoCondition converts ModuleCondition types to ConditionType types
// this will then get converted to CompStateType in updateComponentStatus by the CheckCondtitionType function
// return nil if ModuleCondition is unknown
func convertModuleConditiontoCondition(moduleCondion modulesv1alpha1.ModuleCondition) installv1alpha1.ConditionType {
	switch moduleCondion {
	// install ConditionTypes
	case modulesv1alpha1.CondPreInstall:
		return installv1alpha1.CondPreInstall
	case modulesv1alpha1.CondInstallStarted:
		return installv1alpha1.CondInstallStarted
	case modulesv1alpha1.CondInstallComplete:
		return installv1alpha1.CondInstallComplete
	// upgrade ConditionTypes
	case modulesv1alpha1.CondPreUpgrade, modulesv1alpha1.CondUpgradeStarted:
		return installv1alpha1.CondUpgradeStarted
	case modulesv1alpha1.CondUpgradeComplete:
		return installv1alpha1.CondUpgradeComplete
	}
	// otherwise return UninstallStarted
	return installv1alpha1.CondUninstallStarted
}
