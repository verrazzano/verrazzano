// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	cmconstants "github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager/constants"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
)

// TestIsLetsEncryptProductionEnv tests the IsLetsEncryptProductionEnv functions
// GIVEN a call to IsLetsEncryptProductionEnv
// WHEN an LetsEncrypt configuration is passed in
// THEN the function returns true if the LetsEncrypt environment is for the LE production env
func TestIsLetsEncryptProductionEnv(t *testing.T) {
	assert.True(t, IsLetsEncryptProductionEnv(v1alpha1.LetsEncryptACMEIssuer{Environment: cmconstants.LetsEncryptProduction}))
	assert.True(t, IsLetsEncryptProductionEnv(v1alpha1.LetsEncryptACMEIssuer{}))
	assert.False(t, IsLetsEncryptProductionEnv(v1alpha1.LetsEncryptACMEIssuer{Environment: cmconstants.LetsEncryptStaging}))
	assert.True(t, IsLetsEncryptProductionEnv(v1beta1.LetsEncryptACMEIssuer{Environment: cmconstants.LetsEncryptProduction}))
	assert.True(t, IsLetsEncryptProductionEnv(v1beta1.LetsEncryptACMEIssuer{}))
	assert.False(t, IsLetsEncryptProductionEnv(v1beta1.LetsEncryptACMEIssuer{Environment: cmconstants.LetsEncryptStaging}))
	assert.True(t, IsLetsEncryptProductionEnv(v1alpha1.Acme{Environment: cmconstants.LetsEncryptProduction}))
	assert.True(t, IsLetsEncryptProductionEnv(v1alpha1.Acme{}))
	assert.False(t, IsLetsEncryptProductionEnv(v1alpha1.Acme{Environment: cmconstants.LetsEncryptStaging}))
	assert.True(t, IsLetsEncryptProductionEnv(v1beta1.Acme{Environment: cmconstants.LetsEncryptProduction}))
	assert.False(t, IsLetsEncryptProductionEnv(v1beta1.Acme{Environment: cmconstants.LetsEncryptStaging}))
	assert.True(t, IsLetsEncryptProductionEnv(v1beta1.Acme{}))
}

// TestIsLetsEncryptStagingEnv tests the IsLetsEncryptStagingEnv functions
// GIVEN a call to IsLetsEncryptStagingEnv
// WHEN a Verrazzano configuration is passed in
// THEN the function returns true if the LetsEncrypt environment is for the LE staging env
func TestIsLetsEncryptStagingEnv(t *testing.T) {
	assert.True(t, IsLetsEncryptStagingEnv(
		v1alpha1.LetsEncryptACMEIssuer{
			Environment: cmconstants.LetsEncryptStaging,
		},
	))
	assert.False(t, IsLetsEncryptStagingEnv(
		v1alpha1.LetsEncryptACMEIssuer{
			Environment: cmconstants.LetsEncryptProduction,
		},
	))
	assert.False(t, IsLetsEncryptStagingEnv(
		v1beta1.LetsEncryptACMEIssuer{
			Environment: "foo",
		},
	))
	assert.False(t, IsLetsEncryptStagingEnv(
		v1beta1.LetsEncryptACMEIssuer{
			Environment: "",
		},
	))
	assert.True(t, IsLetsEncryptStagingEnv(
		v1alpha1.Acme{
			Environment: cmconstants.LetsEncryptStaging,
		},
	))
	assert.False(t, IsLetsEncryptStagingEnv(
		v1alpha1.Acme{
			Environment: cmconstants.LetsEncryptProduction,
		},
	))
	assert.False(t, IsLetsEncryptStagingEnv(
		v1beta1.Acme{
			Environment: "foo",
		},
	))
	assert.False(t, IsLetsEncryptStagingEnv(
		v1beta1.Acme{
			Environment: "",
		},
	))

	assert.False(t, IsLetsEncryptStagingEnv(v1alpha1.LetsEncryptACMEIssuer{}))
	assert.False(t, IsLetsEncryptStagingEnv(v1beta1.LetsEncryptACMEIssuer{}))
	assert.False(t, IsLetsEncryptStagingEnv(v1alpha1.Acme{}))
	assert.False(t, IsLetsEncryptStagingEnv(v1beta1.Acme{}))
}

// TestIsLetsEncryptProvider tests the IsLetsEncryptProvider functions
// GIVEN a call to IsLetsEncryptProvider
// WHEN a various valid/invalid LetsEncrypt provider names are passed in
// THEN the function returns true if the provider type matches the LetsEncrypt type (case-insensitive), false otherwise
func TestIsLetsEncryptProvider(t *testing.T) {
	assert.True(t, IsLetsEncryptProvider(v1alpha1.Acme{Provider: v1alpha1.LetsEncrypt}))
	assert.True(t, IsLetsEncryptProvider(v1alpha1.Acme{Provider: "LETSencRYPt"}))
	assert.False(t, IsLetsEncryptProvider(v1alpha1.Acme{Provider: "foo"}))
	assert.True(t, IsLetsEncryptProvider(v1beta1.Acme{Provider: v1beta1.LetsEncrypt}))
	assert.True(t, IsLetsEncryptProvider(v1beta1.Acme{Provider: "LETSencRYPt"}))
	assert.False(t, IsLetsEncryptProvider(v1beta1.Acme{Provider: "foo"}))
}
