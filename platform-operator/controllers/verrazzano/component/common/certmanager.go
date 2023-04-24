// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package common

import (
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	apiextensionsv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
)

type getAPIExtV1ClientFuncType func(log ...vzlog.VerrazzanoLogger) (apiextensionsv1client.ApiextensionsV1Interface, error)

var getAPIExtV1ClientFunc getAPIExtV1ClientFuncType = k8sutil.GetAPIExtV1Client

func SetAPIExtV1ClientFunc(f getAPIExtV1ClientFuncType) {
	getAPIExtV1ClientFunc = f
}

func ResetAPIExtV1ClientFunc() {
	getAPIExtV1ClientFunc = k8sutil.GetAPIExtV1Client
}

var certManagerCRDNames = []string{
	"certificaterequests.cert-manager.io",
	"orders.acme.cert-manager.io",
	"certificates.cert-manager.io",
	"clusterissuers.cert-manager.io",
	"issuers.cert-manager.io",
}

func GetRequiredCertManagerCRDNames() []string {
	return certManagerCRDNames
}

func CertManagerExistsInCluster(log vzlog.VerrazzanoLogger) error {
	exists, err := CertManagerCrdsExist()
	if err != nil {
		return err
	}
	if !exists {
		return log.ErrorfThrottledNewErr("CertManager custom resources not found in cluster")
	}
	return nil
}

func CertManagerCrdsExist() (bool, error) {
	client, err := getAPIExtV1ClientFunc()
	if err != nil {
		return false, err
	}
	crdsExist, err := CheckCRDsExist(GetRequiredCertManagerCRDNames(), err, client)
	if err != nil {
		return false, err
	}
	// Found required CRDs
	return crdsExist, nil
}
