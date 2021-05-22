// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestLoadBom tests loading the bom json into a struct
// GIVEN a json file
//  WHEN I call loadBom
//  THEN the correct verrazzano bom is returned
func TestLoadBom(t *testing.T) {
	assert := assert.New(t)
	bom, err := loadBom("testdata/test_bom.json")
	assert.NoError(err, "error calling loadBom")
	assert.Equal("ghcr.io", bom.Registry, "Wrong registry name")
}
