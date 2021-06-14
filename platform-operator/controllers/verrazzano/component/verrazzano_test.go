// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/application-operator/constants"
)

// TestVzResolveNamespace tests the Verrazzano component name
// GIVEN a Verrazzano component
//  WHEN I call resolveNamespace
//  THEN the Verrazzano namespace name is correctly resolved
func TestVzResolveNamespace(t *testing.T) {
	const defNs = constants.VerrazzanoSystemNamespace
	assert := assert.New(t)
	ns := resolveVerrazzanoNamespace("")
	assert.Equal(defNs, ns, "Wrong namespace resolved for verrazzano when using empty namespace")
	ns = resolveVerrazzanoNamespace("default")
	assert.Equal(defNs, ns, "Wrong namespace resolved for verrazzano when using default namespace")
	ns = resolveVerrazzanoNamespace("custom")
	assert.Equal("custom", ns, "Wrong namespace resolved for verrazzano when using custom namesapce")
}
