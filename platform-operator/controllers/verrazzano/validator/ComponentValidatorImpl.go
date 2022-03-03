// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package validator

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
)

type ComponentValidatorImpl struct{}

var _ v1alpha1.ComponentValidator = ComponentValidatorImpl{}

func (c ComponentValidatorImpl) ValidateInstall(vz *v1alpha1.Verrazzano) []error {
	var errs []error
	for _, comp := range registry.GetComponents() {
		if err := comp.ValidateInstall(vz); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (c ComponentValidatorImpl) ValidateUpdate(old *v1alpha1.Verrazzano, new *v1alpha1.Verrazzano) []error {
	var errs []error
	for _, comp := range registry.GetComponents() {
		if err := comp.ValidateUpdate(old, new); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
