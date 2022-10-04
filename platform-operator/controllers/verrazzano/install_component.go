// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	ctrl "sigs.k8s.io/controller-runtime"
)

// componentInstallState identifies the state of a component during install
type componentInstallState string

const (
	// compStateInit is the state when a component is initialized
	compStateInstallInit componentInstallState = "componentStateInit"

	// compStateReady is the state when a component is ready
	compStateInstallReady componentInstallState = "componentStateReady"

	// compStateDisabled is the state when a component is disabled
	compStateInstallDisabled componentInstallState = "componentStateDisabled"

	// compStateInstallStarted is the state when a component writes the Install Started Condition
	compStateInstallStarted componentInstallState = "componentStateInstallStarted"

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

	globalInstallStart componentInstallState = "globalInstallStart"
)

// componentInstallContext has the install context for a Verrazzano component install
type componentInstallContext struct {
	state componentInstallState
}

// installComponents will install the components as required
func (r *Reconciler) installComponents(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, preUpgrade bool) (ctrl.Result, error) {
	spiCtx, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	spiCtx.Log().Progress("Reconciling components for Verrazzano installation")

	tracker := getInstallTracker(cr)

	var requeue bool

	// Loop through all of the Verrazzano components and install each one.
	// Don't move to the next component until the current one has been succcessfully installed
	for _, comp := range registry.GetComponents() {
		installContext := tracker.getComponentInstallContext(comp.Name())
		result, err := r.installSingleComponent(spiCtx, installContext, comp, preUpgrade)
		if err != nil || result.Requeue {
			requeue = true
		}

	}
	if requeue {
		return newRequeueWithDelay(), nil
	}

	deleteInstallTracker(cr)

	// All components have been installed
	return ctrl.Result{}, nil
}

