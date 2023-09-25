// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controller

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulestatus "github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/vzinstance"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

// initializeComponentStatus Initialize the component status field with the known set that indicate they support the
// operator-based installation.  This is so that we know ahead of time exactly how many components we expect to install
// via the operator, and when we're done installing.
func (r Reconciler) initializeComponentStatus(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) result.Result {
	if actualCR.Status.Components == nil {
		actualCR.Status.Components = make(map[string]*vzv1alpha1.ComponentStatusDetails)
	}

	newContext, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	statusUpdated := false
	for _, comp := range registry.GetComponents() {
		if status, ok := actualCR.Status.Components[comp.Name()]; ok {
			if status.LastReconciledGeneration == 0 {
				actualCR.Status.Components[comp.Name()] = status
				status.LastReconciledGeneration = actualCR.Generation
			}
			// Skip components that have already been processed
			continue
		}
		if comp.IsOperatorInstallSupported() {
			// If the component is installed then mark it as ready
			compContext := newContext.Init(comp.Name()).Operation(vzconst.InitializeOperation)
			lastReconciled := int64(0)
			state := vzv1alpha1.CompStateDisabled
			installed, err := comp.IsInstalled(compContext)
			if err != nil {
				log.Errorf("Failed to determine if component %s is installed: %v", comp.Name(), err)
				return result.NewResultShortRequeueDelayWithError(err)
			}
			if installed {
				state = vzv1alpha1.CompStateReady
				lastReconciled = compContext.ActualCR().Generation
			}
			actualCR.Status.Components[comp.Name()] = &vzv1alpha1.ComponentStatusDetails{
				Name:                     comp.Name(),
				State:                    state,
				LastReconciledGeneration: lastReconciled,
			}
			statusUpdated = true
		}
	}
	// Update the status
	if statusUpdated {
		// Use Status update directly so any conflicting updates get rejected.
		// This is needed for integration with the new Verrazzano controller used for modules
		// Basically, that controller was seeing the component status updates and actualCR.ating
		// Module actualCR., which in turn updated the component status conditions.  However, this code
		// was subsequently re-initializing the component status because it didn't know there was an update conflict
		// and that it needed to requeue, so it was using a stale copy of the VZ actualCR.
		r.Client.Status().Update(context.TODO(), actualCR)
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

func (r Reconciler) updateStatusIfNeeded(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, opResult controllerutil.OperationResult) result.Result {
	if opResult == controllerutil.OperationResultNone {
		return result.NewResult()
	}
	if !r.isUpgrading(actualCR) {
		if err := r.updateStatusInstalling(log, actualCR); err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}
	return result.NewResult()
}

// isUpgrading returns true if spec indicates upgrade.
func (r Reconciler) isUpgrading(actualCR *vzv1alpha1.Verrazzano) bool {
	return actualCR.Spec.Version != "" && actualCR.Spec.Version != actualCR.Status.Version
}

// updateStatusInstalling adds installing condition and sets the state
func (r Reconciler) updateStatusInstalling(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) error {
	var conditionsToRemove = map[vzv1alpha1.ConditionType]bool{
		vzv1alpha1.CondInstallStarted:  true,
		vzv1alpha1.CondInstallComplete: true,
		vzv1alpha1.CondInstallFailed:   true,
	}

	var conditionToSearch = map[vzv1alpha1.ConditionType]bool{
		vzv1alpha1.CondInstallComplete: true,
		vzv1alpha1.CondInstallFailed:   true,
	}

	// Remove old install conditions if a previous install occurred
	conditions := actualCR.Status.Conditions
	if doesAnyConditionExist(actualCR, conditionToSearch) {
		conditions = removePreviousConditions(actualCR, conditionsToRemove)
	}

	// Return if condition already added
	if findConditionByType(conditions, vzv1alpha1.CondInstallStarted) {
		return nil
	}

	cond := newCondition("Verrazzano install is in progress", vzv1alpha1.CondInstallStarted)
	conditions = append(conditions, cond)

	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: actualCR,
		Conditions: conditions,
		State:      vzv1alpha1.VzStateReconciling,
	})
	return nil
}

