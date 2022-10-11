// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
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

// Name with list internal
const name6 = `aa.bb[0].cc`
const val6 = `val_6`
const expanded6 = `aa:
  bb:
    - cc: val_6`

// Name with multiple list internal
const name7 = `aa[0].bb[0].cc`
const val7 = `val_7`
const expanded7 = `aa:
  - bb:
      - cc: val_7`

// Final object list
const name8 = `aa[0].bb[0].cc[0]`
const val8 = `val_8`
const expanded8 = `aa:
  - bb:
      - cc:
          - val_8`

// Escaped characters
const name9 = "aa\\.bb"
const val9 = `val_9`
const expanded9 = `aa.bb: val_9`

// Multiline value
const name10 = "aa"
const val10 = `val_10
val_10
val_10`
const expanded10 = `aa: |
  val_10
  val_10
  val_10`

// Nested value
const name11 = "aa.bb"
const val11 = `val_11
val_11
val_11`
const expanded11 = `aa:
  bb: |
    val_11
    val_11
    val_11`

// TestExpand tests the Expand function
// GIVEN a set of dot separated names
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
		{
			testName:  "6",
			name:      name6,
			forceList: false,
			values:    []string{val6},
			expected:  expanded6,
		},
		{
			testName:  "7",
			name:      name7,
			forceList: false,
			values:    []string{val7},
			expected:  expanded7,
		},
		{
			testName:  "8",
			name:      name8,
			forceList: false,
			values:    []string{val8},
			expected:  expanded8,
		},
		{
			testName:  "9",
			name:      name9,
			forceList: false,
			values:    []string{val9},
			expected:  expanded9,
		},
		{
			testName:  "10",
			name:      name10,
			forceList: false,
			values:    []string{val10},
			expected:  expanded10,
		},
		{
			testName:  "11",
			name:      name11,
			forceList: false,
			values:    []string{val11},
			expected:  expanded11,
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
// GIVEN a set of dot separated names
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
