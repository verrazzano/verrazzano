// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppendIstioOverrides tests the Istio override for the global hub
// GIVEN the registry ovverride env var is set
//  WHEN I call appendIstioOverrides
//  THEN the Istio global.hub helm override is added to the provided array/slice.
func TestAppendIstioOverrides(t *testing.T) {
	assert := assert.New(t)

	os.Setenv(constants.RegistryOverrideEnvVar, "myreg.io")
	defer os.Unsetenv(constants.RegistryOverrideEnvVar)

	kvs, err := appendIstioOverrides(nil, "", "", "", nil)
	assert.NoError(err, "appendIstioOverrides returned an error ")
	assert.Len(kvs, 1, "appendIstioOverrides returned wrong number of key:value pairs")
	assert.Equal(istioGlobalHubKey, kvs[0].key)
	assert.Equal("myreg.io", kvs[0].value)
}

// TestAppendIstioOverridesNoRegistryOverride tests the Istio override for the global hub when no registry override is specified
// GIVEN the registry ovverride env var is NOT set
//  WHEN I call appendIstioOverrides
//  THEN no overrides are added to the provided array/slice
func TestAppendIstioOverridesNoRegistryOverride(t *testing.T) {
	assert := assert.New(t)

	kvs, err := appendIstioOverrides(nil, "", "", "", nil)
	assert.NoError(err, "appendIstioOverrides returned an error ")
	assert.Len(kvs, 0, "appendIstioOverrides returned wrong number of key:value pairs")
}
