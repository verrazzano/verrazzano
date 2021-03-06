// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
)

// TestAppendAppOperatorOverrides tests the Keycloak override for the theme images
// GIVEN an env override for the app operator image
//  WHEN I call appendApplicationOperatorOverrides
//  THEN the "image" key is set with the image override.
func TestAppendAppOperatorOverrides(t *testing.T) {
	assert := assert.New(t)

	customImage := "myreg.io/myrepo/v8o/verrazzano-application-operator-dev:local-20210707002801-b7449154"

	kvs, err := appendApplicationOperatorOverrides(nil, "", "", "", nil)
	assert.NoError(err, "appendApplicationOperatorOverrides returned an error ")
	assert.Len(kvs, 0, "appendApplicationOperatorOverrides returned an unexpected number of key:value pairs")

	os.Setenv(constants.VerrazzanoAppOperatorImageEnvVar, customImage)
	defer os.Unsetenv(constants.RegistryOverrideEnvVar)

	SetUnitTestBomFilePath(testBomFilePath)
	kvs, err = appendApplicationOperatorOverrides(nil, "", "", "", nil)
	assert.NoError(err, "appendApplicationOperatorOverrides returned an error ")
	assert.Len(kvs, 1, "appendApplicationOperatorOverrides returned wrong number of key:value pairs")
	assert.Equalf("image", kvs[0].key, "Did not get expected image key")
	assert.Equalf(customImage, kvs[0].value, "Did not get expected image value")
}
