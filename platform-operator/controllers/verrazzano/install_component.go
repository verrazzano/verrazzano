// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)


// componentInstallState identifies the state of a component during install
type componentInstallState string

const (
	// compStateInit is the state when a component is initialized
	compStateInit componentInstallState = "componentStateInit"

	// compStateInstallStart is the state when a component is starting the install flow
	compStateInstallStart componentInstallState = "componentStateInstallStart"

	// compStatePreInstall is the state when a component does a pre-install
	compStatePreInstall componentInstallState = "compStatePreInstall"

	// compStateInstall is the state where a component does an install
	compStateInstall componentInstallState = "compStateInstall"

	// compStateInstallWaitReady is the state when a component is waiting for install ready
	compStateInstallWaitReady componentInstallState = "compStateInstallWaitReady"

	// compStatePostInstall is the state when a component is doing a post-install
	compStatePostInstall componentInstallState = "compStatePostInstall"

	// compStateInstalleDone is the state when component install is done
	compStateInstalleDone componentInstallState = "compStateInstalleDone"

	// compStateInstallEnd is the terminal state
	compStateInstallEnd componentInstallState = "compStateEnd"
)

// componentInstallContext has the install context for a Verrazzano component install
type componentInstallContext struct {
	state componentInstallState
}

// installComponents will install the components as required
func (r *Reconciler) installComponents(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, tracker *installTracker, preUpgrade bool) (ctrl.Result, error) {
	spiCtx, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
	if err != nil {

		return newRequeueWithDelay(), err
	}

	spiCtx.Log().Progress("Reconciling components for Verrazzano installation")


	// Loop through all of the Verrazzano components and install each one.
	// Don't move to the next component until the current one has been succcessfully installed
	for _, comp := range registry.GetComponents() {
		installContext := tracker.getComponentInstallContext(comp.Name())
		result, err := r.installSingleComponent(spiCtx, installContext, comp, preUpgrade)
		if err != nil || result.Requeue {
			return result, err
		}

	}
	// All components have been installed
	return ctrl.Result{}, nil
}


