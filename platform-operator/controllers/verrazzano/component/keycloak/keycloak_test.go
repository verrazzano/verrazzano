// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package keycloak

import (
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
)

const testBomFilePath = "../../testdata/test_bom.json"

// TestAppendKeycloakOverrides tests the Keycloak override for the theme images
// GIVEN a Verrazzano BOM
//  WHEN I call AppendKeycloakOverrides
//  THEN the Keycloak theme override is added to the Key:Value array.
func TestAppendKeycloakOverrides(t *testing.T) {
	const defNs = constants.VerrazzanoSystemNamespace
	assert := assert.New(t)

	config.SetDefaultBomFilePath(testBomFilePath)
	kvs, err := AppendKeycloakOverrides(nil, "", "", "", nil)
	assert.NoError(err, "AppendKeycloakOverrides returned an error ")
	assert.Len(kvs, 1, "AppendKeycloakOverrides returned wrong number of Key:Value pairs")
}
