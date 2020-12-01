// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetComponents(t *testing.T) {
	assert := assert.New(t)
	comps := GetComponents()
	assert.Len(comps, 1, "Wrong number of components")
	assert.Equal(comps[0].Name(), "verrazzano")
}
