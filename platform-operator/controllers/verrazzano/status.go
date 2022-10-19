// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/vzinstance"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

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
