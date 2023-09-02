// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/vzlog"
	vpovzlog "github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// initializeComponentStatus Initialize the component status field with the known set that indicate they support the
// operator-based installation.  This is so that we know ahead of time exactly how many components we expect to install
// via the operator, and when we're done installing.
func (r *Reconciler) initializeComponentStatus(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano) result.Result {
	if cr.Status.Components == nil {
		cr.Status.Components = make(map[string]*installv1alpha1.ComponentStatusDetails)
	}

	newContext, err := spi.NewContext(vpovzlog.DefaultLogger(), r.Client, cr, nil, r.DryRun)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}

	statusUpdated := false
	for _, comp := range registry.GetComponents() {
		if status, ok := cr.Status.Components[comp.Name()]; ok {
			if status.LastReconciledGeneration == 0 {
				cr.Status.Components[comp.Name()] = status
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
			installed, err := comp.IsInstalled(compContext)
			if err != nil {
				log.Errorf("Failed to determine if component %s is installed: %v", comp.Name(), err)
				return result.NewResultShortRequeueDelayWithError(err)
			}
			if installed {
				state = installv1alpha1.CompStateReady
				lastReconciled = compContext.ActualCR().Generation
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
		// Use Status update directly so any conflicting updates get rejected.
		// This is needed for integration with the new Verrazzano controller used for modules
		// Basically, that controller was seeing the component status updates and creating
		// Module CRs, which in turn updated the component status conditions.  However, this code
		// was subsequently re-initializing the component status because it didn't know there was an update conflict
		// and that it needed to requeue, so it was using a stale copy of the VZ CR.
		r.Client.Status().Update(context.TODO(), cr)
		return result.NewResultShortRequeueDelayWithError(err)
	}
	return result.NewResult()
}
