// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"time"

	"github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	ctrl "sigs.k8s.io/controller-runtime"
)

// componentUninstallState identifies the state of a component during Uninstall
type componentUninstallState string

const (
	// compStateInit is the state when a component is starting the Uninstall flow
	compStateUninstallStart componentUninstallState = "compStateUninstallStart"

	// compStatePreUninstall is the state when a component does a pre-Uninstall
	compStatePreUninstall componentUninstallState = "compStatePreUninstall"

	// compStateUninstall is the state where a component does an Uninstall
	compStateUninstall componentUninstallState = "compStateUninstall"

	// compStateWaitUninstallCompleted is the state when a component is waiting for Uninstall to complete
	compStateWaitUninstallCompleted componentUninstallState = "compStateWaitUninstallCompleted"

	// compStatePostUninstall is the state when a component is doing a post-Uninstall
	compStatePostUninstall componentUninstallState = "compStatePostUninstall"

	// compStateUninstallDone is the state when component Uninstall is done
	compStateUninstallDone componentUninstallState = "compStateUninstallDone"

	// compStateUninstallEnd is the terminal state
	compStateUninstallEnd componentUninstallState = "compStateUninstallEnd"
)

// componentUninstallContext has the Uninstall context for a Verrazzano component Uninstall
type componentUninstallContext struct {
	state componentUninstallState
}

// UninstallComponents will Uninstall the components as required
func (r *Reconciler) UninstallComponents(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, tracker *UninstallTracker) (ctrl.Result, error) {
	spiCtx, err := spi.NewContext(log, r.Client, cr, r.DryRun)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	// Loop through all of the Verrazzano components and Uninstall each one.
	// Don't move to the next component until the current one has been succcessfully Uninstalld
	for _, comp := range registry.GetComponents() {
		UninstallContext := tracker.getComponentUninstallContext(comp.Name())
		result, err := r.UninstallSingleComponent(spiCtx, UninstallContext, comp)
		if err != nil || result.Requeue {
			return result, err
		}

	}
	// All components have been Uninstalld
	return ctrl.Result{}, nil
}

// UninstallSingleComponent Uninstalls a single component
func (r *Reconciler) UninstallSingleComponent(spiCtx spi.ComponentContext, UninstallContext *componentUninstallContext, comp spi.Component) (ctrl.Result, error) {
	compName := comp.Name()
	compContext := spiCtx.Init(compName).Operation(vzconst.UninstallOperation)
	compLog := compContext.Log()

	for UninstallContext.state != compStateEnd {
		switch UninstallContext.state {
		case compStateInit:
			// Check if component is installed, if not continue
			installed, err := comp.IsInstalled(compContext)
			if err != nil {
				compLog.Errorf("Failed checking if component %s is installed: %v", compName, err)
				return ctrl.Result{}, err
			}
			if installed {
				compLog.Oncef("Component %s is installed and will be Uninstalld", compName)
				UninstallContext.state = compStatePreUninstall
			} else {
				compLog.Oncef("Component %s is not installed; Uninstall being skipped", compName)
				UninstallContext.state = compStateEnd
			}

		case compStatePreUninstall:
			compLog.Oncef("Component %s pre-Uninstall running", compName)
			if err := comp.PreUninstall(compContext); err != nil {
				compLog.Errorf("Failed pre-upgrading component %s: %v", compName, err)
				return ctrl.Result{}, err
			}
			UninstallContext.state = compStateUninstall

		case compStateUninstall:
			compLog.Progressf("Component %s Uninstall running", compName)
			if err := comp.Uninstall(compContext); err != nil {
				compLog.Errorf("Failed upgrading component %s, will retry: %v", compName, err)
				// check to see whether this is due to a pending Uninstall
				r.resolvePendingUninstalls(compName, compLog)
				// requeue for 30 to 60 seconds later
				return controller.NewRequeueWithDelay(30, 60, time.Second), nil
			}
			UninstallContext.state = compStateWaitReady

		case compStateWaitReady:
			if !comp.IsReady(compContext) {
				compLog.Progressf("Component %s has been Uninstalld. Waiting for the component to be ready", compName)
				return newRequeueWithDelay(), nil
			}
			compLog.Progressf("Component %s is ready after being Uninstalld", compName)
			UninstallContext.state = compStatePostUninstall

		case compStatePostUninstall:
			compLog.Oncef("Component %s post-Uninstall running", compName)
			if err := comp.PostUninstall(compContext); err != nil {
				return ctrl.Result{}, err
			}
			UninstallContext.state = compStateUninstallDone

		case compStateUninstallDone:
			compLog.Oncef("Component %s has successfully Uninstalld", compName)
			UninstallContext.state = compStateEnd
		}
	}
	// Component has been Uninstalld
	return ctrl.Result{}, nil
}

// getComponentUninstallContext gets the Uninstall context for the component
func (vuc *UninstallTracker) getComponentUninstallContext(compName string) *componentUninstallContext {
	context, ok := vuc.compMap[compName]
	if !ok {
		context = &componentUninstallContext{
			state: compStateInit,
		}
		vuc.compMap[compName] = context
	}
	return context
}
