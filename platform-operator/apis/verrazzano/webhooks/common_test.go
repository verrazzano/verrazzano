// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package webhooks

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestMinVersion tests validates combinations vzVersion and minVersion
func TestMinVersion(t *testing.T) {
	assert.False(t, isMinVersion("1.5.0", ""), "Correct, minVersion cannot be empty")
	assert.True(t, isMinVersion("1.5.0", "1.4.0"), "Correct, minVersion is less than vzVersion")
	assert.False(t, isMinVersion("", ""), "Correct, cannot be empty")
	assert.False(t, isMinVersion("", "1.4.0"), "Correct, vzVersion cannot be empty")
	assert.False(t, isMinVersion("1.4.0", "1.5.0"), "Correct, vzVersion is less than minVersion")
}
