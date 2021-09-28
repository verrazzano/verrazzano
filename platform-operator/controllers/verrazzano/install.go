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

// reconcileComponents reconciles each component using the following rules:
// 1. Always requeue until all enabled components have completed installation
// 2. Don't update the component state until all the work in that state is done, since
//    that update will cause a state transition
// 3. Loop through all components before returning, except for the case
//    where update status fails, in which case we exit the function and requeue
//    immediately.
func (r *Reconciler) reconcileComponents(_ context.Context, log *zap.SugaredLogger, cr *vzapi.Verrazzano) (ctrl.Result, error) {

	var requeue bool

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
			if err := r.updateComponentStatus(log, cr, comp.Name(), "PreInstall started", vzapi.PreInstall); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			requeue = true

		case vzapi.PreInstalling:
			log.Infof("PreInstalling component %s", comp.Name())
			if !registry.ComponentDependenciesMet(log, r.Client, comp) {
				log.Infof("Dependencies not met for %s: %v", comp.Name(), comp.GetDependencies())
				requeue = true
				continue
			}
			if err := comp.PreInstall(log, r, cr.Namespace, r.DryRun); err != nil {
				log.Errorf("Error calling comp.PreInstall for component %s: %v", comp.Name(), err.Error())
				requeue = true
				continue
			}
			// If component is not installed,install it
			if err := comp.Install(log, r, cr.Namespace, r.DryRun); err != nil {
				log.Errorf("Error calling comp.Install for component %s: %v", comp.Name(), err.Error())
				requeue = true
				continue
			}
			if err := r.updateComponentStatus(log, cr, comp.Name(), "Install started", vzapi.InstallStarted); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			// Install started requeue to check status
			requeue = true
		case vzapi.Installing:
			// For delete, we should look at the VZ resource delete timestamp and shift into Quiescing/Uninstalling state
			// If component is enabled -- need to replicate scripts' config merging logic here
			// If component is in deployed state, continue
			if comp.IsReady(log, r.Client, cr.Namespace) {
				if err := comp.PostInstall(log, r, cr.Namespace, r.DryRun); err != nil {
					return newRequeueWithDelay(), err
				}
				log.Infof("Component %s successfully installed", comp.Name())
				if err := r.updateComponentStatus(log, cr, comp.Name(), "Install complete", vzapi.InstallComplete); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
				// Don't requeue because of this component, it is done install
				continue
			}
			// Install of this component is not done, requeue to check status
			requeue = true
		}
	}
	if requeue {
		return newRequeueWithDelay(), nil
	}
	return ctrl.Result{}, nil
}

// IsEnabled returns true if the component spec has enabled set to true
// Enabled=true is the default
func isComponentEnabled(cr *vzapi.Verrazzano, componentName string) bool {
	switch componentName {
	case coherence.ComponentName:
		return coherence.IsEnabled(cr.Spec.Components.CoherenceOperator)
	case weblogic.ComponentName:
		return weblogic.IsEnabled(cr.Spec.Components.WebLogicOperator)
	}
	return true
}
