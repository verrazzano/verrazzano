// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/base/controllerspi"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
)

// initializedSet is needed to keep track of which Verrazzano CRs have been initialized
var initializedSet = make(map[string]bool)

// Set to true during unit testing
var unitTesting bool

// Reconcile reconciles the Module CR
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	actualCR := &vzapi.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error, don't requeue
		return result.NewResult()
	}

	// Wait for legacy verrazzano controller to initialize status
	if actualCR.Status.Components == nil {
		return result.NewResultShortRequeueDelay()
	}

	// Get effective CR and set the effective CR status with the actual status
	// Note that the reconciler code only udpdate the status, which is why the effective
	// CR is passed.  If was ever to update the spec, then the actual CR would need to be used.
	effectiveCR, err := transform.GetEffectiveCR(actualCR)
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	effectiveCR.Status = actualCR.Status

	log := vzlog.DefaultLogger()

	// VZ components can be installed, updated, upgraded, or uninstalled independently
	// Process all the components and only requeue are the end, so that operations
	// (like uninstall) are not blocked by a different component's failure
	res1 := r.createOrUpdateModules(log, effectiveCR)
	res2 := r.deleteModules(log, effectiveCR)

	// Requeue if any of the previous operations indicate a requeue is needed
	if res1.ShouldRequeue() || res2.ShouldRequeue() {
		return result.NewResultShortRequeueDelay()
	}

	// All the modules have been reconciled and are ready
	return result.NewResult()
}

// createOrUpdateModules creates or updates all the modules
func (r Reconciler) createOrUpdateModules(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano) result.Result {
	semver, err := validators.GetCurrentBomVersion()
	if err != nil {
		return result.NewResultShortRequeueDelayWithError(fmt.Errorf("Failed to get BOM version: %v", err))
	}

	version := semver.ToString()

	// Create or Update a Module for each enabled resource
	for _, comp := range registry.GetComponents() {
		createOrUpdate, err := r.shouldCreateOrUpdateModule(effectiveCR, comp)
		if err != nil {
			return result.NewResultShortRequeueDelayWithError(err)
		}
		if !createOrUpdate {
			continue
		}

		module := moduleapi.Module{
			ObjectMeta: metav1.ObjectMeta{
				Name:      comp.Name(),
				Namespace: constants.VerrazzanoInstallNamespace,
			},
		}
		_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, &module, func() error {
			// TODO For now have the module version match the VZ version
			return r.mutateModule(log, effectiveCR, &module, comp, version, version)
		})
		if err != nil {
			if !errors.IsConflict(err) {
				log.Errorf("Failed createOrUpdate module %s: %v", module.Name, err)
			}
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}
	return result.NewResult()
}

// mutateModule mutates the module for the create or update callback
func (r Reconciler) mutateModule(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano, module *moduleapi.Module, comp spi.Component, vzVersion string, moduleVersion string) error {
	if module.Annotations == nil {
		module.Annotations = make(map[string]string)
	}
	module.Annotations[constants.VerrazzanoCRNameAnnotation] = effectiveCR.Name
	module.Annotations[constants.VerrazzanoCRNamespaceAnnotation] = effectiveCR.Namespace
	module.Annotations[constants.VerrazzanoObservedGenerationAnnotation] = strconv.FormatInt(effectiveCR.Generation, 10)
	module.Annotations[constants.VerrazzanoVersionAnnotation] = vzVersion

	module.Spec.ModuleName = module.Name
	module.Spec.TargetNamespace = comp.Namespace()

	module.Spec.Version = moduleVersion
	if err := r.setModuleValues(log, effectiveCR, module, comp); err != nil {
		return err
	}
	return nil
}

// shouldCreateOrUpdateModule returns true if the Module should be created or updated
func (r Reconciler) shouldCreateOrUpdateModule(effectiveCR *vzapi.Verrazzano, comp spi.Component) (bool, error) {
	if !comp.ShouldUseModule() {
		return false, nil
	}
	if !comp.IsEnabled(effectiveCR) {
		return false, nil
	}

	// get the module
	module := &moduleapi.Module{}
	if err := r.Client.Get(context.TODO(), types.NamespacedName{Namespace: constants.VerrazzanoInstallNamespace, Name: comp.Name()}, module); err != nil {
		if errors.IsNotFound(err) {
			// module doesn't exist, need to create it
			return true, nil
		}
		return false, err
	}

	// if module doesn't have the current VZ generation then return true
	if module.Annotations != nil {
		gen := module.Annotations[constants.VerrazzanoObservedGenerationAnnotation]
		return gen != strconv.FormatInt(effectiveCR.Generation, 10), nil
	}

	return true, nil
}

// deleteModules deletes all the modules, optionally only deleting ones that disabled
func (r Reconciler) deleteModules(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano) result.Result {
	var reterr error
	var deletedCount int
	var moduleCount int

	// If deletion timestamp is non-zero then the VZ CR got deleted
	fullUninstall := !effectiveCR.GetDeletionTimestamp().IsZero()

	// Delete all modules.  Loop through all the components once even if error occurs.
	for _, comp := range registry.GetComponents() {
		if !comp.ShouldUseModule() {
			continue
		}
		moduleCount++

		// If not full uninstall then only disabled components should be uninstalled
		if !fullUninstall && comp.IsEnabled(effectiveCR) {
			continue
		}
		module := moduleapi.Module{ObjectMeta: metav1.ObjectMeta{
			Name:      comp.Name(),
			Namespace: constants.VerrazzanoInstallNamespace,
		}}
		err := r.Client.Delete(context.TODO(), &module, &client.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				deletedCount++
				continue
			}
			if !errors.IsConflict(err) {
				log.Progressf("Failed to delete Component %s, retrying: %v", comp.Name(), err)
			}
			reterr = err
			continue
		}
	}
	if deletedCount != moduleCount {
		return result.NewResultShortRequeueDelay()
	}

	// return last error found so that we retry
	if reterr != nil {
		return result.NewResultShortRequeueDelayWithError(reterr)
	}
	// All modules have been deleted and the Module CRs are gone
	return result.NewResult()
}
