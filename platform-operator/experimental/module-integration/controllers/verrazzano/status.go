// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulestatus "github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
)

// updateVzState updates the status state in the Verrazzano CR
func (r *Reconciler) updateStatus(log vzlog.VerrazzanoLogger, cr *vzapi.Verrazzano) result.Result {
	err := r.Client.Status().Update(context.TODO(), cr)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}

// initializeComponentStatus Initialize the component status field with the known set that indicate they support the
// operator-based install.  This is so that we know ahead of time exactly how many components we expect to install
// via the operator, and when we're done installing.
func (r *Reconciler) initializeComponentStatus(log vzlog.VerrazzanoLogger, cr *vzapi.Verrazzano) result.Result {
	if cr.Status.Components == nil {
		cr.Status.Components = make(map[string]*vzapi.ComponentStatusDetails)
	}

	newContext, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelay()
	}

	statusUpdated := false
	for _, comp := range registry.GetComponents() {
		_, ok := cr.Status.Components[comp.Name()]
		if ok {
			// Skip components that have already been processed
			continue
		}
		if comp.IsOperatorInstallSupported() {
			// If the component is installed then mark it as ready
			compContext := newContext.Init(comp.Name()).Operation(vzconst.InitializeOperation)
			lastReconciled := int64(0)
			state := vzapi.CompStateDisabled
			if !unitTesting {
				installed, err := comp.IsInstalled(compContext)
				if err != nil {
					log.Errorf("Failed to determine if component %s is installed: %v", comp.Name(), err)
					return result.NewResultShortRequeueDelay()

				}
				if installed {
					state = vzapi.CompStateReady
					lastReconciled = compContext.ActualCR().Generation
				}
			}
			compStatus := &vzapi.ComponentStatusDetails{
				Name:                     comp.Name(),
				State:                    state,
				LastReconciledGeneration: lastReconciled,
			}
			cr.Status.Components[comp.Name()] = compStatus
			statusUpdated = true
		}
	}
	// Update the status
	if statusUpdated {
		err := r.Client.Status().Update(context.TODO(), cr)
		if err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
		return result.NewResultShortRequeueDelay()
	}
	return result.NewResult()
}

// Update the component status in place for VZ CR
func (r *Reconciler) loadModuleStatusIntoComponentStatus(vzcr *vzapi.Verrazzano, compName string, module *moduleapi.Module) *vzapi.ComponentStatusDetails {
	compStatus := vzcr.Status.Components[compName]
	if compStatus == nil {
		// legacy verrazzano controller has not initialized the component status yet.
		return nil
	}

	cond := modulestatus.GetReadyCondition(module)
	if cond == nil {
		return compStatus
	}

	var available vzapi.ComponentAvailability = vzapi.ComponentUnavailable
	var state = vzapi.CompStateReconciling
	lastGen := compStatus.LastReconciledGeneration

	// The module is only done when it is ready condition is true and the
	// last reconciled generation matches the spec generation
	if cond.Status == corev1.ConditionTrue && module.Status.LastSuccessfulGeneration == module.Generation {
		available = vzapi.ComponentAvailable
		state = vzapi.CompStateReady
		lastGen = vzcr.Generation
	}

	compStatus.State = state
	compStatus.Available = &available
	compStatus.LastReconciledGeneration = lastGen
	compStatus.ReconcilingGeneration = vzcr.Generation

	return compStatus
}

//
//// updateStatus updates the status in the Verrazzano CR
//func (r *Reconciler) updateStatus(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, message string, conditionType installv1alpha1.ConditionType, version *string) error {
//	t := time.Now().UTC()
//	condition := installv1alpha1.Condition{
//		Type:    conditionType,
//		Status:  corev1.ConditionTrue,
//		Message: message,
//		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
//			t.Year(), t.Month(), t.Day(),
//			t.Hour(), t.Minute(), t.Second()),
//	}
//	conditions := appendConditionIfNecessary(log, cr.Name, cr.Status.Conditions, condition)
//
//	// Set the state of resource
//	state := conditionToVzState(conditionType)
//	log.Debugf("Setting Verrazzano resource condition and state: %v/%v", condition.Type, state)
//
//	event := &vzstatus.UpdateEvent{
//		Verrazzano: cr,
//		Version:    version,
//		State:      state,
//		Conditions: conditions,
//	}
//
//	if conditionType == installv1alpha1.CondInstallComplete {
//		spiCtx, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
//		if err != nil {
//			spiCtx.Log().Errorf("Failed to create component context: %v", err)
//			return err
//		}
//		event.InstanceInfo = vzinstance.GetInstanceInfo(spiCtx)
//	}
//
//	// Update the status
//	r.StatusUpdater.Update(event)
//	return nil
//}
