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

// checkInstallComplete checks to see if the install is complete
func (r *Reconciler) checkInstallComplete(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano) (bool, error) {
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

func checkCondtitionType(currentCondition vzv1alpha1.ConditionType) vzv1alpha1.CompStateType {
	switch currentCondition {
	case vzv1alpha1.CondPreInstall:
		return vzv1alpha1.CompStatePreInstalling
	case vzv1alpha1.CondInstallStarted:
		return vzv1alpha1.CompStateInstalling
	case vzv1alpha1.CondUninstallStarted:
		return vzv1alpha1.CompStateUninstalling
	case vzv1alpha1.CondUpgradeStarted:
		return vzv1alpha1.CompStateUpgrading
	case vzv1alpha1.CondUpgradePaused:
		return vzv1alpha1.CompStateUpgrading
	case vzv1alpha1.CondUninstallComplete:
		return vzv1alpha1.CompStateUninstalled
	case vzv1alpha1.CondInstallFailed, vzv1alpha1.CondUpgradeFailed, vzv1alpha1.CondUninstallFailed:
		return vzv1alpha1.CompStateFailed
	}
	// Return ready for vzv1alpha1.CondInstallComplete, vzv1alpha1.CondUpgradeComplete
	return vzv1alpha1.CompStateReady
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

// setInstallStartedCondition
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

// The new Verrazzano controller actualCR.ates Modules for components.  Make sure those modules are ready.
func (r *Reconciler) modulesReady(compCtx spi.ComponentContext) (bool, error) {
	for _, comp := range registry.GetComponents() {
		if !comp.IsEnabled(compCtx.EffectiveCR()) {
			continue
		}

		module := moduleapi.Module{}
		nsn := types.NamespacedName{Namespace: vzconst.VerrazzanoInstallNamespace, Name: comp.Name()}
		err := r.Client.Get(context.TODO(), nsn, &module, &client.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			compCtx.Log().ErrorfThrottled("Failed to get Module %s, retrying: %v", comp.Name(), err)
			return false, err
		}

		cond := modulestatus.GetReadyCondition(&module)
		if cond == nil {
			return false, nil
		}
		if module.Status.LastSuccessfulGeneration != module.Generation {
			return false, nil
		}
		if module.Status.LastSuccessfulVersion != module.Spec.Version {
			return false, nil
		}
	}
	return true, nil
}
