// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
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

// CertManagerExistsInCluster returns an error if the CRDs do not exist in the cluster
// - used in situations where only an error can be returned
func CertManagerExistsInCluster(log vzlog.VerrazzanoLogger) error {
	exists, err := CertManagerCrdsExist()
	if err != nil {
		return err
	}
	if !exists {
		return log.ErrorfThrottledNewErr("CertManager CRDs not found in cluster")
	}
	return nil
}

// CertManagerCrdsExist returns true if the Cert-Manager CRDs exist in the cluster, false otherwise
func CertManagerCrdsExist() (bool, error) {
	crdsExist, err := common.CheckCRDsExist(GetRequiredCertManagerCRDNames())
	if err != nil {
		return false, err
	}
	// Found required CRDs
	return crdsExist, nil
}
