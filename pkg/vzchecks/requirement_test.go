// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vzchecks

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

// TestGetVZRequirement tests getting Verrazzano requirement for various profiles
func TestGetVZRequirement(t *testing.T) {
	var tests = []struct {
		profile ProfileType
		//nodeCount int
		cpu     string
		memory  string
		storage string
	}{
		{Dev, "2", "16G", "100G"},
		{Prod, "4", "32G", "100G"},
		{ManagedCluster, "4", "32G", "100G"},
		{"", "4", "32G", "100G"},
	}

	for _, tt := range tests {
		t.Run(string(tt.profile), func(t *testing.T) {
			vzReq := getVZRequirement(tt.profile)
			//assert.Equal(t, tt.nodeCount, vzReq.nodeCount)
			assert.Equal(t, tt.cpu, vzReq.cpu.allocatable.String())
			assert.Equal(t, tt.memory, vzReq.memory.allocatable.String())
			assert.Equal(t, tt.storage, vzReq.ephemeralStorage.allocatable.String())
		})
	}
}

// TestUnspecifiedProfileRequirement tests getting Verrazzano requirement for unspecified profile
func TestUnspecifiedProfileRequirement(t *testing.T) {
	vzReq := getVZRequirement("unspecified")
	assert.Equal(t, vzReq, VZRequirement{})
}
