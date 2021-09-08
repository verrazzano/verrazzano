// Copyright (c) 2020, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	vzcr "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

)

// Empty struct for strategy
type simple struct {
}

const simpleBase = `
name: base
`
const simpleOverlay = `
name: overlay
`
const simpleMerged = `name: overlay`

// Nested struct and test YAML
type nested1 struct {
	Name string `json:"name"`
	Host struct {
		Name string `json:"name"`
		IP   string `json:"name"`
	}
}

type Patch struct {
	Version string `json:"version"`
	Date    string `json:"date"`
}

// Nested struct and test YAML
type nested2 struct {
	Name string `json:"name"`
	Host struct {
		Name string `json:"name"`
		IP   string `json:"name"`
	}
	Platform struct {
		Vendor string `json:"vendor"`
		OS     struct {
			Name    string `json:"name"`
			Patches []Patch `json:"patches" patchStrategy:"merge" patchMergeKey:"version"`
		}
	}
}


//  patchesReplace specifies that the array will be replaced on a merge
type patchesReplace struct {
	Patches []Patch `json:"patches"`
}

//  patchesMerge specifies that the array will merge
type patchesMerge struct {
	Patches []Patch `json:"patches" patchStrategy:"merge" patchMergeKey:"version"`
}


const nested1Base = `
name: base
host:
  name: example.com
`
const nested1Overlay = `
name: overlay
host:
  ip: 1.2.3.4
`
const nested1Merged = `
name: overlay
host:
  name: example.com
  ip: 1.2.3.4`

// Yaml with nested struct arrays
const nested2Base = `
name: base
host:
  ip: 1.2.3.4
  name: foo
platform:
  vendor: company1
  os:
    name: linux
    patches: 
    - version: 0.5.0
      date: 01/01/2020
`
const nested2Overlay = `
platform:
  os:
    patches:
    - version: 0.6.0
      date: 02/02/2022
`

const nested2Merged = `
name: base
host:
  ip: 1.2.3.4
  name: foo
platform:
  vendor: company1
  os:
    name: linux
    patches: 
    - version: 0.6.0
      date: 02/02/2022
    - version: 0.5.0
      date: 01/01/2020
`

func TestMergeSimple(t *testing.T) {
	assert := assert.New(t)
	merged, err := MergeString(simpleBase, simpleOverlay, simple{})
	assert.NoError(err, merged, "error merging simple yaml")
	assert.YAMLEq(simpleMerged, merged, "simple yaml should be equal")
}

func TestMergeNested1(t *testing.T) {
	assert := assert.New(t)
	merged, err := MergeString(nested1Base, nested1Overlay, nested1{})
	assert.NoError(err, merged, "error merging nested yaml")
	assert.YAMLEq(nested1Merged, merged, "nested yaml should be equal")
}

func TestMergeNested2(t *testing.T) {
	assert := assert.New(t)
	merged, err := MergeString(nested2Base, nested2Overlay, nested2{})
	assert.NoError(err, merged, "error merging nested yaml")
	assert.YAMLEq(nested2Merged, merged, "nested yaml should be equal")
}

const yaml1 = `
host:
  name: foo
  ip: 1.2.3.4
name: base
platform:
  os:
    name: linux
    patches:
    - date: 02/02/2022
      version: 0.6.0
    - date: 01/01/2020
      version: 0.5.0
  vendor: company1
`
const yaml2 = `
host:
  ip: 1.2.3.4
  name: foo
name: base
platform:
  os:
    name: linux
    patches:
    - date: 02/02/2022
      version: 0.6.0
    - date: 01/01/2020
      version: 0.5.0
  vendor: company1
`
// yaml3 has same contents as yaml1 and yaml2 but different order
// Note, array items have to be in same order
const yaml3 = `
platform:
  os:
    name: linux
    patches:
    - date: 02/02/2022
      version: 0.6.0
    - date: 01/01/2020
      version: 0.5.0
  vendor: company1
host:
  name: foo
  ip: 1.2.3.4
name: base
`

// Test the YamlEq function so we know that works
// the YAMLEq function doesn't consider 2 lists to be equal if they are not in the same order
func TestYamlEq(t *testing.T) {
	assert := assert.New(t)
	assert.YAMLEq(yaml1, yaml2, "yaml should be equal")
	assert.YAMLEq(yaml1, yaml3, "yaml should be equal")
}

func TestProfiles(t *testing.T) {
	tests := []struct {
		name     string
		base     string
		overlay  string
		expected string
	}{
		{
			name: "1",
			base:    "./testdata/dev.yaml",
			overlay:  "./testdata/managed.yaml",
			expected: "./testdata/managed.yaml",
		},
		{
			name: "2",
			base:    "./testdata/managed.yaml",
			overlay:  "./testdata/keycloak.yaml",
			expected: "./testdata/managed_with_keycloak.yaml",
		},
		{
			name: "3",
			base:    "./testdata/cert_base.yaml",
			overlay:  "./testdata/cert_overlay.yaml",
			expected: "./testdata/cert_merged.yaml",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			merged, err := MergeFiles(test.base, test.overlay, vzcr.Verrazzano{})
			assert.NoError(err, merged, "error merging profiles")
			expected, err := ioutil.ReadFile(filepath.Join(test.expected))
			assert.NoError(err, merged, "error reading profiles")
			assert.YAMLEq(merged, string(expected), "merged profile is incorrect ")
		})
	}
}


const patches1 = `
patches:
- date: 02/02/2022
  version: 0.6.0
`

const patches2 = `
patches:
- date: 01/01/2020
  version: 0.5.0
`

const patchesMerged = `
patches:
- date: 01/01/2020
  version: 0.5.0
- date: 02/02/2022
  version: 0.6.0
`

// Test the profile merge
func TestPatchesReplace(t *testing.T) {
	assert := assert.New(t)
	merged, err := MergeString(patches1, patches2, patchesReplace{})
	assert.NoError(err, merged, "error merging patches")
	assert.YAMLEq(merged, string(patches2), "merged profile is incorrect ")
}

// Test the profile merge
func TestPatchesMerge(t *testing.T) {
	assert := assert.New(t)
	merged, err := MergeString(patches1, patches2, patchesMerge{})
	assert.NoError(err, merged, "error merging patches")
	assert.YAMLEq(merged, string(patchesMerged), "merged profile is incorrect ")
}
