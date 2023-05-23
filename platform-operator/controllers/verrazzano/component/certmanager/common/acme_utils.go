// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"strings"
)

func IsLetsEncryptProductionEnv(acme interface{}) bool {
	if v1alpha1ACME, ok := acme.(vzapi.LetsEncryptACMEIssuer); ok {
		return strings.ToLower(v1alpha1ACME.Environment) == LetsencryptProduction
	}
	if v1beta1ACME, ok := acme.(v1beta1.LetsEncryptACMEIssuer); ok {
		return strings.ToLower(v1beta1ACME.Environment) == LetsencryptProduction
	}
	return false
}

func IsLetsEncryptStaging(acme interface{}) bool {
	if v1alpha1ACME, ok := acme.(vzapi.LetsEncryptACMEIssuer); ok {
		return v1alpha1ACME.Environment == LetsEncryptStaging
	}
	if v1beta1ACME, ok := acme.(v1beta1.LetsEncryptACMEIssuer); ok {
		return v1beta1ACME.Environment == LetsEncryptStaging
	}
	return false
}

func IsLetsEncryptStagingEnv(acme interface{}) bool {
	if v1alpha1ACME, ok := acme.(vzapi.LetsEncryptACMEIssuer); ok {
		return strings.ToLower(v1alpha1ACME.Environment) == LetsEncryptStaging
	}
	if v1beta1ACME, ok := acme.(v1beta1.LetsEncryptACMEIssuer); ok {
		return strings.ToLower(v1beta1ACME.Environment) == LetsEncryptStaging
	}
	return false
}
