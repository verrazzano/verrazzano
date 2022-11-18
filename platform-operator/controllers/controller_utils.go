// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package controllers

import (
	"fmt"
	vzstatus "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/status"

	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"

	installv1alpha1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// VzContainsResource checks to see if the resource is listed in the Verrazzano
func VzContainsResource(ctx spi.ComponentContext, objectName string, objectKind string) (string, bool) {
	for _, component := range registry.GetComponents() {
		if component.MonitorOverrides(ctx) {
			if found := componentContainsResource(component.GetOverrides(ctx.EffectiveCR()).([]installv1alpha1.Overrides), objectName, objectKind); found {
				return component.Name(), found
			}
		}
	}
	return "", false
}

// componentContainsResource looks through the component override list see if the resource is listed
func componentContainsResource(Overrides []installv1alpha1.Overrides, objectName string, objectKind string) bool {
	for _, override := range Overrides {
		if objectKind == constants.ConfigMapKind && override.ConfigMapRef != nil {
			if objectName == override.ConfigMapRef.Name {
				return true
			}
		}
		if objectKind == constants.SecretKind && override.SecretRef != nil {
			if objectName == override.SecretRef.Name {
				return true
			}
		}
	}
	return false
}

// UpdateVerrazzanoForInstallOverrides mutates the status subresource of Verrazzano Custom Resource specific
// to a component to cause a reconcile
func UpdateVerrazzanoForInstallOverrides(statusUpdater vzstatus.Updater, componentCtx spi.ComponentContext, componentName string) error {
	cr := componentCtx.ActualCR()
	// Return an error to requeue if Verrazzano Component Status hasn't been initialized
	if cr.Status.Components == nil {
		return fmt.Errorf("Components not initialized")
	}
	// Set ReconcilingGeneration to 1 to re-enter install flow
	details := cr.Status.Components[componentName].DeepCopy()
	details.ReconcilingGeneration = 1
	componentsToUpdate := map[string]*installv1alpha1.ComponentStatusDetails{
		componentName: details,
	}
	statusUpdater.Update(&vzstatus.UpdateEvent{
		Verrazzano: cr,
		Components: componentsToUpdate,
	})
	return nil
}

// ProcDeletedOverride checks Verrazzano CR for an override resource that has now been deleted,
// and updates the CR if the resource is found listed as an override
func ProcDeletedOverride(statusUpdater vzstatus.Updater, c client.Client, vz *installv1alpha1.Verrazzano, objectName string, objectKind string) error {
	// DefaultLogger is used since we only need to create a component context and any actual logging isn't being performed
	log := vzlog.DefaultLogger()
	ctx, err := spi.NewContext(log, c, vz, nil, false)
	if err != nil {
		return err
	}

	compName, ok := VzContainsResource(ctx, objectName, objectKind)
	if !ok {
		return nil
	}

	if err := UpdateVerrazzanoForInstallOverrides(statusUpdater, ctx, compName); err != nil {
		return err
	}
	return nil
}
