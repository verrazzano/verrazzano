// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type testSubComponent struct {
	kvs map[string]string
}

// These are the key/values pairs that will be passed to helm as overrides.
// The map key is the subcomponent name.
var testSubcomponetHelmKeyValues = map[string]*testSubComponent{
	"istiocoredns": {
		kvs: map[string]string{
			"istiocoredns.coreDNSImage": "ghcr.io/verrazzano/coredns",
			"istiocoredns.coreDNSTag": "1.6.2",
			"istiocoredns.coreDNSPluginImage": "ghcr.io/verrazzano/istio-coredns-plugin:0.2-20201016204812-23723dcb",
		},
	},
}

// TestLoadBom tests loading the bom json into a struct
// GIVEN a json file
// WHEN I call loadBom
// THEN the correct verrazzano bom is returned
func TestBom(t *testing.T) {
	assert := assert.New(t)
	bom, err := NewBom("testdata/test_bom.json")
	assert.NoError(err, "error calling NewBom")
	assert.Equal("ghcr.io", bom.bomDoc.Registry, "Wrong registry name")
	assert.Len(bom.bomDoc.Components,14, "incorrect number of Bom components")

	// Validate each component
	for _, comp := range bom.bomDoc.Components {
		for _, sub := range comp.SubComponents {
			// Get the expected key/value pair overrides
			expectedSub := testSubcomponetHelmKeyValues[sub.Name]
			if expectedSub == nil{
				fmt.Println("Skipping subcomponent " + sub.Name)
				continue
			}
			//// Get the key value override list for this subcomponent
			foundKvs, err := bom.buildOverrides(sub.Name)
			assert.NoError(err, "error calling buildOverrides")
			assert.Equal(len(expectedSub.kvs), len(foundKvs), "Incorrect override list len")

			// Loop through the found kv pairs and make sure they match
			for _, kv := range foundKvs {
				expectedVal, ok := expectedSub.kvs[kv.key]
				assert.True(ok,"Found unexpected key in override list")
				assert.Equal(expectedVal, kv.value, "Found unexpected key value in override list")
				}
			}
	}



}
