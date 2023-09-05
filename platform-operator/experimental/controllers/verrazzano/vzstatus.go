// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulestatus "github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/vzlog"
	vpovzlog "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/healthcheck"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/vzinstance"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var unitesting bool

// initializeComponentStatus Initialize the component status field with the known set that indicate they support the
// operator-based installation.  This is so that we know ahead of time exactly how many components we expect to install
// via the operator, and when we're done installing.
func (r *Reconciler) initializeComponentStatus(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) result.Result {
	if actualCR.Status.Components == nil {
		actualCR.Status.Components = make(map[string]*vzv1alpha1.ComponentStatusDetails)
	}

	newContext, err := spi.NewContext(vpovzlog.DefaultLogger(), r.Client, actualCR, nil, r.DryRun)
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

// checkReconcileComplete checks to see if the reconcile is complete
func (r *Reconciler) checkReconcileComplete(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) (bool, error) {
	ready, err := r.checkComponentReadyState(log, actualCR)
	if err != nil {
		return false, err
	}
	if !ready {
		return false, nil
	}

	// Set install complete IFF all subcomponent status' are "CompStateReady"
	message := "Verrazzano install completed successfully"
	// Status update must be performed on the actual actualCR.read from K8S
	return true, r.updateStatus(log, actualCR, message, vzv1alpha1.CondInstallComplete, nil)
}

// checkUpgradeComplete checks to see if the upgrade is complete
func (r *Reconciler) checkUpgradeComplete(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) (bool, error) {
	if actualCR == nil {
		return false, nil
	}
	if actualCR.Status.State != vzv1alpha1.VzStateUpgrading {
		return true, nil
	}
	ready, err := r.checkComponentReadyState(log, actualCR)
	if err != nil {
		return false, err
	}
	if !ready {
		return false, nil
	}
	// Set upgrade complete IFF all subcomponent status' are "CompStateReady"
	message := "Verrazzano upgrade completed successfully"
	// Status and State update must be performed on the actual actualCR.read from K8S
	return true, r.updateVzStatusAndState(log, actualCR, message, vzv1alpha1.CondUpgradeComplete, vzv1alpha1.VzStateReady)
}

// updateStatus updates the status in the Verrazzano CR
func (r *Reconciler) updateStatus(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, message string, conditionType vzv1alpha1.ConditionType, version *string) error {
	t := time.Now().UTC()
	condition := vzv1alpha1.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}
	conditions := appendConditionIfNecessary(log, actualCR.Name, actualCR.Status.Conditions, condition)

	// Set the state of resource
	state := conditionToVzState(conditionType)
	log.Debugf("Setting Verrazzano resource condition and state: %v/%v", condition.Type, state)

	event := &vzstatus.UpdateEvent{
		Verrazzano: actualCR,
		Version:    version,
		State:      state,
		Conditions: conditions,
	}

	if conditionType == vzv1alpha1.CondInstallComplete {
		spiCtx, err := spi.NewContext(vpovzlog.DefaultLogger(), r.Client, actualCR, nil, r.DryRun)
		if err != nil {
			spiCtx.Log().Errorf("Failed to init component context: %v", err)
			return err
		}
		event.InstanceInfo = vzinstance.GetInstanceInfo(spiCtx)
	}

	// Update the status
	r.StatusUpdater.Update(event)
	return nil
}

// updateVzState updates the status state in the Verrazzano CR
func (r *Reconciler) updateVzState(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, state vzv1alpha1.VzStateType) {
	log.Debugf("Setting Verrazzano state: %v", state)
	// Update the status
	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: actualCR,
		State:      state,
	})
}

// updateVzState updates the status state in the Verrazzano CR
func (r *Reconciler) updateVzStatusAndState(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, message string, conditionType vzv1alpha1.ConditionType, state vzv1alpha1.VzStateType) error {
	t := time.Now().UTC()
	condition := vzv1alpha1.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}
	conditions := appendConditionIfNecessary(log, actualCR.Name, actualCR.Status.Conditions, condition)
	log.Debugf("Setting Verrazzano state: %v", state)

	spiCtx, err := spi.NewContext(vpovzlog.DefaultLogger(), r.Client, actualCR, nil, r.DryRun)
	if err != nil {
		spiCtx.Log().Errorf("Failed to actualCR.ate component context: %v", err)
		return err
	}

	// Update the status
	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano:   actualCR,
		State:        state,
		Conditions:   conditions,
		InstanceInfo: vzinstance.GetInstanceInfo(spiCtx),
	})
	return nil
}

