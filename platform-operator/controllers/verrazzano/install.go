// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	ctrl "sigs.k8s.io/controller-runtime"

	"go.uber.org/zap"
)

// reconcileComponents Reconcile components individually
func (r *Reconciler) reconcileComponents(_ context.Context, log *zap.SugaredLogger, cr *vzapi.Verrazzano) (ctrl.Result, error) {

	var requeue bool

	compContext := spi.ComponentContext{
		Client:          r,
		DryRun:          r.DryRun,
		Config:          cr,
		EffectiveConfig: cr,
	}

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
			r.updateComponentStatus(log, cr, comp.Name(), "PreInstall started", vzapi.PreInstall)
			requeue = true
			continue
		case vzapi.PreInstalling:
			log.Infof("PreInstalling component %s", comp.Name())
			if !registry.ComponentDependenciesMet(log, comp, &compContext) {
				log.Infof("Dependencies not met for %s: %v", comp.Name(), comp.GetDependencies())
				requeue = true
				continue
			}
			if err := comp.PreInstall(log, &compContext); err != nil {
				return newRequeueWithDelay(), err
			}
			// If component is not installed,install it
			if err := comp.Install(log, &compContext); err != nil {
				return newRequeueWithDelay(), err
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
			if comp.IsReady(log, nil) {
				if err := comp.PostInstall(log, &compContext); err != nil {
					return newRequeueWithDelay(), err
				}
				log.Infof("Component %s successfully installed", comp.Name())
				if err := r.updateComponentStatus(log, cr, comp.Name(), "Install complete", vzapi.InstallComplete); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
				requeue = true
				continue
			}
			// Install started requeue to check status
			requeue = true

			//case vzapi.Failed, vzapi.Error:
			//case vzapi.Disabled:
			//case vzapi.Upgrading:
			//case vzapi.Updating:
			//case vzapi.Quiescing:
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
		return coherence.IsEnabled(cr.Spec.Components.Coherence)
	case weblogic.ComponentName:
		return weblogic.IsEnabled(cr.Spec.Components.WebLogic)
	}
	return true
}
