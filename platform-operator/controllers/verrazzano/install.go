// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzcontext "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/context"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// reconcileWatchedComponents is the state where watched components are reconciled
	reconcileWatchedComponents reconcileState = "watchedComponents"

	// reconcileVZState is the state where we examine the VZ State and the component generation values and determine what to do
	reconcileVZState reconcileState = "reconcileVZState"

	// reconcileStartInstall is the state where the VZ Install Started status is written
	reconcileStartInstall reconcileState = "startInstall"

	// reconcileInstallComponents is the state where the components are being installed
	reconcileInstallComponents reconcileState = "installComponents"

	// reconcileEnd is the terminal state
	reconcileEnd reconcileState = "reconcileEnd"
)

// reconcileState identifies the state of a VZ reconcile
type reconcileState string

// installTracker has the Install context for the Verrazzano Install
// This tracker keeps an in-memory Install state for Verrazzano and the components that
// are being Install.
type installTracker struct {
	vzState reconcileState
	gen     int64
	compMap map[string]*componentInstallContext
}

// installTrackerMap has a map of InstallTrackers, one entry per Verrazzano CR resource generation
var installTrackerMap = make(map[string]*installTracker)

// getUpgradeTrackerKey gets the tracker key for the Verrazzano resource
func getInstallTrackerKey(cr *vzapi.Verrazzano) string {
	return fmt.Sprintf("%s-%s-%s", cr.Namespace, cr.Name, string(cr.UID))
}

// getInstallTracker gets the install tracker for Verrazzano
func getInstallTracker(cr *vzapi.Verrazzano) *installTracker {
	key := getInstallTrackerKey(cr)
	vuc, ok := installTrackerMap[key]
	// If the entry is missing or the generation is different create a new entry
	if !ok || vuc.gen != cr.Generation {
		vuc = &installTracker{
			vzState: reconcileWatchedComponents,
			gen:     cr.Generation,
			compMap: make(map[string]*componentInstallContext),
		}
		installTrackerMap[key] = vuc
	}
	return vuc
}

// deleteInstallTracker deletes the install tracker for the Verrazzano resource
func deleteInstallTracker(cr *vzapi.Verrazzano) {
	key := getInstallTrackerKey(cr)
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

	for tracker.vzState != reconcileEnd {
		switch tracker.vzState {

		// reconcileWatchedComponents reconciles first to fix up any broken components
		case reconcileWatchedComponents:
			if spiCtx.ActualCR().Status.State != vzapi.VzStateUpgrading {
				if err := r.reconcileWatchedComponents(spiCtx); err != nil {
					return ctrl.Result{Requeue: true}, err
				}
			}
			tracker.vzState = reconcileVZState

		case reconcileVZState:
			// reconcileComponents is called from Ready, Reconciling, and Upgrading states
			// if the VZ state is Ready, start an install if the generation is updated and end reconciling if not
			// if the VZ state is not Ready, proceed with installing components
			if spiCtx.ActualCR().Status.State == vzapi.VzStateReady {
				if checkGenerationUpdated(spiCtx) {
					tracker.vzState = reconcileStartInstall
				} else {
					tracker.vzState = reconcileEnd
				}
				continue
			}
			tracker.vzState = reconcileInstallComponents

		case reconcileStartInstall:
			if err := r.setInstallingState(vzctx.Log, spiCtx.ActualCR()); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			tracker.vzState = reconcileInstallComponents

		case reconcileInstallComponents:
			res, err := r.installComponents(spiCtx, tracker, preUpgrade)
			if err != nil || res.Requeue {
				return res, err
			}
			tracker.vzState = reconcileEnd
		}
	}

	deleteInstallTracker(spiCtx.ActualCR())
	return ctrl.Result{}, nil
}

// checkGenerationUpdated loops through the components and calls checkConfigUpdated on each
func checkGenerationUpdated(spiCtx spi.ComponentContext) bool {
	for _, comp := range registry.GetComponents() {
		componentStatus, ok := spiCtx.ActualCR().Status.Components[comp.Name()]
		if !ok {
			spiCtx.Log().Debugf("Did not find status details in map for component %s", comp.Name())
			// if we can't find the component status, enter install loop to try to fix it
			return true
		}
		if checkConfigUpdated(spiCtx, componentStatus, comp.Name()) &&
			comp.IsEnabled(spiCtx.EffectiveCR()) &&
			comp.MonitorOverrides(spiCtx) {

			spiCtx.Log().Oncef("Verrazzano CR generation change detected, generation: %v, component: %s, component reconciling generation: %v, component lastreconciling generation %v",
				spiCtx.ActualCR().Generation, comp.Name(), componentStatus.ReconcilingGeneration, componentStatus.LastReconciledGeneration)
			return true
		}
	}
	return false
}

// reconcileWatchedComponents loops through the components and calls the component Reconcile function
// if it a watched component
func (r *Reconciler) reconcileWatchedComponents(spiCtx spi.ComponentContext) error {
	for _, comp := range registry.GetComponents() {
		if r.IsWatchedComponent(comp.GetJSONName()) {
			if err := comp.Reconcile(spiCtx); err != nil {
				spiCtx.Log().ErrorfThrottled("Error reconciling component %: %v", comp.Name(), err)
				return err
			}
			r.ClearWatch(comp.GetJSONName())
		}
	}
	return nil
}
