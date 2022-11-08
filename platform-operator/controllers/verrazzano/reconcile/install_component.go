// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	ctrl "sigs.k8s.io/controller-runtime"
)

// componentInstallState identifies the state of a component during install
type componentInstallState string

const (
	// compStateInstallInitDetermineComponentState is the state when a component is initialized
	compStateInstallInitDetermineComponentState componentInstallState = "componentStateInit"

	// compStateInstallInitReady is the state when a component is ready
	compStateInstallInitReady componentInstallState = "componentStateReady"

	// compStateInstallInitDisabled is the state when a component is disabled
	compStateInstallInitDisabled componentInstallState = "componentStateDisabled"

	// compStateWriteInstallStartedStatus is the state when a component writes the Install Started status condition
	compStateWriteInstallStartedStatus componentInstallState = "componentStateInstallStarted"

	// compStatePreInstall is the state when a component does a pre-install
	compStatePreInstall componentInstallState = "compStatePreInstall"

	// compStateInstall is the state where a component does an install
	compStateInstall componentInstallState = "compStateInstall"

	// compStateInstallWaitReady is the state when a component is waiting for install to be ready
	compStateInstallWaitReady componentInstallState = "compStateInstallWaitReady"

	// compStatePostInstall is the state when a component is doing a post-install
	compStatePostInstall componentInstallState = "compStatePostInstall"

	// compStateInstallComplete is the state when component writes the Install Complete status
	compStateInstallComplete componentInstallState = "compStateInstallComplete"

	// compStateInstallEnd is the terminal state
	compStateInstallEnd componentInstallState = "compStateInstallEnd"
)

// componentInstallContext has the install context for a Verrazzano component install
type componentInstallContext struct {
	state componentInstallState
}

// installComponents will install the components as required
func (r *Reconciler) installComponents(spiCtx spi.ComponentContext, tracker *installTracker, preUpgrade bool) (ctrl.Result, error) {
	spiCtx.Log().Progress("Installing components")

	var requeue bool

	// Loop through all of the Verrazzano components and install each one
	for _, comp := range registry.GetComponents() {
		installContext := tracker.getComponentInstallContext(comp.Name())
		if result := r.installSingleComponent(spiCtx, installContext, comp, preUpgrade); result.Requeue {
			requeue = true
		}

	}
	if requeue {
		return newRequeueWithDelay(), nil
	}

	// All components have been installed
	return ctrl.Result{}, nil
}

// installSingleComponent installs a single component
func (r *Reconciler) installSingleComponent(spiCtx spi.ComponentContext, installContext *componentInstallContext, comp spi.Component, preUpgrade bool) ctrl.Result {
	compName := comp.Name()
	compContext := spiCtx.Init(compName).Operation(vzconst.InstallOperation)
	compLog := compContext.Log()

	componentStatus, ok := spiCtx.ActualCR().Status.Components[comp.Name()]
	if !ok {
		compLog.Debugf("Did not find status details in map for component %s", comp.Name())
		installContext.state = compStateInstallEnd
	}

	for installContext.state != compStateInstallEnd {
		switch installContext.state {
		case compStateInstallInitDetermineComponentState:
			compLog.Debugf("Component %s is being reconciled", compName)

			if skipComponentFromDetermineComponentState(compContext, comp, preUpgrade) {
				installContext.state = compStateInstallEnd
				continue
			}

			// Determine the next state based on the component status state
			installContext.state = chooseCompState(componentStatus)

		case compStateInstallInitDisabled:
			if skipComponentFromDisabledState(compContext, comp, preUpgrade) {
				installContext.state = compStateInstallEnd
				continue
			}
			installContext.state = compStateWriteInstallStartedStatus

		case compStateInstallInitReady:
			if skipComponentFromReadyState(compContext, comp, componentStatus) {
				installContext.state = compStateInstallEnd
				continue
			}
			installContext.state = compStateWriteInstallStartedStatus

		case compStateWriteInstallStartedStatus:
			oldState := componentStatus.State
			oldGen := componentStatus.ReconcilingGeneration
			componentStatus.ReconcilingGeneration = 0
			if err := r.updateComponentStatus(compContext, "Install started", vzapi.CondInstallStarted); err != nil {
				compLog.ErrorfThrottled("Error writing component Installing state to the status: %v", err)
				return ctrl.Result{Requeue: true}
			}
			compLog.Oncef("CR.generation: %v reset component %s state: %v generation: %v to state: %v generation: %v ",
				spiCtx.ActualCR().Generation, compName, oldState, oldGen, componentStatus.State, componentStatus.ReconcilingGeneration)

			installContext.state = compStatePreInstall

		case compStatePreInstall:
			if !registry.ComponentDependenciesMet(comp, compContext) {
				compLog.Progressf("Component %s waiting for dependencies %v to be ready", comp.Name(), comp.GetDependencies())
				return ctrl.Result{Requeue: true}
			}
			compLog.Progressf("Component %s pre-install is running ", compName)
			if err := comp.PreInstall(compContext); err != nil {
				// Only log this error if this is not a RetryableError
				if _, ok := err.(ctrlerrors.RetryableError); !ok {
					compLog.ErrorfThrottled("Error running PreInstall for component %s: %v", compName, err)
				}

				return ctrl.Result{Requeue: true}
			}

			installContext.state = compStateInstall

		case compStateInstall:
			// If component is not installed,install it
			compLog.Oncef("Component %s install started ", compName)
			if err := comp.Install(compContext); err != nil {
				// Only log this error if this is not a RetryableError
				if _, ok := err.(ctrlerrors.RetryableError); !ok {
					compLog.ErrorfThrottled("Error running Install for component %s: %v", compName, err)
				}

				return ctrl.Result{Requeue: true}
			}

			installContext.state = compStateInstallWaitReady

		case compStateInstallWaitReady:
			if !comp.IsReady(compContext) {
				compLog.Progressf("Component %s has been installed. Waiting for the component to be ready", compName)
				return ctrl.Result{Requeue: true}
			}
			compLog.Oncef("Component %s successfully installed", comp.Name())

			installContext.state = compStatePostInstall

		case compStatePostInstall:
			compLog.Oncef("Component %s post-install running", compName)
			if err := comp.PostInstall(compContext); err != nil {
				// Only log this error if this is not a RetryableError
				if _, ok := err.(ctrlerrors.RetryableError); !ok {
					compLog.ErrorfThrottled("Error running PostInstall for component %s: %v", compName, err)
				}

				return ctrl.Result{Requeue: true}
			}

			installContext.state = compStateInstallComplete

		case compStateInstallComplete:
			if err := r.updateComponentStatus(compContext, "Install complete", vzapi.CondInstallComplete); err != nil {
				compLog.ErrorfThrottled("Error writing component Ready state to the status: %v", err)
				return ctrl.Result{Requeue: true}
			}

			installContext.state = compStateInstallEnd
		}

	}
	// Component has been installed
	return ctrl.Result{}
}

