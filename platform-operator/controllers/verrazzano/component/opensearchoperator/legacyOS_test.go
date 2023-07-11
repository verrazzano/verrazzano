// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

//func TestGetNewClaimName(t *testing.T) {
//	var tests = []struct {
//		oldPVCName string
//		newPVCName string
//		nodePool   string
//	}{
//		{"vmi-system-es-data-1", "data-opensearch-es-data-1", "es-data"},
//		{"vmi-system-es-data", "data-opensearch-es-data-0", "es-data"},
//		{"elasticsearch-master-vmi-system-es-master-2", "data-opensearch-es-master-2", "es-master"},
//		{"vmi-system-data", "data-opensearch-data-0", "data"},
//		{"vmi-system-my-ingest-node-3", "data-opensearch-my-ingest-node-3", "my-ingest-node"},
//		{"elasticsearch-master-vmi-system-my-master-node-2", "data-opensearch-my-master-node-2", "my-master-node"},
//	}
//
//	for _, tt := range tests {
//		nodePool, newPVCName := getNewClaimName(tt.oldPVCName)
//		assert.Equal(t, tt.newPVCName, newPVCName)
//		assert.Equal(t, tt.nodePool, nodePool)
//	}
//}

// TestGetNodeNameFromClaimName tests the getNodeNameFromClaimName function
// GIVEN a list of OpenSearch nodes and claim names
// WHEN getNodeNameFromClaimName is called
// THEN expected node name is returned for each claim name
func TestGetNodeNameFromClaimName(t *testing.T) {
	var tests = []struct {
		nodes            []vzapi.OpenSearchNode
		claimNames       []string
		expectedNodeName []string
	}{
		{
			[]vzapi.OpenSearchNode{{Name: "es-data"}, {Name: "es-data1"}},
			[]string{"vmi-system-es-data", "vmi-system-es-data1-1", "vmi-system-es-data-tqxkq", "vmi-system-es-data1-1-8m66v"},
			[]string{"es-data", "es-data1", "es-data", "es-data1"},
		},
		{
			[]vzapi.OpenSearchNode{{Name: "es-data"}, {Name: "es-data1"}},
			[]string{"vmi-system-es-data1", "vmi-system-es-data-1", "vmi-system-es-data1-tqxkq", "vmi-system-es-data-1-8m66v"},
			[]string{"es-data1", "es-data", "es-data1", "es-data"},
		},
		{
			[]vzapi.OpenSearchNode{{Name: "es-master"}},
			[]string{"elasticsearch-master-vmi-system-es-master-0"},
			[]string{"es-master"},
		},
	}

	for _, tt := range tests {
		for i := range tt.claimNames {
			nodePool := getNodeNameFromClaimName(tt.claimNames[i], tt.nodes)
			assert.Equal(t, tt.expectedNodeName[i], nodePool)
		}
	}
}
