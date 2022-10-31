// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzcontext "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/status"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/vzinstance"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

// checkInstallComplete checks to see if the install is complete
func (r *Reconciler) checkInstallComplete(vzctx vzcontext.VerrazzanoContext) (bool, error) {
	log := vzctx.Log
	actualCR := vzctx.ActualCR
	ready, err := r.checkComponentReadyState(vzctx)
	if err != nil {
		return false, err
	}
	if !ready {
		return false, nil
	}
	// Set install complete IFF all subcomponent status' are "CompStateReady"
	message := "Verrazzano install completed successfully"
	// Status update must be performed on the actual CR read from K8S
	return true, r.updateStatus(log, actualCR, message, installv1alpha1.CondInstallComplete, nil)
}

// checkUpgradeComplete checks to see if the upgrade is complete
func (r *Reconciler) checkUpgradeComplete(vzctx vzcontext.VerrazzanoContext) (bool, error) {
	if vzctx.ActualCR == nil {
		return false, nil
	}
	if vzctx.ActualCR.Status.State != installv1alpha1.VzStateUpgrading {
		return true, nil
	}
	log := vzctx.Log
	actualCR := vzctx.ActualCR
	ready, err := r.checkComponentReadyState(vzctx)
	if err != nil {
		return false, err
	}
	if !ready {
		return false, nil
	}
	// Set upgrade complete IFF all subcomponent status' are "CompStateReady"
	message := "Verrazzano upgrade completed successfully"
	// Status and State update must be performed on the actual CR read from K8S
	return true, r.updateVzStatusAndState(log, actualCR, message, installv1alpha1.CondUpgradeComplete, installv1alpha1.VzStateReady)
}

// updateStatus updates the status in the Verrazzano CR
func (r *Reconciler) updateStatus(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, message string, conditionType installv1alpha1.ConditionType, version *string) error {
	t := time.Now().UTC()
	condition := installv1alpha1.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}
	conditions := appendConditionIfNecessary(log, cr.Name, cr.Status.Conditions, condition)

	// Set the state of resource
	state := conditionToVzState(conditionType)
	log.Debugf("Setting Verrazzano resource condition and state: %v/%v", condition.Type, state)

	// Update the status
	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: cr,
		Version:    version,
		State:      state,
		Conditions: conditions,
	})
	return nil
}

// updateVzState updates the status state in the Verrazzano CR
func (r *Reconciler) updateVzState(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, state installv1alpha1.VzStateType) {
	log.Debugf("Setting Verrazzano state: %v", state)
	// Update the status
	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: cr,
		State:      state,
	})
}

// updateVzState updates the status state in the Verrazzano CR
func (r *Reconciler) updateVzStatusAndState(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, message string, conditionType installv1alpha1.ConditionType, state installv1alpha1.VzStateType) error {
	t := time.Now().UTC()
	condition := installv1alpha1.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}
	conditions := appendConditionIfNecessary(log, cr.Name, cr.Status.Conditions, condition)
	log.Debugf("Setting Verrazzano state: %v", state)
	// Update the status
	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: cr,
		State:      state,
		Conditions: conditions,
	})
	return nil
}

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

	componentStatus := cr.Status.Components[componentName]
	if componentStatus == nil {
		componentStatus = &installv1alpha1.ComponentStatusDetails{
			Name: componentName,
		}
	}
	var instanceInfo *installv1alpha1.InstanceInfo
	if conditionType == installv1alpha1.CondInstallComplete {
		instanceInfo = vzinstance.GetInstanceInfo(compContext)
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
	componentStatus.Conditions = appendConditionIfNecessary(log, componentStatus.Name, componentStatus.Conditions, condition)

	// Set the state of resource
	componentStatus.State = checkCondtitionType(conditionType)

	// Set the version of component when install and upgrade complete
	if conditionType == installv1alpha1.CondInstallComplete || conditionType == installv1alpha1.CondUpgradeComplete {
		if bomFile, err := r.getBOM(); err == nil {
			if component, er := bomFile.GetComponent(componentName); er == nil {
				componentStatus.Version = component.Version
			}
		}
	}

	// Update the status
	r.StatusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: cr,
		Components: map[string]*installv1alpha1.ComponentStatusDetails{
			componentName: componentStatus,
		},
		InstanceInfo: instanceInfo,
	})
	return nil
}

func appendConditionIfNecessary(log vzlog.VerrazzanoLogger, resourceName string, conditions []installv1alpha1.Condition, newCondition installv1alpha1.Condition) []installv1alpha1.Condition {
	var newConditionsList []installv1alpha1.Condition
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
		return installv1alpha1.CompStateUninstalled
	case installv1alpha1.CondInstallFailed, installv1alpha1.CondUpgradeFailed, installv1alpha1.CondUninstallFailed:
		return installv1alpha1.CompStateFailed
	}
	// Return ready for installv1alpha1.CondInstallComplete, installv1alpha1.CondUpgradeComplete
	return installv1alpha1.CompStateReady
}