func appendConditionIfNecessary(log vzlog.VerrazzanoLogger, resourceName string, conditions []vzv1alpha1.Condition, newCondition vzv1alpha1.Condition) []vzv1alpha1.Condition {
	var newConditionsList []vzv1alpha1.Condition
	for i, existingCondition := range conditions {
		if existingCondition.Type != newCondition.Type {
			// Skip any existing conditions of the same type as the new condition. We will append
			// the new condition at the end. If there are duplicate conditions from a legacy
			// VZ resource, they will all be skipped.
			newConditionsList = append(newConditionsList, conditions[i])
		}
	}
	log.Debugf("Adding/modifying %s resource newCondition: %v", resourceName, newCondition.Type)
	// Always put the new condition at the end of the list since the kubectl status display and
	// some upgrade stuff depends on the most recent condition being the last one
	return append(newConditionsList, newCondition)
}

// Convert a condition to a VZ State
func conditionToVzState(currentCondition vzv1alpha1.ConditionType) vzv1alpha1.VzStateType {
	switch currentCondition {
	case vzv1alpha1.CondInstallStarted:
		return vzv1alpha1.VzStateReconciling
	case vzv1alpha1.CondUninstallStarted:
		return vzv1alpha1.VzStateUninstalling
	case vzv1alpha1.CondUpgradeStarted:
		return vzv1alpha1.VzStateUpgrading
	case vzv1alpha1.CondUpgradePaused:
		return vzv1alpha1.VzStatePaused
	case vzv1alpha1.CondUninstallComplete:
		return vzv1alpha1.VzStateReady
	case vzv1alpha1.CondInstallFailed, vzv1alpha1.CondUpgradeFailed, vzv1alpha1.CondUninstallFailed:
		return vzv1alpha1.VzStateFailed
	}
	// Return ready for vzv1alpha1.CondInstallComplete, vzv1alpha1.CondUpgradeComplete
	return vzv1alpha1.VzStateReady
}

// updateStatusToUpgradeStarted updates the condition and state for upgrade started.
func (r *Reconciler) updateStatusToUpgradeStarted(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) error {
	return r.updateStatus(log, actualCR, fmt.Sprintf("Verrazzano upgrade to version %s in progress", actualCR.Spec.Version),
		vzv1alpha1.CondUpgradeStarted, nil)
}

// updateStatusToInstallStarted updates the condition and state for install started.
func (r *Reconciler) updateStatusToInstallStarted(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) error {
	return r.updateStatus(log, actualCR, fmt.Sprintf("Verrazzano install in progress"),
		vzv1alpha1.CondInstallStarted, nil)
}

// isUpgrading returns true if spec indicates upgrade.
func (r *Reconciler) isUpgrading(actualCR *vzv1alpha1.Verrazzano) bool {
	return actualCR.Spec.Version != "" && actualCR.Spec.Version != actualCR.Status.Version
}

// updateStatusToDone updates the state to Ready and InstallComplete or UpgradeComplete
func (r *Reconciler) updateStatusToDone(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) error {
	if r.isUpgrading(actualCR) {
		return r.updateStatus(log, actualCR, fmt.Sprintf("Verrazzano install in progress"),
			vzv1alpha1.CondUpgradeComplete, &actualCR.Spec.Version)
	}
	return r.updateStatus(log, actualCR, fmt.Sprintf("Verrazzano install in progress"),
		vzv1alpha1.CondInstallComplete, &actualCR.Spec.Version)
}

// updateUpgradingConditionAndState adds upgrading condition and sets the state
func (r *Reconciler) updateUpgradingConditionAndState(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) error {
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

	// Add new upgrading conditions if one doesn't exist
	if findConditionByType(actualCR, vzv1alpha1.CondUpgradeStarted) {
		cond := newCondition(fmt.Sprintf("Verrazzano upgrade to version %s in progress", actualCR.Spec.Version), vzv1alpha1.CondUpgradeStarted)
		conditions = append(conditions, cond)
	}

	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: actualCR,
		Conditions: conditions,
		State:      vzv1alpha1.VzStateUpgrading,
	})
	return nil
}

// updateInstallingConditionAndState adds upgrading condition and sets the state
func (r *Reconciler) updateInstallingConditionAndState(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) error {
	var conditionsToRemove = map[vzv1alpha1.ConditionType]bool{
		vzv1alpha1.CondInstallStarted:  true,
		vzv1alpha1.CondInstallComplete: true,
		vzv1alpha1.CondInstallFailed:   true,
	}

	var conditionToSearch = map[vzv1alpha1.ConditionType]bool{
		vzv1alpha1.CondInstallComplete: true,
		vzv1alpha1.CondInstallFailed:   true,
	}

	// Remove old upgrade conditions if a previous upgrade occurred
	conditions := actualCR.Status.Conditions
	if doesAnyConditionExist(actualCR, conditionToSearch) {
		conditions = removePreviousConditions(actualCR, conditionsToRemove)
	}

	// Add new upgrading conditions if one doesn't exist
	if findConditionByType(actualCR, vzv1alpha1.CondInstallStarted) {
		cond := newCondition(fmt.Sprintf("Verrazzano install is in progress", actualCR.Spec.Version), vzv1alpha1.CondInstallStarted)
		conditions = append(conditions, cond)
	}

	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: actualCR,
		Conditions: conditions,
		State:      vzv1alpha1.VzStateUpgrading,
	})
	return nil
}