// updateStatusUninstalling adds uninstalling condition and sets the state
func (r Reconciler) updateStatusUninstalling(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) error {
	var conditionToSearch = map[vzv1alpha1.ConditionType]bool{
		vzv1alpha1.CondUninstallStarted:  true,
		vzv1alpha1.CondUninstallComplete: true,
	}

	// For uninstall return if uninstall or complete already writtne
	if doesAnyConditionExist(actualCR, conditionToSearch) {
		return nil
	}

	cond := newCondition("Verrazzano uninstall is in progress", vzv1alpha1.CondUninstallStarted)
	conditions := append(actualCR.Status.Conditions, cond)

	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: actualCR,
		Conditions: conditions,
		State:      vzv1alpha1.VzStateUninstalling,
	})
	return nil
}

// updateStatusUpgrading adds upgrading condition and sets the state
func (r Reconciler) updateStatusUpgrading(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) error {
	var conditionsToRemove = map[vzv1alpha1.ConditionType]bool{
		vzv1alpha1.CondUpgradeStarted:  true,
		vzv1alpha1.CondUpgradeComplete: true,
		vzv1alpha1.CondUpgradePaused:   true,
		vzv1alpha1.CondUpgradeFailed:   true,
	}

	var conditionsToSearch = map[vzv1alpha1.ConditionType]bool{
		vzv1alpha1.CondUpgradeComplete: true,
		vzv1alpha1.CondUpgradePaused:   true,
		vzv1alpha1.CondUpgradeFailed:   true,
	}

	// Remove old upgrade conditions if a previous upgrade occurred
	conditions := actualCR.Status.Conditions
	if doesAnyConditionExist(actualCR, conditionsToSearch) {
		conditions = removePreviousConditions(actualCR, conditionsToRemove)
	}

	// Return if condition already added
	if findConditionByType(conditions, vzv1alpha1.CondUpgradeStarted) {
		return nil
	}

	cond := newCondition(fmt.Sprintf("Verrazzano upgrade to version %s in progress", actualCR.Spec.Version), vzv1alpha1.CondUpgradeStarted)
	conditions = append(conditions, cond)

	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: actualCR,
		Conditions: conditions,
		State:      vzv1alpha1.VzStateUpgrading,
	})
	return nil
}

func (r Reconciler) updateStatusInstallUpgradeComplete(actualCR *vzv1alpha1.Verrazzano) error {
	// Set complete status
	if r.isUpgrading(actualCR) {
		return r.updateStatusUpgradeComplete(actualCR)
	}
	return r.updateStatusInstallComplete(actualCR)
}

// updateStatusInstallComplete updates the status condition and state for install complete
func (r Reconciler) updateStatusInstallComplete(actualCR *vzv1alpha1.Verrazzano) error {
	return r.updateStatusComplete(actualCR, "Verrazzano install complete", vzv1alpha1.CondInstallComplete)
}

// updateStatusUninstallComplete updates the status condition and state for uninstall complete
func (r Reconciler) updateStatusUninstallComplete(actualCR *vzv1alpha1.Verrazzano) error {
	return r.updateStatusComplete(actualCR, "Verrazzano uninstall complete", vzv1alpha1.CondUninstallComplete)
}

// updateStatusInstallComplete updates the status condition and state for upgrade complete
func (r Reconciler) updateStatusUpgradeComplete(actualCR *vzv1alpha1.Verrazzano) error {
	return r.updateStatusComplete(actualCR, "Verrazzano upgrade complete", vzv1alpha1.CondUpgradeComplete)
}

