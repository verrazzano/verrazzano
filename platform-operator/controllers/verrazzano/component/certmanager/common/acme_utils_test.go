// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

// TestIsLetsEncryptProductionEnv tests the IsLetsEncryptProductionEnv functions
// GIVEN a call to IsLetsEncryptProductionEnv
// WHEN an ACME configuration is passed in
// THEN the function returns true if the ACME environment is for the LE production env
func TestIsLetsEncryptProductionEnv(t *testing.T) {
	assert.True(t, IsLetsEncryptProductionEnv(v1alpha1.Acme{Environment: letsencryptProduction}))
	assert.False(t, IsLetsEncryptProductionEnv(v1alpha1.Acme{Environment: letsEncryptStaging}))
}

// TestIsLetsEncryptStaging tests the IsLetsEncryptStaging functions
// GIVEN a call to IsLetsEncryptStaging
// WHEN a Verrazzano configuration is passed in
// THEN the function returns true if the ACME environment is for the LE staging env
func TestIsLetsEncryptStaging(t *testing.T) {
	assert.True(t, IsLetsEncryptStaging(
		v1alpha1.Acme{
			Environment: letsEncryptStaging,
		},
	))
	assert.False(t, IsLetsEncryptStaging(
		v1alpha1.Acme{
			Environment: letsencryptProduction,
		},
	))
	assert.False(t, IsLetsEncryptStaging(
		v1alpha1.Acme{
			Environment: "foo",
		},
	))
	assert.False(t, IsLetsEncryptStaging(
		v1alpha1.Acme{
			Environment: "",
		},
	))
}
