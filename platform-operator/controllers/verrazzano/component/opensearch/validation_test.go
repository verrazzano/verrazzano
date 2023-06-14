// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
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
		Replicas: &replicas,
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
			v1beta1vz := &v1beta1.Verrazzano{}
			assert.NoError(t, tt.vz.ConvertTo(v1beta1vz))
			if err := validateNoDuplicatedConfiguration(v1beta1vz); (err != nil) != tt.hasError {
				t.Errorf("validateNoDuplicatedConfiguration() error = %v, hasError: %v", err, tt.hasError)
			}
		})
	}
}
