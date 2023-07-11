// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulestatus "github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzcontext "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"
	corev1 "k8s.io/api/core/v1"
)

// updateVzState updates the status state in the Verrazzano CR
func (r *Reconciler) updateStatus(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) result.Result {
	err := r.Client.Status().Update(context.TODO(), cr)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
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
			vzctx.Log.Errorf("Failed to create component context: %v", err)
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
func (r *Reconciler) initializeComponentStatus(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) result.Result {
	if cr.Status.Components == nil {
		cr.Status.Components = make(map[string]*installv1alpha1.ComponentStatusDetails)
	}

	newContext, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelay()
	}

	statusUpdated := false
	for _, comp := range registry.GetComponents() {
		compStatus, ok := cr.Status.Components[comp.Name()]
		if ok {
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
					return result.NewResultShortRequeueDelay()

				}
				if installed {
					state = installv1alpha1.CompStateReady
					lastReconciled = compContext.ActualCR().Generation
				}
			}
			compStatus = &installv1alpha1.ComponentStatusDetails{
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