// checkConfigUpdated checks if the component config in the VZ CR has been updated
// back looking at the VZ Generation, component ReconcilingGeneration, and component LastReconciledGeneration values
func checkConfigUpdated(ctx spi.ComponentContext, componentStatus *vzapi.ComponentStatusDetails) bool {
	// The component is being reconciled/installed with ReconcilingGeneration of the CR
	// if CR.Generation > ReconcilingGeneration then re-enter install flow
	if componentStatus.ReconcilingGeneration > 0 {
		return ctx.ActualCR().Generation > componentStatus.ReconcilingGeneration
	}
	// The component has been reconciled/installed with LastReconciledGeneration of the CR
	// if CR.Generation > LastReconciledGeneration then re-enter install flow
	return (componentStatus.State == vzapi.CompStateReady) &&
		(ctx.ActualCR().Generation > componentStatus.LastReconciledGeneration)
}

// Check if the component can be installed in this Verrazzano installation based on version
// Components might require a specific a minimum version of Verrazzano > 1.0.0
func isVersionOk(log vzlog.VerrazzanoLogger, compVersion string, vzVersion string) bool {
	if len(vzVersion) == 0 {
		return true
	}
	vzSemver, err := semver.NewSemVersion(vzVersion)
	if err != nil {
		log.Errorf("Failed getting semver from status: %v", err)
		return false
	}
	compSemver, err := semver.NewSemVersion(compVersion)
	if err != nil {
		log.Errorf("Failed creating new semver for component: %v", err)
		return false
	}

	// return false if VZ version is too low to install component, else true
	return !vzSemver.IsLessThan(compSemver)
}

// getComponentInstallContext gets the install context for the component
func (vuc *installTracker) getComponentInstallContext(compName string) *componentInstallContext {
	context, ok := vuc.compMap[compName]
	if !ok {
		context = &componentInstallContext{
			state: compStateInstallInitDetermineComponentState,
		}
		vuc.compMap[compName] = context
	}
	return context
}

// chooseCompState choose the next componentInstallStatus based on the component status state
func chooseCompState(componentStatus *vzapi.ComponentStatusDetails) componentInstallState {
	switch componentStatus.State {
	case vzapi.CompStateDisabled:
		return compStateInstallInitDisabled
	case vzapi.CompStatePreInstalling:
		return compStateWriteInstallStartedStatus
	case vzapi.CompStateInstalling:
		return compStateWriteInstallStartedStatus
	default:
		return compStateInstallInitReady
	}
}

// skipComponentFromDetermineComponentState contains the logic about whether to go straight to the component terminal state from compStateInstallInitDetermineComponentState
func skipComponentFromDetermineComponentState(compContext spi.ComponentContext, comp spi.Component, preUpgrade bool) bool {
	if !comp.IsEnabled(compContext.EffectiveCR()) {
		compContext.Log().Oncef("Component %s is disabled, skipping install", comp.Name())
		// User has disabled component in Verrazzano CR, don't install
		return true
	}
	// Some components, like MySQL Operator, need to be installed before upgrade
	if preUpgrade && !comp.ShouldInstallBeforeUpgrade() {
		return true
	}

	return false
}

// skipComponentFromDisabledState contains the logic about whether to go straight to the component terminal state from compStateInstallInitDisabled
func skipComponentFromDisabledState(compContext spi.ComponentContext, comp spi.Component, preUpgrade bool) bool {
	// Only check for min VPO version if this is not the PreUpgrade case
	if !preUpgrade && !isVersionOk(compContext.Log(), comp.GetMinVerrazzanoVersion(), compContext.ActualCR().Status.Version) {
		// User needs to do upgrade before this component can be installed
		compContext.Log().Progressf("Component %s cannot be installed until Verrazzano is upgraded to at least version %s",
			comp.Name(), comp.GetMinVerrazzanoVersion())
		return true
	}
	return false
}

// skipComponentFromReadyState contains the logic about whether to go straight to the component terminal state from compStateInstallInitReady
func skipComponentFromReadyState(compContext spi.ComponentContext, comp spi.Component, componentStatus *vzapi.ComponentStatusDetails) bool {
	// Don't reconcile (updates) during install
	if !isInstalled(compContext.ActualCR().Status) {
		return true
	}
	// only run component install if the component generation does not match the CR generation
	if !checkConfigUpdated(compContext, componentStatus) {
		return true
	}
	if !comp.MonitorOverrides(compContext) {
		compContext.Log().Oncef("Skipping update for component %s, monitorChanges set to false", comp.Name())
		return true
	}
	return false
}
