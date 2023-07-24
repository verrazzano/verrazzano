// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	modulestatus "github.com/verrazzano/verrazzano-modules/module-operator/controllers/module/status"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"strconv"
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
// operator-based installation.  This is so that we know ahead of time exactly how many components we expect to install
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

// updateStatusForComponents updates the vz CR status for all the components based on the module status.
// Requeue if all components are not ready.
func (r Reconciler) updateStatusForAllComponents(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano) result.Result {
	var readyCount int
	var moduleCount int

	for _, comp := range registry.GetComponents() {
		if !comp.IsEnabled(effectiveCR) {
			continue
		}
		if !comp.ShouldUseModule() {
			continue
		}
		moduleCount++

		// get the module
		module := &moduleapi.Module{}
		if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoInstallNamespace, Name: comp.Name()}, module); err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			log.ErrorfThrottled("Failed getting Module %s: %v", comp.Name(), err)
			continue
		}
		// Set the VZ status from the module status
		compStatus, err := r.updateStatusForSingleComponent(effectiveCR, comp.Name(), module)
		if err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
		if compStatus != nil && compStatus.State == vzapi.CompStateReady {
			readyCount++
		}
	}

	// Update the status.  If it didn't change then the Kubernetes API server will not be called
	err := r.Client.Status().Update(context.TODO(), effectiveCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	if moduleCount != readyCount {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	return result.NewResult()
}

// updateStatusForSingleComponent the Verrazzano CR component status with the latest status from the Module
func (r *Reconciler) updateStatusForSingleComponent(vzcr *vzapi.Verrazzano, compName string, module *moduleapi.Module) (*vzapi.ComponentStatusDetails, error) {
	compStatus := vzcr.Status.Components[compName]
	if compStatus == nil {
		// legacy Verrazzano controller has not initialized the component status yet.
		return nil, nil
	}

	// Get the Module ready condition, return if it doesn't exist yet
	cond := modulestatus.GetReadyCondition(module)
	if cond == nil {
		return compStatus, nil
	}

	// Make sure the module has the current Verrazzano generation, it not then we have an old copy of the module.
	if module.Annotations == nil {
		return compStatus, nil
	}
	gen, _ := module.Annotations[vzconst.VerrazzanoObservedGenerationAnnotation]
	if gen != strconv.FormatInt(vzcr.Generation, 10) {
		return compStatus, nil
	}

	// Init vars used to update the VZ component status
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

	return compStatus, nil
}
