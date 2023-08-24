// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"context"
	"fmt"
	moduleapi "github.com/verrazzano/verrazzano-modules/module-operator/apis/platform/v1alpha1"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/result"
	"github.com/verrazzano/verrazzano-modules/pkg/controller/spi/controllerspi"
	"github.com/verrazzano/verrazzano/pkg/bom"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
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
	"strings"
)

// Reconcile reconciles the Verrazzano CR
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

	// If an upgrade is pending, do not reconcile; an upgrade is pending if the VPO has been upgraded, but the user
	// has not started an upgrade of the Verrazzano install.
	if upgradePending, err := r.isUpgradePending(actualCR); upgradePending || err != nil {
		// return an error if encountered, otherwise returns an empty result to stop reconciling
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
		_, err = controllerutil.CreateOrUpdate(context.TODO(), r.Client, &module, func() error {
			// TODO For now have the module version match the VZ version
			return r.mutateModule(log, effectiveCR, &module, comp, version.ToString())
		})
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

	module.Spec.ModuleName = module.Name
	module.Spec.TargetNamespace = comp.Namespace()
	module.Spec.Version = moduleVersion

	return r.setModuleValues(log, effectiveCR, module, comp)
}

// isUpgradePending Returns true if the Spec version is set but doesn't match what's in the BOM.
//   - if spec version is empty then it's either an initial install or an update within the same version
//   - if the Spec version IS NOT empty and matches the BOM, we're either in an upgrade or updating within the same version
//   - if the Spec version IS empty and does NOT match the BOM, then we have updated the VPO but not yet upgraded, so we should
//     skip reconciling
func (r Reconciler) isUpgradePending(actualCR *vzapi.Verrazzano) (bool, error) {
	if actualCR == nil {
		return false, fmt.Errorf("no Verrazzano CR provided")
	}
	newBom, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return false, err
	}
	bomVersion := strings.TrimSpace(newBom.GetVersion())
	specVersion := strings.TrimSpace(actualCR.Spec.Version)
	if len(specVersion) > 0 {
		return specVersion != bomVersion, nil
	}
	return false, nil
}
