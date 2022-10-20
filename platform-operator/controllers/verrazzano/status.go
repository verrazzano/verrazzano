// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzcontext "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"
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
	return true, r.updateStatus(log, actualCR, message, installv1alpha1.CondInstallComplete)
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
func (r *Reconciler) updateStatus(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, message string, conditionType installv1alpha1.ConditionType) error {
	t := time.Now().UTC()
	condition := installv1alpha1.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}
	cr.Status.Conditions = appendConditionIfNecessary(log, cr.Name, cr.Status.Conditions, condition)

	// Set the state of resource
	cr.Status.State = conditionToVzState(conditionType)
	log.Debugf("Setting Verrazzano resource condition and state: %v/%v", condition.Type, cr.Status.State)

	// Update the status
	return r.updateVerrazzanoStatus(log, cr)
}

// updateVzState updates the status state in the Verrazzano CR
func (r *Reconciler) updateVzState(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, state installv1alpha1.VzStateType) error {
	// Set the state of resource
	cr.Status.State = state
	log.Debugf("Setting Verrazzano state: %v", cr.Status.State)

	// Update the status
	return r.updateVerrazzanoStatus(log, cr)
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
	cr.Status.Conditions = appendConditionIfNecessary(log, cr.Name, cr.Status.Conditions, condition)

	// Set the state of resource
	cr.Status.State = state
	log.Debugf("Setting Verrazzano state: %v", cr.Status.State)

	// Update the status
	return r.updateVerrazzanoStatus(log, cr)
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
		cr.Status.VerrazzanoInstance = vzinstance.GetInstanceInfo(compContext)
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
	return r.updateVerrazzanoStatus(log, cr)
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

	vz.Status.Version = bomSemVer.ToString()
	return r.updateStatus(log, vz, "Verrazzano install in progress", installv1alpha1.CondInstallStarted)
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
	for _, comp := range registry.GetComponents() {
		if status, ok := cr.Status.Components[comp.Name()]; ok {
			if status.LastReconciledGeneration == 0 {
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
			cr.Status.Components[comp.Name()] = &installv1alpha1.ComponentStatusDetails{
				Name:                     comp.Name(),
				State:                    state,
				LastReconciledGeneration: lastReconciled,
			}
			statusUpdated = true
		}
	}
	// Update the status
	if statusUpdated {
		return newRequeueWithDelay(), r.updateVerrazzanoStatus(log, cr)
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
	return r.updateStatus(log, vz, msg, newCondition)
}

func (r *Reconciler) updateVerrazzanoStatus(log vzlog.VerrazzanoLogger, vz *installv1alpha1.Verrazzano) error {
	r.HealthCheck.SetAvailabilityStatus(vz)
	err := r.Status().Update(context.TODO(), vz)
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