// Convert a condition to a VZ State
func conditionToVzState(currentCondition installv1alpha1.ConditionType) installv1alpha1.VzStateType {
	switch currentCondition {
	case installv1alpha1.CondInstallStarted:
		return installv1alpha1.VzStateReconciling
	case installv1alpha1.CondUninstallStarted:
		return installv1alpha1.VzStateUninstalling
	case installv1alpha1.CondUpgradeStarted:
		return installv1alpha1.VzStateUpgrading
	case installv1alpha1.CondUpgradePaused:
		return installv1alpha1.VzStatePaused
	case installv1alpha1.CondUninstallComplete:
		return installv1alpha1.VzStateReady
	case installv1alpha1.CondInstallFailed, installv1alpha1.CondUpgradeFailed, installv1alpha1.CondUninstallFailed:
		return installv1alpha1.VzStateFailed
	}
	// Return ready for installv1alpha1.CondInstallComplete, installv1alpha1.CondUpgradeComplete
	return installv1alpha1.VzStateReady
}

// setInstallStartedCondition
func (r *Reconciler) setInstallingState(log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	// Set the version in the status.  This will be updated when the starting install condition is updated.
	bomSemVer, err := validators.GetCurrentBomVersion()
	if err != nil {
		return err
	}

	version := bomSemVer.ToString()
	return r.updateStatus(log, vz, "Verrazzano install in progress", installv1alpha1.CondInstallStarted, &version)
}

// checkComponentReadyState returns true if all component-level status' are "CompStateReady" for enabled components
func (r *Reconciler) checkComponentReadyState(vzctx vzcontext.VerrazzanoContext) (bool, error) {
	cr := vzctx.ActualCR
	if unitTesting {
		for _, compStatus := range cr.Status.Components {
			if compStatus.State != installv1alpha1.CompStateDisabled && compStatus.State != installv1alpha1.CompStateReady {
				return false, nil
			}
		}
		return true, nil
	}

	// Return false if any enabled component is not ready
	for _, comp := range registry.GetComponents() {
		spiCtx, err := spi.NewContext(vzctx.Log, r.Client, vzctx.ActualCR, nil, r.DryRun)
		if err != nil {
			spiCtx.Log().Errorf("Failed to create component context: %v", err)
			return false, err
		}
		if comp.IsEnabled(spiCtx.EffectiveCR()) && cr.Status.Components[comp.Name()].State != installv1alpha1.CompStateReady {
			spiCtx.Log().Progressf("Waiting for component %s to be ready", comp.Name())
			return false, nil
		}
	}
	return true, nil
}

// initializeComponentStatus Initialize the component status field with the known set that indicate they support the
// operator-based install.  This is so that we know ahead of time exactly how many components we expect to install
// via the operator, and when we're done installing.
func (r *Reconciler) initializeComponentStatus(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {
	if cr.Status.Components == nil {
		cr.Status.Components = make(map[string]*installv1alpha1.ComponentStatusDetails)
	}

	newContext, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	statusUpdated := false
	componentsToUpdate := map[string]*installv1alpha1.ComponentStatusDetails{}
	for _, comp := range registry.GetComponents() {
		if status, ok := cr.Status.Components[comp.Name()]; ok {
			if status.LastReconciledGeneration == 0 {
				componentsToUpdate[comp.Name()] = status
				status.LastReconciledGeneration = cr.Generation
			}
			// Skip components that have already been processed
			continue
		}
		if comp.IsOperatorInstallSupported() {
			// If the component is installed then mark it as ready
			compContext := newContext.Init(comp.Name()).Operation(vzconst.InitializeOperation)
			lastReconciled := int64(0)
			state := installv1alpha1.CompStateDisabled
			if !unitTesting {
				installed, err := comp.IsInstalled(compContext)
				if err != nil {
					log.Errorf("Failed to determine if component %s is installed: %v", comp.Name(), err)
					return newRequeueWithDelay(), err
				}
				if installed {
					state = installv1alpha1.CompStateReady
					lastReconciled = compContext.ActualCR().Generation
				}
			}
			componentsToUpdate[comp.Name()] = &installv1alpha1.ComponentStatusDetails{
				Name:                     comp.Name(),
				State:                    state,
				LastReconciledGeneration: lastReconciled,
			}
			statusUpdated = true
		}
	}
	// Update the status
	if statusUpdated {
		r.StatusUpdater.Update(&vzstatus.UpdateEvent{
			Verrazzano: cr,
			Components: componentsToUpdate,
		})
		return newRequeueWithDelay(), nil
	}
	return ctrl.Result{}, nil
}

// setUninstallCondition sets the Verrazzano resource condition in status for uninstall
func (r *Reconciler) setUninstallCondition(log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano, newCondition installv1alpha1.ConditionType, msg string) (err error) {
	// Add the uninstall started condition if not already added
	for _, condition := range vz.Status.Conditions {
		if condition.Type == newCondition {
			return nil
		}
	}
	return r.updateStatus(log, vz, msg, newCondition, nil)
}
