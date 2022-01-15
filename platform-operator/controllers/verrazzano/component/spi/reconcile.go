// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package spi

import (
	"fmt"
	ctrlerrors "github.com/verrazzano/verrazzano/pkg/controller/errors"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzctrlcommon "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/common"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"time"
)

//Reconcile - common component lifecycle reconciler
func Reconcile(compContext ComponentContext, comp Component) error {
	compName := comp.Name()
	cr := compContext.ActualCR()
	compLog := compContext.Log()

	//compContext := ctx.Init(compName).Operation(vzconst.InstallOperation)
	compLog.Debugf("processing install for %s", compName)

	if !comp.IsOperatorInstallSupported() {
		compLog.Debugf("component based install not supported for %s", compName)
		return nil
	}

	componentStatus, statusFound := compContext.ActualCR().Status.Components[comp.Name()]
	if !statusFound {
		return fmt.Errorf("did not find status details in map for component %s", comp.Name())
	}

	switch componentStatus.State {
	case vzapi.Ready:
		// For delete, we should look at the VZ resource delete timestamp and shift into Quiescing/Uninstalling state
		compLog.Oncef("component %s is ready", compName)
		if implementsComponentInternal(comp) {
			var comp interface{} = comp
			if err := comp.(ComponentInternal).ReconcileSteadyState(compContext); err != nil {
				return err
			}
		}
		return nil
	case vzapi.Disabled:
		if !comp.IsEnabled(compContext) {
			compLog.Oncef("component %s is disabled, skipping install", compName)
			// User has disabled component in Verrazzano CR, don't install
			return nil
		}
		if !isVersionOk(compContext.Log(), comp.GetMinVerrazzanoVersion(), cr.Status.Version) {
			// User needs to do upgrade before this component can be installed
			compLog.Progressf("Component %s cannot be installed until Verrazzano is upgraded to at least version %s",
				comp.Name(), comp.GetMinVerrazzanoVersion())
			return nil
		}
		if err := updateComponentStatus(compContext, "PreInstall started", vzapi.PreInstall); err != nil {
			return err
		}
		return ctrlerrors.RetryableError{
			Source:    comp.Name(),
			Operation: compContext.GetOperation(),
		}
	case vzapi.PreInstalling:
		compLog.Debugf("PreInstalling component %s", comp.Name())
		// Can't do the dependency check here at present, introduces a cycle with the registry
		if !ComponentDependenciesMet(comp, compContext) {
			compLog.Progressf("Component %s waiting for dependencies %v to be ready", comp.Name(), comp.GetDependencies())
			return ctrlerrors.RetryableError{
				Source:    comp.Name(),
				Operation: compContext.GetOperation(),
				Result:    ctrl.Result{},
			}
		}
		compLog.Progressf("Component %s pre-install is running ", compName)
		if err := comp.PreInstall(compContext); err != nil {
			return err
		}
		// If component is not installed,install it
		compLog.Oncef("Component %s install started ", compName)
		if err := comp.Install(compContext); err != nil {
			return err
		}
		if err := updateComponentStatus(compContext, "Install started", vzapi.InstallStarted); err != nil {
			return err
		}
		// Install started requeue to check status
		return ctrlerrors.RetryableError{
			Source:    comp.Name(),
			Operation: compContext.GetOperation(),
		}
	case vzapi.Installing:
		// For delete, we should look at the VZ resource delete timestamp and shift into Quiescing/Uninstalling state
		// If component is enabled -- need to replicate scripts' config merging logic here
		// If component is in deployed state, continue
		if comp.IsReady(compContext) {
			compLog.Progressf("Component %s post-install is running ", compName)
			if err := comp.PostInstall(compContext); err != nil {
				return err
			}
			compLog.Oncef("Component %s successfully installed", compName)
			if err := updateComponentStatus(compContext, "Install complete", vzapi.InstallComplete); err != nil {
				return err
			}
			// Don't requeue because of this component, it is done install
			return nil
		}
		// Install of this component is not done, requeue to check status
		compLog.Progressf("Component %s waiting to finish installing", compName)
		return ctrlerrors.RetryableError{
			Source:    comp.Name(),
			Operation: compContext.GetOperation(),
		}
	}
	return nil
}

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

