// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package yaml

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const name_1 = `aa`
const val_1 = `val_1`
const expanded_1 = `aa: val_1`

const name_2 = `aa.bb`
const val_2 = `val_2`
const expanded_2 = `aa:
  bb: val_2`

const name_3 = `aa.bb."cc\.dd"`
const val_3 = `val_3`
const expanded_3 = `aa:
  bb:
    cc.dd: val_3`

// TestExpand tests the Expand function
// GIVEN a set of dot seperated names
// WHEN Expand is called
// THEN ensure that the expanded result is correct.
func TestExpand(t *testing.T) {
	const indent = 2

	tests := []struct {
		testName string
		name     string
		value    string
		expected string
	}{
		{
			testName: "1",
			name:     name_1,
			value:    val_1,
			expected: expanded_1,
		},
		{
			testName: "2",
			name:     name_2,
			value:    val_2,
			expected: expanded_2,
		},
		{
			testName: "3",
			name:     name_3,
			value:    val_3,
			expected: expanded_3,
		},
	}
	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			assert := assert.New(t)
			s, err := Expand(test.name, test.value, indent)
			assert.NoError(err, s, "error merging profiles")
			assert.Equal(test.expected, s, "Result does not match expected value")
		})
	}
}
