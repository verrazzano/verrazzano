// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validator

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	cmvalidate "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/validate"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/transform"
)

type ComponentValidatorImpl struct{}

var _ v1alpha1.ComponentValidator = ComponentValidatorImpl{}

var _ v1beta1.ComponentValidator = ComponentValidatorImpl{}

func (c ComponentValidatorImpl) ValidateInstall(vz *v1alpha1.Verrazzano) []error {
	var errs []error

	// Perform validations on the actual CR prior to computing the effective CR
	errs = append(errs, validateActualCRV1Alpha1(vz)...)
	if len(errs) > 0 {
		return errs
	}

	effectiveCR, err := transform.GetEffectiveCR(vz)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	for _, comp := range registry.GetComponents() {
		if err := comp.ValidateInstall(effectiveCR); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (c ComponentValidatorImpl) ValidateInstallV1Beta1(vz *v1beta1.Verrazzano) []error {
	var errs []error

	// Perform validations on the actual CR prior to computing the effective CR
	errs = append(errs, validateActualCRV1Beta1(vz)...)
	if len(errs) > 0 {
		return errs
	}

	effectiveCR, err := transform.GetEffectiveV1beta1CR(vz)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	for _, comp := range registry.GetComponents() {
		if err := comp.ValidateInstallV1Beta1(effectiveCR); err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (c ComponentValidatorImpl) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) []error {
	var errs []error

	// Perform validations on the actual CR prior to computing the effective CR
	errs = append(errs, validateActualCRV1Alpha1(new)...)
	if len(errs) > 0 {
		return errs
	}

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
		if err := comp.ValidateUpdate(effectiveOld, effectiveNew); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (c ComponentValidatorImpl) ValidateUpdateV1Beta1(old *v1beta1.Verrazzano, new *v1beta1.Verrazzano) []error {
	var errs []error

	// Perform validations on the actual CR prior to computing the effective CR
	errs = append(errs, validateActualCRV1Beta1(new)...)
	if len(errs) > 0 {
		return errs
	}

	effectiveNew, err := transform.GetEffectiveV1beta1CR(new)
	if err != nil {
		errs = append(errs, err)
		return errs
	}
	effectiveOld, err := transform.GetEffectiveV1beta1CR(old)
	if err != nil {
		errs = append(errs, err)
		return errs
	}

	for _, comp := range registry.GetComponents() {
		if err := comp.ValidateUpdateV1Beta1(effectiveOld, effectiveNew); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

// validateActualCRV1Beta1 peforms validations of the actual CR stored in etcd; this allows us to perform
// validations based on user intent, before the EffectiveCR is computed
func validateActualCRV1Beta1(vz *v1beta1.Verrazzano) []error {
	var errs []error
	if vz == nil {
		return []error{}
	}
	if actualCRErrs := cmvalidate.ValidateActualConfigurationV1Beta1(vz); actualCRErrs != nil {
		errs = append(errs, actualCRErrs...)
	}
	return errs
}

// validateActualCRV1Alpha1 peforms validations of the actual CR stored in etcd; this allows us to perform
// validations based on user intent, before the EffectiveCR is computed
func validateActualCRV1Alpha1(vz *v1alpha1.Verrazzano) []error {
	var errs []error
	if vz == nil {
		return []error{}
	}
	if actualCRErrs := cmvalidate.ValidateActualConfigurationV1Alpha1(vz); actualCRErrs != nil {
		errs = append(errs, actualCRErrs...)
	}
	return errs
}
