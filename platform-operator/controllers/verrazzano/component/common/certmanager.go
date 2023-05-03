// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package common

import (
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	client "sigs.k8s.io/controller-runtime/pkg/client"
)

type newClientFuncType func(opts client.Options) (client.Client, error)

var newClientFunc newClientFuncType = k8sutil.NewControllerRuntimeClient

func SetNewClientFunc(f newClientFuncType) {
	newClientFunc = f
}

func ResetNewClientFunc() {
	newClientFunc = k8sutil.NewControllerRuntimeClient
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

func CertManagerExistsInCluster(log vzlog.VerrazzanoLogger, cli client.Client) error {
	exists, err := CertManagerCrdsExist(cli)
	if err != nil {
		return err
	}
	if !exists {
		return log.ErrorfThrottledNewErr("CertManager CRDs not found in cluster")
	}
	return nil
}

func CertManagerCrdsExist(cli client.Client) (bool, error) {
	var err error
	crtClient := cli
	if crtClient == nil {
		if crtClient, err = newClientFunc(client.Options{}); err != nil {
			return false, err
		}
	}
	crdsExist, err := CheckCRDsExist(crtClient, GetRequiredCertManagerCRDNames())
	if err != nil {
		return false, err
	}
	// Found required CRDs
	return crdsExist, nil
}
