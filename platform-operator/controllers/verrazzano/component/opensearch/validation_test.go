// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

var (
	emptyComponent = createVZ(&vzapi.ElasticsearchComponent{})
)

func createInstallArgs(m, d, i int) []vzapi.InstallArgs {
	return []vzapi.InstallArgs{
		{
			Name:  "nodes.master.replicas",
			Value: fmt.Sprintf("%d", m),
		},
		{
			Name:  "nodes.data.replicas",
			Value: fmt.Sprintf("%d", d),
		},
		{
			Name:  "nodes.ingest.replicas",
			Value: fmt.Sprintf("%d", i),
		},
	}
}

func createVZ(opensearch *vzapi.ElasticsearchComponent) *vzapi.Verrazzano {
	enabled := true
	opensearch.Enabled = &enabled
	return &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: opensearch,
			},
		},
	}
}

func createNG(name string, replicas int32, roles []vmov1.NodeRole) vzapi.OpenSearchNode {
	return vzapi.OpenSearchNode{
		Name:     name,
		Replicas: replicas,
		Roles:    roles,
	}
}

func TestValidateNoDuplicatedConfiguration(t *testing.T) {
	var tests = []struct {
		name     string
		vz       *vzapi.Verrazzano
		hasError bool
	}{
		{
			"error when node group uses reserved name",
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					createNG("es-master", 1, nil),
				},
			}),
			true,
		},
		{
			"no duplication when component has no args",
			emptyComponent,
			false,
		},
		{
			"no duplication when component does not duplicate args",
			createVZ(&vzapi.ElasticsearchComponent{
				ESInstallArgs: createInstallArgs(3, 3, 3),
				Nodes: []vzapi.OpenSearchNode{
					createNG("a", 1, nil),
					createNG("b", 2, nil),
					createNG("c", 3, nil),
				},
			}),
			false,
		},
		{
			"duplication when component has InstallArgs with the same name",
			createVZ(&vzapi.ElasticsearchComponent{
				ESInstallArgs: []vzapi.InstallArgs{
					{
						Name: "a",
					},
					{
						Name: "b",
					},
					{
						Name: "a",
					},
				},
			}),
			true,
		},
		{
			"duplication when component has Node groups with the same name",
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					createNG("master", 3, nil),
					createNG("data", 3, nil),
					createNG("ingest", 3, nil),
					createNG("master", 3, nil),
				},
			}),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateNoDuplicatedConfiguration(tt.vz); (err != nil) != tt.hasError {
				t.Errorf("validateNoDuplicatedConfiguration() error = %v, hasError: %v", err, tt.hasError)
			}
		})
	}
}

func TestValidateClusterTopology(t *testing.T) {
	ngThreeMasters := createNG("m", 3, []vmov1.NodeRole{vmov1.MasterRole})
	ngThreeData := createNG("d", 3, []vmov1.NodeRole{vmov1.DataRole})
	threeNodeMaster := createVZ(&vzapi.ElasticsearchComponent{
		Nodes: []vzapi.OpenSearchNode{
			ngThreeMasters,
		},
	})
	fourNodeMaster := createVZ(&vzapi.ElasticsearchComponent{
		Nodes: []vzapi.OpenSearchNode{
			createNG("m", 4, []vmov1.NodeRole{vmov1.MasterRole}),
		},
	})
	oneNodeMaster := createVZ(&vzapi.ElasticsearchComponent{
		Nodes: []vzapi.OpenSearchNode{
			createNG("m", 1, []vmov1.NodeRole{vmov1.MasterRole}),
		},
	})
	var tests = []struct {
		name     string
		old      *vzapi.Verrazzano
		new      *vzapi.Verrazzano
		hasError bool
	}{
		{
			"can scale 0 master to 1",
			emptyComponent,
			oneNodeMaster,
			false,
		},
		{
			"can scale 1 master to 3",
			oneNodeMaster,
			threeNodeMaster,
			false,
		},
		{
			"cannot scale 3 masters to 1",
			threeNodeMaster,
			oneNodeMaster,
			true,
		},
		{
			"can scale up from 0 to 3 each of data and master nodes",
			emptyComponent,
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					ngThreeMasters,
					ngThreeData,
				},
			}),
			false,
		},
		{
			"cannot scale 4 masters to 2",
			fourNodeMaster,
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					createNG("m", 2, []vmov1.NodeRole{vmov1.MasterRole}),
				},
			}),
			true,
		},
		{
			"cannot scale 8 masters to 3",
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					createNG("m", 8, []vmov1.NodeRole{vmov1.MasterRole}),
				},
			}),
			threeNodeMaster,
			true,
		},
		{
			"can scale 3 masters to 0",
			threeNodeMaster,
			emptyComponent,
			false,
		},
		{
			"can scale 4 masters to 3",
			fourNodeMaster,
			threeNodeMaster,
			false,
		},
		{
			"can scale 4 data nodes to 3",
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					ngThreeMasters,
					createNG("d", 4, []vmov1.NodeRole{vmov1.DataRole}),
				},
			}),
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					ngThreeMasters,
					ngThreeData,
				},
			}),
			false,
		},
		{
			"cannot scale 5 data nodes to 2",
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					ngThreeMasters,
					createNG("d", 5, []vmov1.NodeRole{vmov1.DataRole}),
				},
			}),
			createVZ(&vzapi.ElasticsearchComponent{
				Nodes: []vzapi.OpenSearchNode{
					ngThreeMasters,
					createNG("d", 2, []vmov1.NodeRole{vmov1.DataRole}),
				},
			}),
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validateClusterTopology(tt.old, tt.new); (err != nil) != tt.hasError {
				t.Errorf("validateClusterTopology() error = %v, hasError: %v", err, tt.hasError)
			}
		})
	}
}

func TestNodeCount(t *testing.T) {
	os := &vzapi.ElasticsearchComponent{
		ESInstallArgs: createInstallArgs(1, 2, 3),
		Nodes: []vzapi.OpenSearchNode{
			createNG("m", 3, []vmov1.NodeRole{vmov1.MasterRole}),
			createNG("md", 3, []vmov1.NodeRole{vmov1.MasterRole, vmov1.DataRole}),
			createNG("di", 2, []vmov1.NodeRole{vmov1.DataRole, vmov1.IngestRole}),
		},
	}

	nc, err := nodeCount(createVZ(os))
	assert.NoError(t, err)
	assert.EqualValues(t, 14, nc.Replicas)
	assert.EqualValues(t, 7, nc.MasterNodes)
	assert.EqualValues(t, 7, nc.DataNodes)
	assert.EqualValues(t, 5, nc.IngestNodes)

}
