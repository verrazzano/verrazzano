// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"strings"
)

func IsLetsEncryptProductionEnv(acme vzapi.Acme) bool {
	return strings.ToLower(acme.Environment) == letsencryptProduction
}

func IsLetsEncryptStaging(compContext spi.ComponentContext) bool {
	acmeEnvironment := compContext.EffectiveCR().Spec.Components.CertManager.Certificate.Acme.Environment
	return acmeEnvironment != "" && strings.ToLower(acmeEnvironment) != "production"
}

func convertIfNecessary(vz interface{}) (*vzapi.Verrazzano, error) {
	if vz == nil {
		return nil, fmt.Errorf("Unable to convert, nil Verrazzano reference")
	}
	if vzv1beta1, ok := vz.(*v1beta1.Verrazzano); ok {
		cr := &vzapi.Verrazzano{}
		if err := cr.ConvertFrom(vzv1beta1); err != nil {
			return nil, err
		}
		return cr, nil
	}
	cr, ok := vz.(*vzapi.Verrazzano)
	if !ok {
		return nil, fmt.Errorf("Unable to convert, not a Verrazzano v1alpha1 reference")
	}
	return cr, nil
}
