// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/registry"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
)

// TestAppendKeycloakOverrides tests the Keycloak override for the theme images
// GIVEN a Verrazzano BOM
//  WHEN I call appendKeycloakOverrides
//  THEN the Keycloak theme override is added to the Key:Value array.
func TestAppendKeycloakOverrides(t *testing.T) {
	const defNs = constants.VerrazzanoSystemNamespace
	assert := assert.New(t)

	config.SetDefaultBomFilePath(registry.testBomFilePath)
	kvs, err := appendKeycloakOverrides(nil, "", "", "", nil)
	assert.NoError(err, "appendKeycloakOverrides returned an error ")
	assert.Len(kvs, 1, "appendKeycloakOverrides returned wrong number of Key:Value pairs")
}
