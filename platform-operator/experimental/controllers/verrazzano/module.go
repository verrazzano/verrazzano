// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	moduleCatalog "github.com/verrazzano/verrazzano/platform-operator/experimental/catalog"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// createOrUpdateModules creates or updates all the modules
func (r Reconciler) createOrUpdateModules(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	catalog, err := moduleCatalog.NewCatalog(config.GetCatalogPath())
	if err != nil {
		log.ErrorfThrottled("Error loading module catalog: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Create or Update a Module for each enabled resource
	for _, comp := range registry.GetComponents() {
		if !comp.IsEnabled(effectiveCR) {
			// If the component is not enabled then check if it is installed.
			// There is an edge case where a component might be disabled, but installed.
			// For example in VMO 1.5 -> 1.6 upgrade, VMO.IsEnabled used to return true if
			// Prometheus was enabled, but for 1.6 it returns false. So 1.6 VMO.IsEnabled might
			// return false, when VMO is really installed.  In that case, we need to create the
			// Module CR so that we can uninstall it (see deleteModules in reconciler.go).
			componentCtx, err := componentspi.NewContext(log, r.Client, actualCR, nil, r.DryRun)
			if err != nil {
				return result.NewResultShortRequeueDelayWithError(err)
			}
			installed, err := comp.IsInstalled(componentCtx)
			if err != nil {
				return result.NewResultShortRequeueDelayWithError(err)
			}
			if !installed {
				continue
			}
		}

		// Get the module version from the catalog
		version := catalog.GetVersion(comp.Name())
		if version == nil {
			err = log.ErrorfThrottledNewErr("Failed to find version for module %s in the module catalog", comp.Name())
			return result.NewResultShortRequeueDelayWithError(err)
		}
		if res := r.createOrUpdateOneModule(log, actualCR, effectiveCR, comp, version); res.ShouldRequeue() {
			return res
		}
	}
	return result.NewResult()
}

func (r Reconciler) createOrUpdateOneModule(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano, comp componentspi.Component, version *semver.SemVersion) result.Result {
	// Create or update the module
	module := moduleapi.Module{ObjectMeta: metav1.ObjectMeta{Name: comp.Name(), Namespace: vzconst.VerrazzanoInstallNamespace}}

	// There seems to be an issue with CreateOrUpdate() returning a false-updated status; if we compare the top-level
	// fields one-by-one they will be Equal if unchanged, but passing in the full Object for compare returns a diff.
	//
	// For now, stash the pre-update version of the Module away, then compare the specs using DeepEqual. That seems to
	// tell us if things have truly changed or not, at least the things we care about.
	//
	moduleExisting := &moduleapi.Module{}
	if err := r.Client.Get(context.TODO(), client.ObjectKeyFromObject(&module), moduleExisting); err != nil && !errors.IsNotFound(err) {
		return result.NewResultShortRequeueDelayWithError(err)
	}
	opResult, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &module, func() error {
		return r.mutateModule(log, actualCR, effectiveCR, &module, comp, version.ToString())
	})
	if err != nil {
		if !errors.IsConflict(err) {
			log.ErrorfThrottled("Failed createOrUpdate module %s: %v", module.Name, err)
		}
		return result.NewResultShortRequeueDelayWithError(err)
	}
	/*
		For debugging DeepEqual
		if !equality.Semantic.DeepEqual(moduleExisting, module) {
			log.Debugf("Full object diff for %s failed", client.ObjectKeyFromObject(&module))
		}
		if !equality.Semantic.DeepEqual(moduleExisting.TypeMeta, module.TypeMeta) {
			log.Debugf("Type metadata diff for %s failed", client.ObjectKeyFromObject(&module))
		}
		if !equality.Semantic.DeepEqual(moduleExisting.ObjectMeta, module.ObjectMeta) {
			log.Infof("Object metadata diff for %s failed", client.ObjectKeyFromObject(&module))
		}
		if !equality.Semantic.DeepEqual(moduleExisting.Status, module.Status) {
			log.Debugf("Status diff for %s failed", client.ObjectKeyFromObject(&module))
		}
	*/
	// Workaround, CreateOrUpdate is returning a false-positive update even when none of the fields change,
	// do a DeepEqual of the before/after module specs to see if anything there changed.
	if equality.Semantic.DeepEqual(moduleExisting.Spec, module.Spec) {
		opResult = controllerutil.OperationResultNone
	}
	// If the copy operation resulted in an update to the target, set the VZ condition to install started/Reconciling
	if res := r.updateStatusIfNeeded(log, actualCR, opResult); res.ShouldRequeue() {
		return res
	}
	return result.NewResult()
}

// mutateModule mutates the module for the create or update callback
func (r Reconciler) mutateModule(log vzlog.VerrazzanoLogger, actualCR *vzv1alpha1.Verrazzano, effectiveCR *vzv1alpha1.Verrazzano, module *moduleapi.Module, comp componentspi.Component, moduleVersion string) error {
	if module.Annotations == nil {
		module.Annotations = make(map[string]string)
	}
	module.Annotations[vzconst.VerrazzanoCRNameAnnotation] = effectiveCR.Name
	module.Annotations[vzconst.VerrazzanoCRNamespaceAnnotation] = effectiveCR.Namespace

	if module.Labels == nil {
		module.Labels = make(map[string]string)
	}
	module.Labels[vzconst.VerrazzanoOwnerLabel] = string(effectiveCR.UID)

	module.Spec.ModuleName = module.Name
	module.Spec.TargetNamespace = comp.Namespace()
	module.Spec.Version = moduleVersion

	// Set the module values and valuesFrom fields
	return r.setModuleValues(log, actualCR, effectiveCR, module, comp)
}

// deleteModules deletes all the modules, optionally only deleting ones that disabled
func (r Reconciler) deleteModules(log vzlog.VerrazzanoLogger, effectiveCR *vzv1alpha1.Verrazzano) result.Result {
	var reterr error
	var deletedCount int
	var moduleCount int

	// If deletion timestamp is non-zero then the VZ CR got deleted
	fullUninstall := !effectiveCR.GetDeletionTimestamp().IsZero()

	// Delete all modules.  Loop through all the modules once even if error occurs.
	for _, comp := range registry.GetComponents() {
		if !comp.ShouldUseModule() {
			continue
		}

		// If not full uninstall then only disabled components should be uninstalled
		if !fullUninstall && comp.IsEnabled(effectiveCR) {
			continue
		}

		// Check if the module exists before trying to delete the other related resources
		module := moduleapi.Module{}
		nsn := types.NamespacedName{Namespace: vzconst.VerrazzanoInstallNamespace, Name: comp.Name()}
		err := r.Client.Get(context.TODO(), nsn, &module, &client.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			log.Progressf("Failed to get Module %s, retrying: %v", comp.Name(), err)
			return result.NewResultShortRequeueDelayIfError(err)
		}

		moduleCount++

		// Delete all the configuration secrets that were referenced by the module
		res := r.deleteConfigSecrets(log, module.Namespace, comp.Name())
		if res.ShouldRequeue() {
			return res
		}

		// Delete all the configuration configmaps that were referenced by the module
		res = r.deleteConfigMaps(log, module.Namespace, comp.Name())
		if res.ShouldRequeue() {
			return res
		}

		// Delete the Module
		err = r.Client.Delete(context.TODO(), &module, &client.DeleteOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				deletedCount++
				continue
			}
			if !errors.IsConflict(err) {
				log.Progressf("Failed to delete Module %s, retrying: %v", comp.Name(), err)
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
