// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppendCertManagerOverridesNoRegistryOverride tests the CertManager overrides for the cainjector and webhook
// images when no registry override is set
// GIVEN the registry ovverride env var is NOT set
//  WHEN I call appendCertManagerOverrides
//  THEN the the additional CertManager overrides were NOT added to the array slice.
func TestAppendCertManagerOverridesNoRegistryOverride(t *testing.T) {
	assert := assert.New(t)

	SetUnitTestBomFilePath(sampleTestBomFilePath)

	kvs, err := appendCertManagerOverrides(nil, "cert-manager", "", "", nil)
	assert.NoError(err, "appendCertManagerOverrides returned an error ")
	assert.Len(kvs, 0, "appendCertManagerOverrides returned wrong number of key:value pairs")

}

// TestAppendCertManagerOverridesRegistryOverride tests the CertManager overrides for the cainjector and webhook
// images when user-supplied registry override is set
// GIVEN the registry ovverride env var IS set
//  WHEN I call appendCertManagerOverrides
//  THEN the the additional CertManager overrides ARE added to the array slice.
func TestAppendCertManagerOverridesRegistryOverride(t *testing.T) {
	assert := assert.New(t)

	SetUnitTestBomFilePath(sampleTestBomFilePath)

	os.Setenv(constants.RegistryOverrideEnvVar, "myreg.io")
	defer os.Unsetenv(constants.RegistryOverrideEnvVar)

	kvs, err := appendCertManagerOverrides(nil, "cert-manager", "", "", nil)
	assert.NoError(err, "appendCertManagerOverrides returned an error ")
	assert.Len(kvs, 4, "appendCertManagerOverrides returned wrong number of key:value pairs")
	t.Logf("Additional CertManager overrides: %v", kvs)
}
