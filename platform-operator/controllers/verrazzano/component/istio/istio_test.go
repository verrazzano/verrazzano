// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package istio

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAppendIstioOverrides tests the Istio override for the global hub
// GIVEN the registry ovverride env var is set
//  WHEN I call AppendIstioOverrides
//  THEN the Istio global.hub helm override is added to the provided array/slice.
func TestAppendIstioOverrides(t *testing.T) {
	assert := assert.New(t)

	config.SetDefaultBomFilePath(testBomFilePath)

	os.Setenv(constants.RegistryOverrideEnvVar, "myreg.io")
	defer os.Unsetenv(constants.RegistryOverrideEnvVar)

	kvs, err := AppendIstioOverrides(nil, "istiod", "", "", nil)
	assert.NoError(err, "AppendIstioOverrides returned an error ")
	assert.Len(kvs, 1, "AppendIstioOverrides returned wrong number of Key:Value pairs")
	assert.Equal(istioGlobalHubKey, kvs[0].Key)
	assert.Equal("myreg.io/verrazzano", kvs[0].Value)

	os.Setenv(constants.ImageRepoOverrideEnvVar, "myrepo")
	defer os.Unsetenv(constants.ImageRepoOverrideEnvVar)
	kvs, err = AppendIstioOverrides(nil, "istiod", "", "", nil)
	assert.NoError(err, "AppendIstioOverrides returned an error ")
	assert.Len(kvs, 1, "AppendIstioOverrides returned wrong number of Key:Value pairs")
	assert.Equal(istioGlobalHubKey, kvs[0].Key)
	assert.Equal("myreg.io/myrepo/verrazzano", kvs[0].Value)
}

// TestAppendIstioOverridesNoRegistryOverride tests the Istio override for the global hub when no registry override is specified
// GIVEN the registry ovverride env var is NOT set
//  WHEN I call AppendIstioOverrides
//  THEN no overrides are added to the provided array/slice
func TestAppendIstioOverridesNoRegistryOverride(t *testing.T) {
	assert := assert.New(t)

	config.SetDefaultBomFilePath(testBomFilePath)

	kvs, err := AppendIstioOverrides(nil, "istiod", "", "", nil)
	assert.NoError(err, "AppendIstioOverrides returned an error ")
	assert.Len(kvs, 0, "AppendIstioOverrides returned wrong number of Key:Value pairs")
}
