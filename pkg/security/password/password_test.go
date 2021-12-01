// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package password

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

// TestGeneratePassword tests generating random passwords
// GIVEN a call to GeneratePassword
//  WHEN the deployment object does NOT have enough replicas available
//  THEN false is returned
func TestGeneratePassword(t *testing.T) {
	var tests = []struct {
		length   int
		hasError bool
	}{
		{-1, true},
		{10, false},
		{15, false},
		{31, false},
		{66, false},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.length), func(t *testing.T) {
			pw, err := GeneratePassword(tt.length)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.length, len(pw))
			}
		})
	}
}
