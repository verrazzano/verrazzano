// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzcr

import (
	"github.com/verrazzano/verrazzano/pkg/constants"
	"strings"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

func IsLetsEncryptProductionEnv(acme interface{}) bool {
	var envName string
	if v1alpha1LEIssuer, ok := acme.(vzapi.LetsEncryptACMEIssuer); ok {
		envName = v1alpha1LEIssuer.Environment
	}
	if v1beta1LEIssuer, ok := acme.(v1beta1.LetsEncryptACMEIssuer); ok {
		envName = v1beta1LEIssuer.Environment
	}
	if v1alpha1ACME, ok := acme.(vzapi.Acme); ok {
		envName = v1alpha1ACME.Environment
	}
	if v1beta1ACME, ok := acme.(v1beta1.Acme); ok {
		envName = v1beta1ACME.Environment
	}
	if len(envName) == 0 {
		// the default if not specified
		return true
	}
	return strings.ToLower(envName) == constants.LetsEncryptProduction
}

func IsLetsEncryptStagingEnv(acme interface{}) bool {
	var envName string
	if v1alpha1LEIssuer, ok := acme.(vzapi.LetsEncryptACMEIssuer); ok {
		envName = v1alpha1LEIssuer.Environment
	}
	if v1beta1LEIssuer, ok := acme.(v1beta1.LetsEncryptACMEIssuer); ok {
		envName = v1beta1LEIssuer.Environment
	}
	if v1alpha1ACME, ok := acme.(vzapi.Acme); ok {
		envName = v1alpha1ACME.Environment
	}
	if v1beta1ACME, ok := acme.(v1beta1.Acme); ok {
		envName = v1beta1ACME.Environment
	}
	return strings.ToLower(envName) == constants.LetsEncryptStaging
}

func IsLetsEncryptProvider(acme interface{}) bool {
	if v1alpha1ACME, ok := acme.(vzapi.Acme); ok {
		return strings.ToLower(string(v1alpha1ACME.Provider)) == strings.ToLower(string(vzapi.LetsEncrypt))
	}
	if v1beta1ACME, ok := acme.(v1beta1.Acme); ok {
		return strings.ToLower(string(v1beta1ACME.Provider)) == strings.ToLower(string(v1beta1.LetsEncrypt))
	}
	return false
}

func IsPrivateIssuer(c interface{}) (bool, error) {
	var isCAIssuer, isLetsEncryptStagingIssuer bool
	var err error
	if v1alpha1Issuer, ok := c.(*vzapi.ClusterIssuerComponent); ok {
		isCAIssuer, err = v1alpha1Issuer.IsCAIssuer()
		if !isCAIssuer {
			isLetsEncryptStagingIssuer = IsLetsEncryptStagingEnv(*v1alpha1Issuer.LetsEncrypt)
		}
	}
	if v1beta1Issuer, ok := c.(*v1beta1.ClusterIssuerComponent); ok {
		isCAIssuer, err = v1beta1Issuer.IsCAIssuer()
		if !isCAIssuer {
			isLetsEncryptStagingIssuer = IsLetsEncryptStagingEnv(*v1beta1Issuer.LetsEncrypt)
		}
	}
	if err != nil {
		return false, err
	}
	return isCAIssuer || isLetsEncryptStagingIssuer, nil
}
