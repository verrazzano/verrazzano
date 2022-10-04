// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzmap

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestUnionStringMaps verifies the union of two string maps
func TestUnionStringMaps(t *testing.T) {
	m1 := map[string]string{
		"a": "1",
		"b": "2",
		"c": "3",
	}
	m2 := map[string]string{
		"a": "1",
		"e": "2",
		"f": "5",
	}

	u := UnionStringMaps(m1, m2)
	for k := range m1 {
		assert.Equal(t, u[k], m1[k])
	}
	for k := range m2 {
		assert.Equal(t, u[k], m2[k])
	}
}
