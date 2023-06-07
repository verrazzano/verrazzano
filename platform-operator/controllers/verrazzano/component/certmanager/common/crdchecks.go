// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
)

var certManagerCRDNames = []string{
	"certificaterequests.cert-manager.io",
	"orders.acme.cert-manager.io",
	"certificates.cert-manager.io",
	"clusterissuers.cert-manager.io",
	"issuers.cert-manager.io",
}

// GetRequiredCertManagerCRDNames returns a list of required/expected Cert-Manager CRDs
func GetRequiredCertManagerCRDNames() []string {
	return certManagerCRDNames
}

// CertManagerCRDsExist returns true if the Cert-Manager CRDs exist in the cluster, false otherwise
func CertManagerCRDsExist() (bool, error) {
	crdsExist, err := common.CheckCRDsExist(GetRequiredCertManagerCRDNames())
	if err != nil {
		return false, err
	}
	// Found required CRDs
	return crdsExist, nil
}