// updateStatusInstallComplete updates the status condition and state for install complete
func (r Reconciler) updateStatusComplete(actualCR *vzv1alpha1.Verrazzano, msg string, conditionType vzv1alpha1.ConditionType) error {
	spiCtx, err := componentspi.NewContext(vzlog.DefaultLogger(), r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		return err
	}

	if findConditionByType(actualCR.Status.Conditions, conditionType) {
		return nil
	}
	cond := newCondition(msg, conditionType)
	conditions := append(actualCR.Status.Conditions, cond)

	version := actualCR.Spec.Version
	if len(version) == 0 {
		var err error
		version, err = getBomVersion()
		if err != nil {
			return err
		}
	}

	// Make sure all components have the correct reconciled generation
	compponents := r.forceSyncComponentReconciledGeneration(actualCR)

	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano:   actualCR,
		Conditions:   conditions,
		Version:      &version,
		Components:   compponents,
		InstanceInfo: vzinstance.GetInstanceInfo(spiCtx),
		State:        vzv1alpha1.VzStateReady,
	})
	return nil
}

// newCondition creates a new condition
func newCondition(message string, conditionType vzv1alpha1.ConditionType) vzv1alpha1.Condition {
	t := time.Now().UTC()
	return vzv1alpha1.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}
}

// removePreviousConditions removes previous conditions for the current work (installStarted/installComplete)
func removePreviousConditions(actualCR *vzv1alpha1.Verrazzano, removeConditions map[vzv1alpha1.ConditionType]bool) (newConditions []vzv1alpha1.Condition) {
	for i, cond := range actualCR.Status.Conditions {
		if _, ok := removeConditions[cond.Type]; !ok {
			newConditions = append(newConditions, actualCR.Status.Conditions[i])
		}
	}
	return
}

func doesAnyConditionExist(actualCR *vzv1alpha1.Verrazzano, targets map[vzv1alpha1.ConditionType]bool) bool {
	for _, cond := range actualCR.Status.Conditions {
		if _, ok := targets[cond.Type]; ok {
			return true
		}
	}
	return false
}

func findConditionByType(conditions []vzv1alpha1.Condition, condType vzv1alpha1.ConditionType) bool {
	for _, cond := range conditions {
		if cond.Type == condType {
			return true
		}
	}
	return false
}

// areModulesDoneReconciling returns true if modules are ready, this includes deleted modules being removed.
func (r Reconciler) areModulesDoneReconciling(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) bool {
	for _, comp := range registry.GetComponents() {
		compCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, false)
		if err != nil {
			compCtx.Log().Errorf("Failed to create component context: %v", err)
			return false
		}

		module := moduleapi.Module{}
		nsn := types.NamespacedName{Namespace: vzconst.VerrazzanoInstallNamespace, Name: comp.Name()}
		if err := r.Client.Get(context.TODO(), nsn, &module, &client.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				// if module is disabled and not found, that means module is done uninstalling (i.e. ready)
				if !comp.IsEnabled(compCtx.EffectiveCR()) {
					continue
				}
				// component enabled but not found, return false since it must exist
				return false
			}
			compCtx.Log().ErrorfThrottled("Failed to get Module %s, retrying: %v", comp.Name(), err)
			return false
		}

		cond := modulestatus.GetReadyCondition(&module)
		if cond == nil || cond.Status != corev1.ConditionTrue {
			return false
		}
		if module.Status.LastSuccessfulGeneration != module.Generation {
			return false
		}
		if module.Status.LastSuccessfulVersion != module.Spec.Version {
			return false
		}
	}
	return true
}

// forceSyncComponentReconciledGeneration Force all Ready components' lastReconciledGeneration to match the VZ CR generation;
// this is applied at the end of a successful VZ CR reconcile.
func (r Reconciler) forceSyncComponentReconciledGeneration(actualCR *vzv1alpha1.Verrazzano) map[string]*vzv1alpha1.ComponentStatusDetails {
	componentsToUpdate := map[string]*vzv1alpha1.ComponentStatusDetails{}
	for compName, componentStatus := range actualCR.Status.Components {
		if componentStatus.State == vzv1alpha1.CompStateReady {
			componentStatus.LastReconciledGeneration = actualCR.Generation
			componentsToUpdate[compName] = componentStatus
		}
	}
	return componentsToUpdate
}

func getBomVersion() (string, error) {
	// Set the version in the status.  This will be updated when the starting install condition is updated.
	bomSemVer, err := validators.GetCurrentBomVersion()
	if err != nil {
		return "", err
	}
	return bomSemVer.ToString(), err
}
