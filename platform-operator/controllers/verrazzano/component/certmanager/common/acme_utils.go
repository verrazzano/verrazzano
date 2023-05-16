// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"strings"
)

func IsLetsEncryptProductionEnv(acme vzapi.Acme) bool {
	return strings.ToLower(acme.Environment) == letsencryptProduction
}

func IsLetsEncryptStaging(acme vzapi.Acme) bool {
	return acme.Environment == letsEncryptStaging
}

func isLetsEncryptProvider(acme v1beta1.Acme) bool {
	return strings.ToLower(string(acme.Provider)) == strings.ToLower(string(vzapi.LetsEncrypt))
}

func isLetsEncryptStagingEnv(acme v1beta1.Acme) bool {
	return strings.ToLower(acme.Environment) == letsEncryptStaging
}

func isLetsEncryptProductionEnv(acme v1beta1.Acme) bool {
	return strings.ToLower(acme.Environment) == letsencryptProduction
}
