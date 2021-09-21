// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	ctrl "sigs.k8s.io/controller-runtime"

	"go.uber.org/zap"
)

// reconcileComponents Reconcile components individually
func (r *Reconciler) reconcileComponents(_ context.Context, log *zap.SugaredLogger, cr *installv1alpha1.Verrazzano) (ctrl.Result, error) {

	result := ctrl.Result{}

	// Loop through all of the Verrazzano components and upgrade each one sequentially for now; will parallelize later
	for _, comp := range registry.GetComponents() {
		if !comp.IsOperatorInstallSupported() {
			continue
		}
		componentState := cr.Status.Components[comp.Name()].State
		switch componentState {
		case installv1alpha1.Ready:
			// For delete, we should look at the VZ resource delete timestamp and shift into Quiescing/Uninstalling state
			continue
		case installv1alpha1.Disabled:
			r.updateComponentStatus(log, cr, comp.Name(), "Install started", installv1alpha1.InstallStarted)
			result.Requeue = true
			continue
		case installv1alpha1.Installing:
			// For delete, we should look at the VZ resource delete timestamp and shift into Quiescing/Uninstalling state
			// If component is enabled -- need to replicate scripts' config merging logic here
			// If component is in deployed state, continue
			if comp.IsReady(log, r.Client, cr.Namespace, r.DryRun) {
				if err := comp.PostInstall(log, r, cr.Namespace, r.DryRun); err != nil {
					return newRequeueWithDelay(), err
				}
				log.Infof("Component %s successfully installed")
				if err := r.updateComponentStatus(log, cr, comp.Name(), "Install complete", installv1alpha1.InstallComplete); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
				result.Requeue = true
				continue
			}
			if !registry.ComponentDependenciesMet(log, r.Client, comp, r.DryRun) {
				log.Infof("Dependencies not met for %s: %v", comp.Name(), comp.GetDependencies())
				result.Requeue = true
				continue
			}
			if err := r.updateComponentStatus(log, cr, comp.Name(), "Install starting", installv1alpha1.InstallStarted); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			if err := comp.PreInstall(log, r, cr.Namespace, r.DryRun); err != nil {
				return newRequeueWithDelay(), err
			}
			// If component is not installed,install it
			if err := comp.Install(log, r, cr.Namespace, r.DryRun); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			//case installv1alpha1.Failed, installv1alpha1.Error:
			//case installv1alpha1.Disabled:
			//case installv1alpha1.Upgrading:
			//case installv1alpha1.Updating:
			//case installv1alpha1.Quiescing:
		}
	}
	return result, nil
}
