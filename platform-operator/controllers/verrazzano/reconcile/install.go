// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/argocd"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzcontext "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// vzStateReconcileWatchedComponents is the state where watched components are reconciled
	vzStateReconcileWatchedComponents reconcileState = "vzReconcileWatchedComponents"

	// vzStateDecideUpdateNeeded is the state where we examine the VZ State and the component generation values and determine what to do
	vzStateDecideUpdateNeeded reconcileState = "vzDecideUpdateNeeded"

	// vzStateSetGlobalInstallStatus is the state where the VZ Install Started status is written
	vzStateSetGlobalInstallStatus reconcileState = "vzSetGlobalInstallStatus"

	// vzStatePreInstall is the global preinstall state
	vzStatePreInstall reconcileState = "vzPreInstall"

	// vzStateInstallComponents is the state where the components are being installed
	vzStateInstallComponents reconcileState = "vzInstallComponents"

	// vzStateWaitModulesReady wait for components installed using Modules to be ready
	vzStateWaitModulesReady reconcileState = "vzWaitModulesReady"

	// vzStatePostInstall is the global PostInstall state
	vzStatePostInstall reconcileState = "vzPostInstall"

	// vzStateReconcileEnd is the terminal state
	vzStateReconcileEnd reconcileState = "vzReconcileEnd"
)

// reconcileState identifies the state of a VZ reconcile
type reconcileState string

// installTracker has the Install context for the Verrazzano Install
// This tracker keeps an in-memory Install state for Verrazzano and the components that
// are being Install.
type installTracker struct {
	vzState reconcileState
	gen     int64
	compMap map[string]*componentTrackerContext
}

// installTrackerMap has a map of InstallTrackers with key from VZ name, namespace, and UID
var installTrackerMap = make(map[string]*installTracker)

// getTrackerKey gets the tracker key for the Verrazzano resource
func getTrackerKey(cr *vzapi.Verrazzano) string {
	return fmt.Sprintf("%s-%s-%s", cr.Namespace, cr.Name, string(cr.UID))
}

// getInstallTracker gets the install tracker for Verrazzano
func getInstallTracker(cr *vzapi.Verrazzano) *installTracker {
	key := getTrackerKey(cr)
	vuc, ok := installTrackerMap[key]
	// If the entry is missing or the generation is different create a new entry
	if !ok || vuc.gen != cr.Generation {
		vuc = &installTracker{
			vzState: vzStateReconcileWatchedComponents,
			gen:     cr.Generation,
			compMap: make(map[string]*componentTrackerContext),
		}
		installTrackerMap[key] = vuc
	}
	return vuc
}

// deleteInstallTracker deletes the install tracker for the Verrazzano resource
func deleteInstallTracker(cr *vzapi.Verrazzano) {
	key := getTrackerKey(cr)
	_, ok := installTrackerMap[key]
	if ok {
		delete(installTrackerMap, key)
	}
}

