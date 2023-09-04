// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusteragent

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGetWatchDescriptors tests the GetWatchDescriptors function impl for this component
// GIVEN a call to GetWatchDescriptors
//
//	WHEN a new component is created
//	THEN the watch descriptors have the correct number of watches
func TestGetWatchDescriptors(t *testing.T) {
	wd := NewComponent().GetWatchDescriptors()
	assert.Len(t, wd, 1)
}
