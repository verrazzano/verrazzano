// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Patch specifies a compoent patch
type Patch struct {
	Version string `json:"version"`
	Date    string `json:"date"`
}

// simple struct is used for trivial merges with no nested fields
type simple struct {
}

// nested1 struct is used for testing nested merges
type nested1 struct {
	Name string `json:"name"`
	Host struct {
		Name string `json:"name"`
		IP   string `json:"IP"`
	}
}

// nested2 struct is used for testing nested merges
type nested2 struct {
	Name string `json:"name"`
	Host struct {
		Name string `json:"name"`
		IP   string `json:"IP"`
	}
	Platform struct {
		Vendor string `json:"vendor"`
		OS     struct {
			Name    string  `json:"name"`
			Patches []Patch `json:"patches" patchStrategy:"merge" patchMergeKey:"version"`
		}
	}
}

// patchesReplace specifies that the array will be replaced on a merge
type patchesReplace struct {
	Patches []Patch `json:"patches"`
}

// patchesMerge specifies that the array will merge
type patchesMerge struct {
	Patches []Patch `json:"patches" patchStrategy:"merge" patchMergeKey:"version"`
}

// simpleBase is the base of a non-nested merge
const simpleBase = `name: base`

// simpleOverlay is the overlay of a non-nested merge
const simpleOverlay = `name: overlay`

// simpleOverlay is the result of non-nested merge
const simpleMerged = `name: overlay`

// TestMergeSimple tests the StrategicMerge function
// GIVEN a set of non-nested YAML strings
// WHEN StrategicMerge is called
// THEN ensure that the merged result is correct.
func TestMergeSimple(t *testing.T) {
	assert := assert.New(t)
	merged, err := StrategicMerge(simple{}, simpleBase, simpleOverlay)
	assert.NoError(err, merged, "error merging simple yaml")
	assert.YAMLEq(simpleMerged, merged, "simple yaml should be equal")
}

// nested1Base is the base of a nested merge
const nested1Base = `
name: base
host:
  name: example.com
`

// nested1Overlay is the overlay of a nested merge
const nested1Overlay = `
name: overlay
host:
  ip: 1.2.3.4
`

// nested1Merged is the result of a nested merge
const nested1Merged = `
name: overlay
host:
  name: example.com
  ip: 1.2.3.4`

// TestMergeNested1 tests the StrategicMerge function with nested YAML
// GIVEN a set of nested YAML strings
// WHEN StrategicMerge is called
// THEN ensure that the merged result is correct.
func TestMergeNested1(t *testing.T) {
	assert := assert.New(t)
	merged, err := StrategicMerge(nested1{}, nested1Base, nested1Overlay)
	assert.NoError(err, merged, "error merging nested yaml")
	assert.YAMLEq(nested1Merged, merged, "nested yaml should be equal")
}

// nested2Base is the base of a nested merge with a list
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

// nested2Base is the overlay of a nested merge with a list
const nested2Overlay = `
platform:
  os:
    patches:
    - version: 0.6.0
      date: 02/02/2022
`

// nested2Merged is the result of a nested merge with a list
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

// TestMergeNested2 tests the StrategicMerge function with nested YAML
// GIVEN a set of nested YAML strings with embedded lists
// WHEN StrategicMerge is called
// THEN ensure that the merged result is correct.
func TestMergeNested2(t *testing.T) {
	assert := assert.New(t)
	merged, err := StrategicMerge(nested2{}, nested2Base, nested2Overlay)
	assert.NoError(err, merged, "error merging nested yaml")
	assert.YAMLEq(nested2Merged, merged, "nested yaml should be equal")
}

// patches1 specifies a list of patches
const patches1 = `
patches:
- date: 01/01/2022
  version: 0.1.0
`

// patches2 specifies a list of patches
const patches2 = `
patches:
- date: 02/02/2020
  version: 0.2.0
`

// patches3 specifies a list of patches with the same key as patches2
const patches3 = `
patches:
- date: 02/22/2022
  version: 0.2.0
`

// patches4 specifies a list of patches
const patches4 = `
patches:
- date: 03/03/2022
  version: 0.3.0
`

// patchesMerged specifies results of the merged patches
const patchesMerged = `
patches:
- date: 03/03/2022
  version: 0.3.0
- date: 02/22/2022
  version: 0.2.0
- date: 01/01/2022
  version: 0.1.0
`

// Test the profile merge
func TestPatchesReplace(t *testing.T) {
	assert := assert.New(t)
	merged, err := StrategicMerge(patchesReplace{}, patches1, patches2)
	assert.NoError(err, merged, "error merging patches")
	assert.YAMLEq(merged, string(patches2), "merged profile is incorrect ")
}

// Test the profile merge
func TestPatchesMerge(t *testing.T) {
	assert := assert.New(t)
	merged, err := StrategicMerge(patchesMerge{}, patches1, patches2, patches3, patches4)
	assert.NoError(err, merged, "error merging patches")
	assert.YAMLEq(merged, string(patchesMerged), "merged profile is incorrect ")
}

// yaml1 contains YAML that is equivalent to yaml2 and yaml3
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

// yaml2 contains YAML that is equivalent to yaml1 and yaml3
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

// yaml3 contains YAML that is equivalent to yaml1 and yaml2
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

// TestYamlEq tests the YAMLEq function
// GIVEN YAML files with identical fields, but in different order
// WHEN YAMLEq is called
// THEN ensure that the boolean is correct
// Note: the YAMLEq function doesn't consider 2 lists to be equal if they are not in the same order
func TestYamlEq(t *testing.T) {
	assert := assert.New(t)
	assert.YAMLEq(yaml1, yaml2, "yaml should be equal")
	assert.YAMLEq(yaml1, yaml3, "yaml should be equal")
}

// TestMergeFiles tests the StrategicMergeFiles function
// GIVEN an array of tests, where each tests specifies files to merge
// WHEN StrategicMergeFiles is called, with some contents being a list that should be merged
// THEN ensure that the merged result is correct.
func TestMergeFiles(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		expected string
	}{
		{
			name: "1",
			files: []string{
				"./testdata/nested2base.yaml",
				"./testdata/nested2base.yaml",
			},
			expected: "./testdata/nested2base.yaml",
		},
		{
			// This case has a list of patches that we expect to be merged
			name: "1",
			files: []string{
				"./testdata/nested2base.yaml",
				"./testdata/nested2overlay1.yaml",
				"./testdata/nested2overlay2.yaml",
			},
			expected: "./testdata/nested2merged.yaml",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			merged, err := StrategicMergeFiles(nested2{}, test.files...)
			assert.NoError(err, merged, "error merging profiles")
			expected, err := os.ReadFile(filepath.Join(test.expected))
			assert.NoError(err, merged, "error reading profiles")
			assert.YAMLEq(merged, string(expected), "merged profile is incorrect ")
		})
	}
}