// reconcileComponents reconciles the components and the VZ State and determines what to do
// from this function, the possible outcomes are
// - global install is started
// - individual components are installed if a global install has already been started
// - a watched component is reconciled
// - this function completes and nothing happens
func (r *Reconciler) reconcileComponents(vzctx vzcontext.VerrazzanoContext, preUpgrade bool) (ctrl.Result, error) {
	spiCtx, err := spi.NewContext(vzctx.Log, r.Client, vzctx.ActualCR, nil, r.DryRun)
	if err != nil {
		spiCtx.Log().Errorf("Failed to create component context: %v", err)
		return newRequeueWithDelay(), err
	}

	tracker := getInstallTracker(spiCtx.ActualCR())

	for tracker.vzState != vzStateReconcileEnd {
		switch tracker.vzState {

		// vzStateReconcileWatchedComponents reconciles first to fix up any broken components
		case vzStateReconcileWatchedComponents:
			if spiCtx.ActualCR().Status.State != vzapi.VzStateUpgrading {
				// loop through all the components and call comp.Reconcile if the component is on the watched list
				if err := r.reconcileWatchedComponents(spiCtx); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
			}
			tracker.vzState = vzStateDecideUpdateNeeded

		case vzStateDecideUpdateNeeded:
			// reconcileComponents is called from Ready, Reconciling, and Upgrading states
			// if the VZ state is Ready, start an install if the generation is updated and end reconciling if not
			// if the VZ state is not Ready, proceed with installing components
			if spiCtx.ActualCR().Status.State == vzapi.VzStateReady {
				if checkGenerationUpdated(spiCtx) {
					// Start global upgrade
					tracker.vzState = vzStateSetGlobalInstallStatus
				} else {
					tracker.vzState = vzStateReconcileEnd
				}
				continue
			}
			// if the VZ state is not Ready, it must be Reconciling or Upgrading
			// in either case, go right to PreInstall
			tracker.vzState = vzStatePreInstall

		case vzStateSetGlobalInstallStatus:
			spiCtx.Log().Oncef("Writing Install Started condition to the Verrazzano status for generation: %d", spiCtx.ActualCR().Generation)
			if err := r.setInstallingState(vzctx.Log, spiCtx.ActualCR()); err != nil {
				spiCtx.Log().ErrorfThrottled("Error writing Install Started condition to the Verrazzano status: %v", err)
				return ctrl.Result{Requeue: true}, err
			}
			tracker.vzState = vzStatePreInstall
			// since we updated the status, requeue to pick up new changes
			return ctrl.Result{Requeue: true}, nil

		case vzStatePreInstall:
			r.beforeInstallComponents(spiCtx)
			SetPreModuleWorkDone(true)
			tracker.vzState = vzStateInstallComponents

		case vzStateInstallComponents:
			res, err := r.installComponents(spiCtx, tracker, preUpgrade)
			if err != nil || res.Requeue {
				return res, err
			}
			tracker.vzState = vzStateWaitModulesReady

		case vzStatePostInstall:
			if !preUpgrade {
				if err := rancher.ConfigureAuthProviders(spiCtx); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
				if err := argocd.ConfigureKeycloakOIDC(spiCtx); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
			}
			tracker.vzState = vzStateReconcileEnd
		}
	}

	deleteInstallTracker(spiCtx.ActualCR())
	return ctrl.Result{}, nil
}

// checkGenerationUpdated loops through the components and calls checkConfigUpdated on each
func checkGenerationUpdated(spiCtx spi.ComponentContext) bool {
	for _, comp := range registry.GetComponents() {
		if comp.ShouldUseModule() {
			// Ignore if this component is being handled by a Module
			continue
		}
		if comp.IsEnabled(spiCtx.EffectiveCR()) {
			componentStatus, ok := spiCtx.ActualCR().Status.Components[comp.Name()]
			if !ok {
				spiCtx.Log().Debugf("Did not find status details in map for component %s", comp.Name())
				// if we can't find the component status, enter install loop to try to fix it
				return true
			}
			if checkConfigUpdated(spiCtx, componentStatus) && comp.MonitorOverrides(spiCtx) {
				spiCtx.Log().Oncef("Verrazzano CR generation change detected, generation: %v, component: %s, component reconciling generation: %v, component lastreconciling generation %v",
					spiCtx.ActualCR().Generation, comp.Name(), componentStatus.ReconcilingGeneration, componentStatus.LastReconciledGeneration)
				return true
			}
		}
	}
	return false
}

// vzStateReconcileWatchedComponents loops through the components and calls the component Reconcile function
// if it a watched component
func (r *Reconciler) reconcileWatchedComponents(spiCtx spi.ComponentContext) error {
	for _, comp := range registry.GetComponents() {
		if comp.ShouldUseModule() {
			// Ignore if this component is being handled by a Module
			continue
		}

		spiCtx.Log().Debugf("Reconciling watched component %s", comp.Name())
		if r.IsWatchedComponent(comp.GetJSONName()) {
			if err := comp.Reconcile(spiCtx); err != nil {
				spiCtx.Log().ErrorfThrottled("Error reconciling watched component %s: %v", comp.Name(), err)
				return err
			}
			r.ClearWatch(comp.GetJSONName())
		}
	}
	return nil
}

func (r *Reconciler) beforeInstallComponents(ctx spi.ComponentContext) {
	r.createRancherIngressAndCertCopies(ctx)
}
func (r *Reconciler) waitForModulesReady(compContext spi.ComponentContext) (ctrl.Result, error) {
	// Loop through all the Verrazzano components being handled by Modules and check if ready
	for _, comp := range registry.GetComponents() {
		if !comp.ShouldUseModule() {
			// Ignore if this component is NOT being handled by a Module
			continue
		}
		if !comp.IsEnabled(compContext.EffectiveCR()) {
			continue
		}
		if !comp.IsReady(compContext) {
			compContext.Log().Oncef("Waiting for the module %s to be ready", comp.Name())
			return newRequeueWithDelay(), nil
		}
	}
	return ctrl.Result{}, nil
}
