// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validator

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
	"go.uber.org/zap"
)

type ComponentValidatorImpl struct{}

var _ v1alpha1.ComponentValidator = ComponentValidatorImpl{}

func (c ComponentValidatorImpl) ValidateInstall(vz *v1alpha1.Verrazzano, log *zap.SugaredLogger) []error {
	var errs []error

	effectiveCR, err := transform.GetEffectiveCR(vz)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	for _, comp := range registry.GetComponents() {
		if err := comp.ValidateInstall(effectiveCR, log); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (c ComponentValidatorImpl) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano, log *zap.SugaredLogger) []error {
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

	for _, comp := range registry.GetComponents() {
		if err := comp.ValidateUpdate(effectiveOld, effectiveNew, log); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