// installSingleComponent installs a single component
func (r *Reconciler) installSingleComponent(spiCtx spi.ComponentContext, installContext *componentInstallContext, comp spi.Component, preUpgrade bool) (ctrl.Result, error) {
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
		case compStateInstallInit:

			compLog.Debugf("Component %s is being reconciled", compName)

			if !comp.IsOperatorInstallSupported() {
				compLog.Debugf("Component based install not supported for %s", compName)
				installContext.state = compStateInstallEnd
				continue
			}

			// Some components, like MySQL Operator, need to be installed before upgrade
			if preUpgrade && !comp.ShouldInstallBeforeUpgrade() {
				installContext.state = compStateInstallEnd
				continue
			}

			switch componentStatus.State {
			case vzapi.CompStateDisabled:
				installContext.state = compStateInstallDisabled
			case vzapi.CompStatePreInstalling:
				installContext.state = compStateInstallStarted
			case vzapi.CompStateInstalling:
				installContext.state = compStateInstallStarted
			default:
				installContext.state = compStateInstallReady
			}
		case compStateInstallDisabled:
			if !comp.IsEnabled(compContext.EffectiveCR()) {
				compLog.Oncef("Component %s is disabled, skipping install", compName)
				// User has disabled component in Verrazzano CR, don't install
				installContext.state = compStateInstallEnd
				continue
			}
			// Only check for min VPO version if this is not the preupgrade case
			if !preUpgrade && !isVersionOk(compLog, comp.GetMinVerrazzanoVersion(), spiCtx.ActualCR().Status.Version) {
				// User needs to do upgrade before this component can be installed
				compLog.Progressf("Component %s cannot be installed until Verrazzano is upgraded to at least version %s",
					comp.Name(), comp.GetMinVerrazzanoVersion())
				installContext.state = compStateInstallEnd
				continue
			}
			if spiCtx.ActualCR().Status.State == vzapi.VzStateReady {
				// This is the case where the component was previously disabled but is now enabled in the effective CR, so
				// we need to prevent the component from being installed when the VPO is upgraded and wait for the user
				// to initiate the upgrade via the VZ CR
				compLog.Oncef("Component %s was previously disabled and upgrade is not in progress, skipping install", compName)
				installContext.state = compStateInstallEnd
				continue
			}
			installContext.state = compStateInstallStarted

		case compStateInstallReady:
			// Don't reconcile (updates) during install
			if !isInstalled(spiCtx.ActualCR().Status) {
				installContext.state = compStateInstallEnd
				continue
			}

			if checkConfigUpdated(spiCtx, componentStatus, compName) && comp.IsEnabled(compContext.EffectiveCR()) {
				if !comp.MonitorOverrides(compContext) && comp.IsEnabled(spiCtx.EffectiveCR()) {
					compLog.Oncef("Skipping update for component %s, monitorChanges set to false", comp.Name())
				} else {
					if spiCtx.ActualCR().Status.State == vzapi.VzStateReady {
						installContext.state = globalInstallStart
						return newRequeueWithDelay(), nil
					}
					installContext.state = compStateInstallStarted
					continue
				}
			}
			installContext.state = compStateInstallEnd

		case compStateInstallStarted:
			oldState := componentStatus.State
			oldGen := componentStatus.ReconcilingGeneration
			componentStatus.ReconcilingGeneration = 0
			if err := r.updateComponentStatus(compContext, "Install started", vzapi.CondInstallStarted); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			compLog.Oncef("CR.generation: %v reset component %s state: %v generation: %v to state: %v generation: %v ",
				spiCtx.ActualCR().Generation, compName, oldState, oldGen, componentStatus.State, componentStatus.ReconcilingGeneration)
			installContext.state = compStatePreInstall

		case compStatePreInstall:
			if !registry.ComponentDependenciesMet(comp, compContext) {
				compLog.Progressf("Component %s waiting for dependencies %v to be ready", comp.Name(), comp.GetDependencies())
				return newRequeueWithDelay(), nil
			}
			compLog.Progressf("Component %s pre-install is running ", compName)
			if err := comp.PreInstall(compContext); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			installContext.state = compStateInstall

		case compStateInstall:
			// If component is not installed,install it
			compLog.Oncef("Component %s install started ", compName)
			if err := comp.Install(compContext); err != nil {
				return ctrl.Result{Requeue: true}, nil
			}
			installContext.state = compStateInstallWaitReady

		case compStateInstallWaitReady:
			if !comp.IsReady(compContext) {
				compLog.Progressf("Component %s has been installed. Waiting for the component to be ready", compName)
				return newRequeueWithDelay(), nil
			}
			compLog.Oncef("Component %s successfully installed", comp.Name())
			installContext.state = compStatePostInstall

		case compStatePostInstall:
			compLog.Oncef("Component %s post-install running", compName)
			if err := comp.PostInstall(compContext); err != nil {
				return ctrl.Result{}, err
			}
			installContext.state = compStateInstallComplete

		case compStateInstallComplete:
			if err := r.updateComponentStatus(compContext, "Install complete", vzapi.CondInstallComplete); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			installContext.state = compStateInstallEnd

		case globalInstallStart:
			err := r.setInstallingState(spiCtx.Log(), spiCtx.ActualCR())
			compLog.Oncef("Reset Verrazzano state to %v for generation %v", spiCtx.ActualCR().Status.State, spiCtx.ActualCR().Generation)
			if err != nil {
				spiCtx.Log().Errorf("Failed to reset state: %v", err)
				return newRequeueWithDelay(), err
			}
			deleteInstallTracker(spiCtx.ActualCR())
			return ctrl.Result{Requeue: true}, nil

		}

	}
	// Component has been installed
	return ctrl.Result{}, nil
}

// getComponentInstallContext gets the install context for the component
func (vuc *installTracker) getComponentInstallContext(compName string) *componentInstallContext {
	context, ok := vuc.compMap[compName]
	if !ok {
		context = &componentInstallContext{
			state: compStateInstallInit,
		}
		vuc.compMap[compName] = context
	}
	return context
}

//enabled := comp.IsEnabled(compContext.EffectiveCR())
//operatorInstall := comp.IsOperatorInstallSupported()
//installBeforeUpgrade := comp.ShouldInstallBeforeUpgrade()
//monitorOverrides := comp.MonitorOverrides(compContext)
//watched := r.IsWatchedComponent(comp.GetJSONName())
//versionOK := isVersionOk(compLog, comp.GetMinVerrazzanoVersion(), spiCtx.ActualCR().Status.Version)
//configUpdated := checkConfigUpdated(spiCtx, componentStatus,  compName)
//compReady := componentStatus.State == vzapi.CompStateReady
//compDisabled := componentStatus.State == vzapi.CompStateDisabled
