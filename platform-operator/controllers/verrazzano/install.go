// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	ctrl "sigs.k8s.io/controller-runtime"

	"go.uber.org/zap"
)

// reconcileComponents Reconcile components individually
func (r *Reconciler) reconcileComponents(_ context.Context, log *zap.SugaredLogger, cr *vzapi.Verrazzano) (ctrl.Result, error) {

	result := ctrl.Result{}

	// Loop through all of the Verrazzano components and upgrade each one sequentially for now; will parallelize later
	for _, comp := range registry.GetComponents() {
		if !comp.IsOperatorInstallSupported() {
			continue
		}
		componentState := cr.Status.Components[comp.Name()].State
		switch componentState {
		case vzapi.Ready:
			// For delete, we should look at the VZ resource delete timestamp and shift into Quiescing/Uninstalling state
			continue
		case vzapi.Disabled:
			if !isComponentEnabled(cr, comp.Name()) {
				// User has disabled component in Verrazzano CR, don't install
				continue
			}
			r.updateComponentStatus(log, cr, comp.Name(), "Install started", vzapi.InstallStarted)
			result.Requeue = true
			continue
		case vzapi.Installing:
			// For delete, we should look at the VZ resource delete timestamp and shift into Quiescing/Uninstalling state
			// If component is enabled -- need to replicate scripts' config merging logic here
			// If component is in deployed state, continue
			if comp.IsReady(log, r.Client, cr.Namespace) {
				log.Infof("Component %s successfully installed")
				if err := r.updateComponentStatus(log, cr, comp.Name(), "Install complete", vzapi.InstallComplete); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
				result.Requeue = true
				continue
			}
			if !registry.ComponentDependenciesMet(log, r.Client, comp) {
				log.Infof("Dependencies not met for %s: %v", comp.Name(), comp.GetDependencies())
				result.Requeue = true
				continue
			}
			if err := r.updateComponentStatus(log, cr, comp.Name(), "Install starting", vzapi.InstallStarted); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			// If component is not installed,install it
			if err := comp.Install(log, r, cr.Namespace, r.DryRun); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			//case vzapi.Failed, vzapi.Error:
			//case vzapi.Disabled:
			//case vzapi.Upgrading:
			//case vzapi.Updating:
			//case vzapi.Quiescing:
		}
	}
	return result, nil
}

// IsEnabled returns true if the component spec has enabled set to true
// Enabled=true is the default
func isComponentEnabled(cr *vzapi.Verrazzano, componentName string) bool {
	switch componentName {
	case coherence.ComponentName:
		if cr.Spec.Components.Coherence == nil || cr.Spec.Components.Coherence.Enabled == nil {
			return true
		}
		return *cr.Spec.Components.Coherence.Enabled
	case weblogic.ComponentName:
		if cr.Spec.Components.WebLogic == nil || cr.Spec.Components.WebLogic.Enabled == nil {
			return true
		}
		return *cr.Spec.Components.WebLogic.Enabled
	}
	return true
}
