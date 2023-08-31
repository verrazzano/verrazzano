// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/semver"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/validators"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	componentspi "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	moduleCatalog "github.com/verrazzano/verrazzano/platform-operator/experimental/catalog"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// Reconcile reconciles the Verrazzano CR
func (r Reconciler) Reconcile(spictx controllerspi.ReconcileContext, u *unstructured.Unstructured) result.Result {
	actualCR := &vzapi.Verrazzano{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, actualCR); err != nil {
		spictx.Log.ErrorfThrottled(err.Error())
		// This is a fatal error which should never happen, don't requeue
		return result.NewResult()
	}

	// Wait for legacy verrazzano controller to initialize status
	if actualCR.Status.Components == nil {
		return result.NewResultShortRequeueDelay()
	}

	// If an upgrade is pending, do not reconcile; an upgrade is pending if the VPO has been upgraded, but the user
	// has not started an upgrade of the Verrazzano install.
	if upgradePending, err := r.isUpgradeRequired(actualCR); upgradePending || err != nil {
		// return an error if encountered, otherwise returns an empty result to stop reconciling
		spictx.Log.Progressf("Upgrade required before reconciling modules")
		return result.NewResultShortRequeueDelayIfError(err)
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
	catalog, err := moduleCatalog.NewCatalog(config.GetCatalogPath())
	if err != nil {
		log.ErrorfThrottled("Error loading module catalog: %v", err)
		return result.NewResultShortRequeueDelayWithError(err)
	}

	// Create or Update a Module for each enabled resource
	for _, comp := range registry.GetComponents() {
		if !comp.IsEnabled(effectiveCR) {
			continue
		}
		if !comp.ShouldUseModule() {
			continue
		}

		version := catalog.GetVersion(comp.Name())
		if version == nil {
			err = log.ErrorfThrottledNewErr("Failed to find version for module %s in the module catalog", comp.Name())
			return result.NewResultShortRequeueDelayWithError(err)
		}

		module := moduleapi.Module{
			ObjectMeta: metav1.ObjectMeta{
				Name:      comp.Name(),
				Namespace: vzconst.VerrazzanoInstallNamespace,
			},
		}
		opResult, err := controllerutil.CreateOrUpdate(context.TODO(), r.Client, &module, func() error {
			return r.mutateModule(log, effectiveCR, &module, comp, version.ToString())
		})
		log.Debugf("Module %s update result: %v", module.Name, opResult)
		if err != nil {
			if !errors.IsConflict(err) {
				log.ErrorfThrottled("Failed createOrUpdate module %s: %v", module.Name, err)
			}
			return result.NewResultShortRequeueDelayWithError(err)
		}
	}
	return result.NewResult()
}

// mutateModule mutates the module for the create or update callback
func (r Reconciler) mutateModule(log vzlog.VerrazzanoLogger, effectiveCR *vzapi.Verrazzano, module *moduleapi.Module, comp componentspi.Component, moduleVersion string) error {
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

	return r.setModuleValues(log, effectiveCR, module, comp)
}

// isUpgradeRequired Returns true if we detect that an upgrade is required but not (at least) in progress:
//   - if the Spec version IS NOT empty is less than the BOM version, an upgrade is required
//   - if the Spec version IS empty the Status version is less than the BOM, then an upgrade is required (upgrade of initial install scenario)
//
// If we return true here, it means we should stop reconciling until an upgrade has been requested
func (r Reconciler) isUpgradeRequired(actualCR *vzapi.Verrazzano) (bool, error) {
	if actualCR == nil {
		return false, fmt.Errorf("no Verrazzano CR provided")
	}
	bomVersion, err := validators.GetCurrentBomVersion()
	if err != nil {
		return false, err
	}

	if len(actualCR.Spec.Version) > 0 {
		specVersion, err := semver.NewSemVersion(actualCR.Spec.Version)
		if err != nil {
			return false, err
		}
		return specVersion.IsLessThan(bomVersion), nil
	}
	if len(actualCR.Status.Version) > 0 {
		statusVersion, err := semver.NewSemVersion(actualCR.Status.Version)
		if err != nil {
			return false, err
		}
		return statusVersion.IsLessThan(bomVersion), nil
	}
	return false, nil
}
