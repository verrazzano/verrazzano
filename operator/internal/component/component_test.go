// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetComponents tests getting the components
// GIVEN a component
//  WHEN I call GetComponents
//  THEN the Get returns the correct components
func TestGetComponents(t *testing.T) {
	assert := assert.New(t)
	comps := GetComponents()
	assert.Len(comps, 4, "Wrong number of components")
	assert.Equal(comps[0].Name(), "verrazzano")
	assert.Equal(comps[1].Name(), "ingress-nginx")
	assert.Equal(comps[2].Name(), "keycloak")
	assert.Equal(comps[3].Name(), "external-dns")
}
