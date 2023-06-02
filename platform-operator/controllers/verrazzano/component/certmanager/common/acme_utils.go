// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
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
	return strings.ToLower(envName) == cmconstants.LetsEncryptProduction
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
	return strings.ToLower(envName) == cmconstants.LetsEncryptStaging
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
