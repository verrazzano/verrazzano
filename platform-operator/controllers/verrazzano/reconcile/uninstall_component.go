// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package reconcile

import (
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/rancher"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	ctrl "sigs.k8s.io/controller-runtime"
)

// componentUninstallState identifies the state of a component during Uninstall
type componentUninstallState string

const (
	// compStateUpgradeStart is the state when a component is starting the Uninstall flow
	compStateUninstallStart componentUninstallState = "compStateUninstallStart"

	// compStatePreUninstall is the state when a component does a pre-Uninstall
	compStatePreUninstall componentUninstallState = "compStatePreUninstall"

	// compStateUninstall is the state where a component does an Uninstall
	compStateUninstall componentUninstallState = "compStateUninstall"

	// compStateWaitUninstalled is the state when a component is waiting to be uninstalled
	compStateWaitUninstalled componentUninstallState = "compStateWaitUninstalled"

	// compStateUninstalleDone is the state when component Uninstall is done
	compStateUninstalleDone componentUninstallState = "compStateUninstalleDone"

	// compStateUninstallEnd is the terminal state
	compStateUninstallEnd componentUninstallState = "compStateUninstallEnd"
)

// UninstallComponents will Uninstall the components as required
func (r *Reconciler) uninstallComponents(log vzlog.VerrazzanoLogger, cr *v1alpha1.Verrazzano, tracker *UninstallTracker) (ctrl.Result, error) {
	spiCtx, err := spi.NewContext(log, r.Client, cr, nil, r.DryRun)
	if err != nil {
		return newRequeueWithDelay(), err
	}

	var requeue bool

	// Loop through the Verrazzano components in uninstall order, and Uninstall each one.
	// Don't block uninstalling the next component if the current one has an error.
	// It is normal for a component to return an error if it is waiting for some condition.
	for _, comp := range registry.GetComponents() {
		if comp.ShouldUseModule() {
			// Requeue until the module is gone
			if !IsModuleUninstallDone() {
				requeue = true
			}
			// Ignore if this component is being handled by a Module
			continue
		}

		uninstallContext := tracker.getComponentUninstallContext(comp.Name())
		result, err := r.uninstallSingleComponent(spiCtx, uninstallContext, comp)
		if err != nil || result.Requeue {
			requeue = true
		}
	}
	if requeue {
		return newRequeueWithDelay(), nil
	}

	// All components have been Uninstalled
	return ctrl.Result{}, nil
}

// UninstallSingleComponent Uninstalls a single component
func (r *Reconciler) uninstallSingleComponent(spiCtx spi.ComponentContext, UninstallContext *componentTrackerContext, comp spi.Component) (ctrl.Result, error) {
	compName := comp.Name()
	compContext := spiCtx.Init(compName).Operation(vzconst.UninstallOperation)
	compLog := compContext.Log()
	rancherProvisioned, err := rancher.IsClusterProvisionedByRancher()
	if err != nil {
		return ctrl.Result{}, err
	}

	for UninstallContext.uninstallState != compStateUninstallEnd {
		switch UninstallContext.uninstallState {
		case compStateUninstallStart:
			// Check if operator based uninstall is supported
			if !comp.IsOperatorUninstallSupported() {
				UninstallContext.uninstallState = compStateUninstallEnd
				continue
			}
			if comp.Name() == rancher.ComponentName && rancherProvisioned {
				compLog.Oncef("Cluster was provisioned by Rancher. Component %s will not be uninstalled.", rancher.ComponentName)
				UninstallContext.uninstallState = compStateUninstallEnd
				continue
			}
			// Check if component is ex, if not continue
			exists, err := comp.Exists(compContext)
			//exists, err := comp.IsInstalled(compContext)
			if err != nil {
				compLog.Errorf("Failed checking if component %s exists in the cluster: %v", compName, err)
				return ctrl.Result{}, err
			}
			if !exists {
				compLog.Debugf("Component %s does not exist in cluster, nothing to do for uninstall", compName)
				UninstallContext.uninstallState = compStateUninstallEnd
				continue
			}
			if err := r.updateComponentStatus(compContext, "Uninstall started", v1alpha1.CondUninstallStarted); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			compLog.Oncef("Component %s is starting to uninstall", compName)
			UninstallContext.uninstallState = compStatePreUninstall

		case compStatePreUninstall:
			compLog.Oncef("Component %s is calling pre-uninstall", compName)
			if err := comp.PreUninstall(compContext); err != nil {
				// Components will log errors, could be waiting for condition
				return ctrl.Result{}, err
			}
			UninstallContext.uninstallState = compStateUninstall

		case compStateUninstall:
			compLog.Oncef("Component %s is calling uninstall", compName)
			if err := comp.Uninstall(compContext); err != nil {
				if !ctrlerrors.IsRetryableError(err) {
					compLog.Errorf("Failed uninstalling component %s, will retry: %v", compName, err)
				}
				return ctrl.Result{}, err
			}
			UninstallContext.uninstallState = compStateWaitUninstalled

		case compStateWaitUninstalled:
			installed, err := comp.IsInstalled(compContext)
			if err != nil {
				compLog.Errorf("Failed checking if component %s is installed: %v", compName, err)
				return newRequeueWithDelay(), nil
			}
			if installed {
				compLog.Oncef("Waiting for component %s to be uninstalled", compName)
				return newRequeueWithDelay(), nil
			}
			compLog.Oncef("Component %s has been uninstalled, running post-uninstall", compName)
			if err := comp.PostUninstall(compContext); err != nil {
				if !ctrlerrors.IsRetryableError(err) {
					compLog.Errorf("PostUninstall for component %s failed: %v", compName, err)
				}
				return newRequeueWithDelay(), nil
			}
			UninstallContext.uninstallState = compStateUninstalleDone

		case compStateUninstalleDone:
			if err := r.updateComponentStatus(compContext, "Uninstall complete", v1alpha1.CondUninstallComplete); err != nil {
				return ctrl.Result{Requeue: true}, err
			}
			compLog.Oncef("Component %s has successfully uninstalled", compName)
			UninstallContext.uninstallState = compStateUninstallEnd
		}
	}
	// Component has been Uninstalled
	return ctrl.Result{}, nil
}

// getComponentUninstallContext gets the Uninstall context for the component
func (vuc *UninstallTracker) getComponentUninstallContext(compName string) *componentTrackerContext {
	context, ok := vuc.compMap[compName]
	if !ok {
		context = &componentTrackerContext{
			uninstallState: compStateUninstallStart,
		}
		vuc.compMap[compName] = context
	}
	return context
}
