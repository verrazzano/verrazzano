// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

func IsOCIDNS(vz *vzapi.Verrazzano) bool {
	return vz.Spec.Components.DNS != nil && vz.Spec.Components.DNS.OCI != nil
}

// IsCAConfig - Check if cert-type is CA, if not it is assumed to be Acme
func IsCAConfig(certConfig vzapi.Certificate) (bool, error) {
	return checkExactlyOneIssuerConfiguration(certConfig)
}

func IsCAConfigV1Beta1(certConfig v1beta1.Certificate) (bool, error) {
	return checkExactlyOneIssuerConfigurationV1Beta1(certConfig)
}
