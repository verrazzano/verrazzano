// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Simple name/value
const name1 = `aa`
const val1 = `val_1`
const expanded1 = `aa: val_1`

// Two level name/value
const name2 = `aa.bb`
const val2 = `val_2`
const expanded2 = `aa:
  bb: val_2`

// Two level name/value with quoted final value segment
const name3 = `aa.bb."cc\.dd"`
const val3 = `val_3`
const expanded3 = `aa:
  bb:
    cc.dd: val_3`

// Name value with valuelist
const name4 = `aa.bb`
const val4a = `val_4a`
const val4b = `val_4b`
const val4c = `val_4c`
const expanded4 = `aa:
  bb:
  - val_4a
  - val_4b
  - val_4c`

// Name value with valuelist
const name5 = `aa.bb`
const val5 = `val_5a`
const expanded5 = `aa:
  bb:
  - val_5a`

// TestExpand tests the Expand function
// GIVEN a set of dot seperated names
// WHEN Expand is called
// THEN ensure that the expanded result is correct.
func TestExpand(t *testing.T) {
	tests := []struct {
		testName  string
		name      string
		forceList bool
		values    []string
		expected  string
	}{
		{
			testName:  "1",
			name:      name1,
			forceList: false,
			values:    []string{val1},
			expected:  expanded1,
		},
		{
			testName:  "2",
			name:      name2,
			forceList: false,
			values:    []string{val2},
			expected:  expanded2,
		},
		{
			testName:  "3",
			name:      name3,
			forceList: false,
			values:    []string{val3},
			expected:  expanded3,
		},
		{
			testName:  "4",
			name:      name4,
			forceList: false,
			values:    []string{val4a, val4b, val4c},
			expected:  expanded4,
		},
		{
			testName:  "5",
			name:      name5,
			forceList: true,
			values:    []string{val5},
			expected:  expanded5,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			assert := assert.New(t)
			s, err := Expand(0, test.forceList, test.name, test.values...)
			assert.NoError(err, s, "error merging profiles")
			assert.Equal(test.expected, s, "Result does not match expected value")
		})
	}
}

// Expanded results with a left margin of 4
const lmExpanded4 = `    aa:
      bb:
      - val_4a
      - val_4b
      - val_4c`

// TestLeftMargin tests the Expand function
// GIVEN a set of dot seperated names
// WHEN Expand is called with a non-zero left margin
// THEN ensure that the expanded result is correct.
func TestLeftMargin(t *testing.T) {
	tests := []struct {
		testName string
		name     string
		values   []string
		expected string
	}{
		{
			testName: "4",
			name:     name4,
			values:   []string{val4a, val4b, val4c},
			expected: lmExpanded4,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			assert := assert.New(t)
			s, err := Expand(4, false, test.name, test.values...)
			assert.NoError(err, s, "error merging profiles")
			assert.Equal(test.expected, s, "Result does not match expected value")
		})
	}
}