// updateWorkingConditionAndState
func updateWorkingConditionAndState(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) error {
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

func findConditionByType(actualCR *vzv1alpha1.Verrazzano, condType vzv1alpha1.ConditionType) bool {
	for _, cond := range actualCR.Status.Conditions {
		if cond.Type == condType {
			return true
		}
	}
	return false
}

// updateDoneConditionAndState
func (r *Reconciler) updateDoneConditionAndState(actualCR *vzv1alpha1.Verrazzano) error {
	if actualCR.Status.State != vzv1alpha1.VzStateReconciling {
		// state is already determinined
		return nil
	}

	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: actualCR,
		State:      vzv1alpha1.VzStateReconciling,
	})
	return nil
}

// deriveStateAndConditionFromModules determines what the VZ state and condition should be based on modules
func (r Reconciler) deriveStateAndConditionFromModules() {

}

// setUpgradingState
func (r *Reconciler) setUpgradingState(log vzlog.VerrazzanoLogger, vz *vzv1alpha1.Verrazzano) error {
	// Set the version in the status.  This will be updated when the starting install condition is updated.
	bomSemVer, err := validators.GetCurrentBomVersion()
	if err != nil {
		return err
	}

	version := bomSemVer.ToString()
	return r.updateStatus(log, vz, "Verrazzano upgrade in progress", vzv1alpha1.CondUpgradeStarted, &version)
}

// setInstallingState
func (r *Reconciler) setInstallingState(log vzlog.VerrazzanoLogger, vz *vzv1alpha1.Verrazzano) error {
	// Set the version in the status.  This will be updated when the starting install condition is updated.
	bomSemVer, err := validators.GetCurrentBomVersion()
	if err != nil {
		return err
	}

	version := bomSemVer.ToString()
	return r.updateStatus(log, vz, "Verrazzano install in progress", vzv1alpha1.CondInstallStarted, &version)
}

// checkComponentReadyState returns true if all component-level status' are "CompStateReady" for enabled components
func (r *Reconciler) checkComponentReadyState(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) (bool, error) {
	// Return false if any enabled component is not ready
	for _, comp := range registry.GetComponents() {
		spiCtx, err := spi.NewContext(vpovzlog.DefaultLogger(), r.Client, actualCR, nil, r.DryRun)
		if err != nil {
			log.Errorf("Failed to actualCR.ate component context: %v", err)
			return false, err
		}
		if comp.IsEnabled(spiCtx.EffectiveCR()) && actualCR.Status.Components[comp.Name()].State != vzv1alpha1.CompStateReady {
			spiCtx.Log().Progressf("Waiting for component %s to be ready", comp.Name())
			return false, nil
		}
	}
	return true, nil
}

// setUninstallCondition sets the Verrazzano resource condition in status for uninstall
func (r *Reconciler) setUninstallCondition(log vzlog.VerrazzanoLogger, vz *vzv1alpha1.Verrazzano, newCondition vzv1alpha1.ConditionType, msg string) (err error) {
	// Add the uninstall started condition if not already added
	for _, condition := range vz.Status.Conditions {
		if condition.Type == newCondition {
			return nil
		}
	}
	return r.updateStatus(log, vz, msg, newCondition, nil)
}

// areModulesDoneReconciling returns true if modules are ready, this includes deleted modules being removed.
func (r *Reconciler) areModulesDoneReconciling(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) bool {
	for _, comp := range registry.GetComponents() {
		compCtx, err := spi.NewContext(vpovzlog.DefaultLogger(), r.Client, actualCR, nil, false)
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
func (r *Reconciler) forceSyncComponentReconciledGeneration(actualCR *vzv1alpha1.Verrazzano) error {
	if !config.Get().ModuleIntegration {
		// only do this with modules integration enabled
		return nil
	}
	componentsToUpdate := map[string]*vzv1alpha1.ComponentStatusDetails{}
	for compName, componentStatus := range actualCR.Status.Components {
		if componentStatus.State == vzv1alpha1.CompStateReady {
			componentStatus.LastReconciledGeneration = actualCR.Generation
			componentsToUpdate[compName] = componentStatus
		}
	}
	// Update the status with the new version and component generations
	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Components: componentsToUpdate,
	})
	return nil
}
