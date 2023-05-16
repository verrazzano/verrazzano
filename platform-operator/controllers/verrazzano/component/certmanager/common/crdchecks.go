// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	crdsExist, err := checkCRDsExist(GetRequiredCertManagerCRDNames())
	if err != nil {
		return false, err
	}
	// Found required CRDs
	return crdsExist, nil
}

func checkCRDsExist(crdNames []string) (bool, error) {
	clientFunc, err := k8sutil.GetAPIExtV1ClientFunc()
	if err != nil {
		return false, err
	}
	for _, crdName := range crdNames {
		_, err := clientFunc.CustomResourceDefinitions().Get(context.TODO(), crdName, metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
	}
	return true, nil
}
