// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"time"

	"github.com/verrazzano/verrazzano/pkg/controller"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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

	// compStateWaitUninstalled is the state when a component is waiting to be uninstalled
	compStateWaitUninstalled componentUninstallState = "compStateWaitUninstalled"

	// compStateUninstalledone is the state when component Uninstall is done
	compStateUninstalledone componentUninstallState = "compStateUninstalledone"

	// compStateUninstallEnd is the terminal state
	compStateUninstallEnd componentUninstallState = "compStateUninstallEnd"
)

// componentUninstallContext has the Uninstall context for a Verrazzano component Uninstall
type componentUninstallContext struct {
	state componentUninstallState
}

// UninstallComponents will Uninstall the components as required
func (r *Reconciler) uninstallComponents(log vzlog.VerrazzanoLogger, cr *installv1alpha1.Verrazzano, tracker *UninstallTracker) (ctrl.Result, error) {
	spiCtx, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	// Loop through all of the Verrazzano components and Uninstall each one.
	// Don't move to the next component until the current one has been succcessfully Uninstalled
	for _, comp := range registry.GetComponents() {
		UninstallContext := tracker.getComponentUninstallContext(comp.Name())
		result, err := r.uninstallSingleComponent(spiCtx, UninstallContext, comp)
		if err != nil || result.Requeue {
			return result, err
		}

	}
	// All components have been Uninstalled
	return ctrl.Result{}, nil
}

// UninstallSingleComponent Uninstalls a single component
func (r *Reconciler) uninstallSingleComponent(spiCtx spi.ComponentContext, UninstallContext *componentUninstallContext, comp spi.Component) (ctrl.Result, error) {
	compName := comp.Name()
	compContext := spiCtx.Init(compName).Operation(vzconst.UninstallOperation)
	compLog := compContext.Log()

	for UninstallContext.state != compStateUninstallEnd {
		switch UninstallContext.state {
		case compStateUninstallStart:
			// Check if operator based uninstall is supported
			if !comp.IsOperatorUninstallSupported() {
				UninstallContext.state = compStateUninstallEnd
				continue
			}
			// Check if component is installed, if not continue
			installed, err := comp.IsInstalled(compContext)
			if err != nil {
				compLog.Errorf("Failed checking if component %s is installed: %v", compName, err)
				return ctrl.Result{}, err
			}
			if !installed {
				compLog.Oncef("Component %s is not installed, nothing to do for uninstall", compName)
				UninstallContext.state = compStateUninstallEnd
				continue
			}
			if err := r.updateComponentStatus(compContext, "Uninstall started", vzapi.CondUninstallStarted); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			compLog.Oncef("Component %s is starting to uninstall", compName)
			UninstallContext.state = compStatePreUninstall

		case compStatePreUninstall:
			compLog.Oncef("Component %s is calling pre-uninstall", compName)
			if err := comp.PreUninstall(compContext); err != nil {
				compLog.Errorf("Failed pre-uninstalling component %s: %v", compName, err)
				return ctrl.Result{}, err
			}
			UninstallContext.state = compStateUninstall

		case compStateUninstall:
			compLog.Progressf("Component %s is calling uninstall", compName)
			if err := comp.Uninstall(compContext); err != nil {
				compLog.Errorf("Failed uninstalling component %s, will retry: %v", compName, err)
				// requeue for 30 to 60 seconds later
				return controller.NewRequeueWithDelay(30, 60, time.Second), nil
			}
			UninstallContext.state = compStateWaitUninstalled

		case compStateWaitUninstalled:
			installed, err := comp.IsInstalled(compContext)
			if err != nil {
				compLog.Errorf("Failed checking if component %s is installed: %v", compName, err)
				return controller.NewRequeueWithDelay(10, 15, time.Second), nil
			}
			if installed {
				compLog.Progressf("Waiting for component %s to be uninstalled", compName)
				return newRequeueWithDelay(), nil
			}
			compLog.Progressf("Component %s has been uninstalled, running post-uninstall", compName)
			if err := comp.PostUninstall(compContext); err != nil {
				compLog.Errorf("PostUninstall for component %s failed: %v", compName, err)
				// requeue for 10 to 15 seconds later
				return controller.NewRequeueWithDelay(10, 15, time.Second), nil
			}
			UninstallContext.state = compStateUninstalledone

		case compStateUninstalledone:
			if err := r.updateComponentStatus(compContext, "Uninstall complete", vzapi.CondUninstallComplete); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			compLog.Oncef("Component %s has successfully uninstalled", compName)
			UninstallContext.state = compStateUninstallEnd
		}
	}
	// Component has been Uninstalled
	return ctrl.Result{}, nil
}

// getComponentUninstallContext gets the Uninstall context for the component
func (vuc *UninstallTracker) getComponentUninstallContext(compName string) *componentUninstallContext {
	context, ok := vuc.compMap[compName]
	if !ok {
		context = &componentUninstallContext{
			state: compStateUninstallStart,
		}
		vuc.compMap[compName] = context
	}
	return context
}
