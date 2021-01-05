// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetComponents(t *testing.T) {
	assert := assert.New(t)
	comps := GetComponents()
	assert.Len(comps, 2, "Wrong number of components")
	assert.Equal(comps[0].Name(), "verrazzano")
	assert.Equal(comps[1].Name(), "nginx-ingress-controller")
}
