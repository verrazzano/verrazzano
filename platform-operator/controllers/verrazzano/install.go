// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"fmt"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"

	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Reconcile upgrade will upgrade the components as required
func (r *Reconciler) reconcileInstall(log *zap.SugaredLogger, _ ctrl.Request, cr *installv1alpha1.Verrazzano) error {

	// Loop through all of the Verrazzano components and upgrade each one sequentially for now; will parallelize later
	for _, comp := range registry.GetComponents() {
		if !comp.IsOperatorInstallSupported() {
			continue
		}
		// If component is enabled -- need to replicate scripts' config merging logic here
		// If component is in deployed state, continue
		if comp.IsReady(log, r.Client, cr.Namespace) {
			log.Infof("Component %s already installed")
			if err := r.updateComponentStatus(log, cr, comp.Name(), "Update ready status", installv1alpha1.InstallComplete); err != nil {
				return err
			}
			continue
		}
		if !registry.ComponentDependenciesMet(log, r.Client, comp) {
			return fmt.Errorf("Dependencies not met for %s: %v", comp.Name(), comp.GetDependencies())
		}
		if err := r.updateComponentStatus(log, cr, comp.Name(), "Install starting", installv1alpha1.InstallStarted); err != nil {
			return err
		}
		// If component is not installed,install it
		if err := comp.Install(log, r, cr.Namespace, r.DryRun); err != nil {
			return err
		}
		if err := r.updateComponentStatus(log, cr, comp.Name(), "Install complete", installv1alpha1.InstallComplete); err != nil {
			return err
		}
	}
	return nil
}
