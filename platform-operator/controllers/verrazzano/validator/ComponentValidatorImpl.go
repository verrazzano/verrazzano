// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validator

import (
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
)

type ComponentValidatorImpl struct{}

var _ v1alpha1.ComponentValidator = ComponentValidatorImpl{}

func (c ComponentValidatorImpl) ValidateInstall(vz *v1alpha1.Verrazzano) []error {
	var errs []error

	effectiveCR, err := transform.GetEffectiveCR(vz)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	if depErrs := dependencyValidation(effectiveCR); len(depErrs) > 0 {
		errs = append(errs, depErrs...)
	}

	for _, comp := range registry.GetComponents() {
		if err := comp.ValidateInstall(effectiveCR); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (c ComponentValidatorImpl) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) []error {
	var errs []error

	effectiveNew, err := transform.GetEffectiveCR(new)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	effectiveOld, err := transform.GetEffectiveCR(old)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	if depErrs := dependencyValidation(effectiveNew); len(depErrs) > 0 {
		errs = append(errs, depErrs...)
	}

	for _, comp := range registry.GetComponents() {
		if err := comp.ValidateUpdate(effectiveOld, effectiveNew); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func dependencyValidation(effectiveCR *v1alpha1.Verrazzano) []error {
	var errs []error
	for _, comp := range registry.GetComponents() {
		if comp.IsEnabled(effectiveCR) {
			for _, dependencyName := range comp.GetDependencies() {
				found, dependency := registry.FindComponent(dependencyName)
				if !found {
					errs = append(errs, fmt.Errorf("dependency not found for %s: %s", comp.GetJSONName(), dependencyName))
				}
				if !dependency.IsEnabled(effectiveCR) {
					errs = append(errs, fmt.Errorf("dependency not enabled for %s: %s", comp.GetJSONName(), dependency.GetJSONName()))
				}
			}
		}
	}
	return errs
}