// installSingleComponent installs a single component
func (r *Reconciler) installSingleComponent(spiCtx spi.ComponentContext, installContext *componentInstallContext, comp spi.Component, preUpgrade bool) (ctrl.Result, error) {
	compName := comp.Name()
	compContext := spiCtx.Init(compName).Operation(vzconst.InstallOperation)
	compLog := compContext.Log()

	for installContext.state != compStateInstallEnd {
		vzState := spiCtx.ActualCR().Status.State
		componentStatus, ok := spiCtx.ActualCR().Status.Components[comp.Name()]
		if !ok {
			compLog.Debugf("Did not find status details in map for component %s", comp.Name())
			installContext.state = compStateInstallEnd
		}

		switch installContext.state {
		case compStateInit:
			compLog.Debugf("Component %s is being reconciled", compName)

			installPermitted := comp.IsEnabled(compContext.EffectiveCR()) comp.IsOperatorInstallSupported()
			operatorInstall := comp.IsOperatorInstallSupported()
			installBeforeUpgrade := comp.ShouldInstallBeforeUpgrade()
			monitorOverrides := comp.MonitorOverrides(compContext)
			watched := r.IsWatchedComponent(comp.GetJSONName())
			versionOK := isVersionOk(compLog, comp.GetMinVerrazzanoVersion(), spiCtx.ActualCR().Status.Version)
			configUpdated := checkConfigUpdated(spiCtx, componentStatus,  compName)
			compReady := componentStatus.State == vzapi.CompStateReady
			compDisabled := componentStatus.State == vzapi.CompStateDisabled

			//Skip component conditions

			// SKIP - User has disabled component in Verrazzano CR
			if !enabled {
				compLog.Oncef("Component %s is disabled, skipping install", compName)
				installContext.state = compStateInstallEnd
			}

			// SKIP - Operator install is not supported
			if !operatorInstall {
				compLog.Debugf("Component based install not supported for %s", compName)
				installContext.state = compStateInstallEnd
			}



			// SKIP - Not PreUpgrade and Version is not minimum version
			if !preUpgrade && !versionOK {
				// User needs to do upgrade before this component can be installed
				compLog.Progressf("Component %s cannot be installed until Verrazzano is upgraded to at least version %s",
					comp.Name(), comp.GetMinVerrazzanoVersion())
				installContext.state = compStateInstallEnd
			}


			// SKIP - Some components, like MySQL Operator, need to be installed before upgrade
			if preUpgrade && !comp.ShouldInstallBeforeUpgrade() {
				installContext.state = compStateInstallEnd
			}

			componentStatus, ok := spiCtx.ActualCR().Status.Components[comp.Name()]
			if !ok {
				compLog.Debugf("Did not find status details in map for component %s", comp.Name())
				installContext.state = compStateInstallEnd
			}

			if checkConfigUpdated(spiCtx, componentStatus,  compName) && comp.IsEnabled(compContext.EffectiveCR()) {
				if !comp.MonitorOverrides(compContext) && comp.IsEnabled(spiCtx.EffectiveCR()) {
					compLog.Oncef("Skipping update for component %s, monitorChanges set to false", comp.Name())
					installContext.state = compStateInstallEnd
				} else {
					oldState := componentStatus.State
					oldGen := componentStatus.ReconcilingGeneration
					componentStatus.ReconcilingGeneration = 0

					if err := r.updateComponentStatus(compContext, "Install Started", vzapi.CondInstallStarted); err != nil {
						return ctrl.Result{Requeue: true}, err
					}
					compLog.Oncef("CR.generation: %v reset component %s state: %v generation: %v to state: %v generation: %v ",
						spiCtx.ActualCR().Generation, compName, oldState, oldGen, componentStatus.State, componentStatus.ReconcilingGeneration)
					if spiCtx.ActualCR().Status.State == vzapi.VzStateReady {
						err := r.setInstallingState(spiCtx.Log(), spiCtx.ActualCR())
						compLog.Oncef("Reset Verrazzano state to %v for generation %v", spiCtx.ActualCR().Status.State, spiCtx.ActualCR().Generation)
						if err != nil {
							spiCtx.Log().Errorf("Failed to reset state: %v", err)
							return newRequeueWithDelay(), err
						}
					}
				}
			}

		case compStateInstallStart:
			compLog.Debugf("Component %s is being reconciled", compName)


		case compStatePreInstall:
			compLog.Oncef("Component %s pre-install running", compName)
			if err := comp.PreInstall(compContext); err != nil {
				compLog.Errorf("Failed pre-upgrading component %s: %v", compName, err)
				return ctrl.Result{}, err
			}
			installContext.state = compStateInstall

		case compStateInstall:
			compLog.Progressf("Component %s install running", compName)
			if err := comp.Install(compContext); err != nil {
				compLog.Errorf("Failed upgrading component %s, will retry: %v", compName, err)
				// check to see whether this is due to a pending install
				r.resolvePendingInstalls(compName, compLog)
				// requeue for 30 to 60 seconds later
				return controller.NewRequeueWithDelay(30, 60, time.Second), nil
			}
			installContext.state = compStateInstallWaitReady

		case compStateInstallWaitReady:
			if !comp.IsReady(compContext) {
				compLog.Progressf("Component %s has been installed. Waiting for the component to be ready", compName)
				return newRequeueWithDelay(), nil
			}
			compLog.Progressf("Component %s is ready after being installed", compName)
			installContext.state = compStatePostInstall

		case compStatePostInstall:
			compLog.Oncef("Component %s post-install running", compName)
			if err := comp.PostInstall(compContext); err != nil {
				return ctrl.Result{}, err
			}
			installContext.state = compStateInstalleDone

		case compStateInstalleDone:
			compLog.Oncef("Component %s has successfully installed", compName)
			if err := r.updateComponentStatus(compContext, "Install complete", installv1alpha1.CondInstallComplete); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			installContext.state = compStateInstallEnd
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
			state: compStateInit,
		}
		vuc.compMap[compName] = context
	}
	return context
}


