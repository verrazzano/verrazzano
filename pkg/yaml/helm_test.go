// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/verrazzano/verrazzano/pkg/bom"
)

const oneOverrideExpect = `name: test-name
`
const oneListOverrideExpect = `name:
- test-name
`
const oneListNestedExpect = `name:
- name: test-name
`
const multipleListNestedExpect = `name:
- name1: test-name
  name2: test-name
  name3: test-name
`
const multilineExpect = `name: |-
  name1
  name2
  name3
`
const multipleListExpect = `name:
- name1: test-name
- name2: test-name
- name3: test-name
`
const escapedExpect = `name.name/name: name
`

const hyphenName = `name:
- name-name: name
`

// TestExpand tests the Expand function
// GIVEN a set of dot separated names
// WHEN Expand is called
// THEN ensure that the expanded result is correct.
func TestHelmValueFileConstructor(t *testing.T) {
	tests := []struct {
		name        string
		kvs         []bom.KeyValue
		expected    string
		expectError bool
	}{
		{
			name:        "test no overrides",
			kvs:         []bom.KeyValue{},
			expected:    "{}\n",
			expectError: false,
		},
		{
			name: "test one override",
			kvs: []bom.KeyValue{
				{Key: "name", Value: "test-name"},
			},
			expected:    oneOverrideExpect,
			expectError: false,
		},
		{
			name: "test one list override",
			kvs: []bom.KeyValue{
				{Key: "name[0]", Value: "test-name"},
			},
			expected:    oneListOverrideExpect,
			expectError: false,
		},
		{
			name: "test nested list",
			kvs: []bom.KeyValue{
				{Key: "name[0].name", Value: "test-name"},
			},
			expected:    oneListNestedExpect,
			expectError: false,
		},
		{
			name: "test multiple list updates",
			kvs: []bom.KeyValue{
				{Key: "name[0].name1", Value: "test-name"},
				{Key: "name[0].name2", Value: "test-name"},
				{Key: "name[0].name3", Value: "test-name"},
			},
			expected:    multipleListNestedExpect,
			expectError: false,
		},
		{
			name: "test multiline",
			kvs: []bom.KeyValue{
				{Key: "name", Value: "name1\nname2\nname3"},
			},
			expected:    multilineExpect,
			expectError: false,
		},
		{
			name: "test array index",
			kvs: []bom.KeyValue{
				{Key: "name[0].name1", Value: "test-name"},
				{Key: "name[1].name2", Value: "test-name"},
				{Key: "name[2].name3", Value: "test-name"},
			},
			expected:    multipleListExpect,
			expectError: false,
		},
		{
			name: "test escaped chars",
			kvs: []bom.KeyValue{
				{Key: `name\.name/name`, Value: "name"},
			},
			expected:    escapedExpect,
			expectError: false,
		},
		{
			name: "test hyphen key",
			kvs: []bom.KeyValue{
				{Key: `name[0].name-name`, Value: "name"},
			},
			expected:    hyphenName,
			expectError: false,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)
			s, err := HelmValueFileConstructor(test.kvs)
			if test.expectError {
				assert.Error(err, s, "expected error not found")
			} else {
				assert.NoError(err, s, "error merging profiles")
			}

			assert.Equal(test.expected, string(s), "Result does not match expected value")
		})
	}
}