func implementsComponentInternal(i interface{}) bool {
	_, ok := i.(ComponentInternal)
	return ok
}

func updateComponentStatus(compContext ComponentContext, message string, conditionType vzapi.ConditionType) error {
	t := time.Now().UTC()
	condition := vzapi.Condition{
		Type:    conditionType,
		Status:  corev1.ConditionTrue,
		Message: message,
		LastTransitionTime: fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02dZ",
			t.Year(), t.Month(), t.Day(),
			t.Hour(), t.Minute(), t.Second()),
	}

	componentName := compContext.GetComponent()
	cr := compContext.ActualCR()
	log := compContext.Log()

	if cr.Status.Components == nil {
		cr.Status.Components = make(map[string]*vzapi.ComponentStatusDetails)
	}
	componentStatus := cr.Status.Components[componentName]
	if componentStatus == nil {
		componentStatus = &vzapi.ComponentStatusDetails{
			Name: componentName,
		}
		cr.Status.Components[componentName] = componentStatus
	}
	componentStatus.Conditions = appendConditionIfNecessary(log, componentStatus, condition)

	// Set the state of resource
	componentStatus.State = vzctrlcommon.CheckConditionType(conditionType)

	return nil
}

func appendConditionIfNecessary(log vzlog.VerrazzanoLogger, compStatus *vzapi.ComponentStatusDetails, newCondition vzapi.Condition) []vzapi.Condition {
	for _, existingCondition := range compStatus.Conditions {
		if existingCondition.Type == newCondition.Type {
			return compStatus.Conditions
		}
	}
	log.Debugf("Adding %s resource newCondition: %v", compStatus.Name, newCondition.Type)
	return append(compStatus.Conditions, newCondition)
}

// componentDependenciesMet Checks if the declared dependencies for the component are ready and available
func ComponentDependenciesMet(c Component, context ComponentContext) bool {
	log := context.Log()
	trace, err := CheckDependencies(c, context, make(map[string]bool), make(map[string]bool))
	if err != nil {
		log.Error(err.Error())
		return false
	}
	if len(trace) == 0 {
		log.Debugf("No dependencies declared for %s", c.Name())
		return true
	}
	log.Debugf("Trace results for %s: %v", c.Name(), trace)
	for _, value := range trace {
		if !value {
			return false
		}
	}
	return true
}

// checkDependencies Check the ready state of any dependencies and check for cycles
func CheckDependencies(c Component, context ComponentContext, visited map[string]bool, stateMap map[string]bool) (map[string]bool, error) {
	compName := c.Name()
	log := context.Log()
	log.Debugf("Checking %s dependencies", compName)
	if _, wasVisited := visited[compName]; wasVisited {
		return stateMap, context.Log().ErrorfNewErr("Failed, illegal state, dependency cycle found for %s", c.Name())
	}
	visited[compName] = true
	for _, dependencyName := range c.GetDependencies() {
		if compName == dependencyName {
			return stateMap, context.Log().ErrorfNewErr("Failed, illegal state, dependency cycle found for %s", c.Name())
		}
		if _, ok := stateMap[dependencyName]; ok {
			// dependency already checked
			log.Debugf("Dependency %s already checked", dependencyName)
			continue
		}
		found, dependency := context.GetComponentRegistry().FindComponent(dependencyName)
		if !found {
			return stateMap, context.Log().ErrorfNewErr("Failed, illegal state, declared dependency not found for %s: %s", c.Name(), dependencyName)
		}
		if trace, err := CheckDependencies(dependency, context, visited, stateMap); err != nil {
			return trace, err
		}
		if !dependency.IsReady(context) {
			stateMap[dependencyName] = false // dependency is not ready
			continue
		}
		stateMap[dependencyName] = true // dependency is ready
	}
	return stateMap, nil
}
