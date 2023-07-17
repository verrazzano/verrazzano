// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"github.com/stretchr/testify/assert"
	"testing"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"

	v1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	userNodePoolOverrideJson = `
{
  "opensearchCluster": {
    "nodePools": [
      {
        "component": "es-master",
        "diskSize": "100Gi"
      }
    ]
  }
}
`
	defaultNodePoolOverrideJson = `
{
  "opensearchCluster": {
    "nodePools": [
      {
        "component": "es-master",
        "replicas": 3,
        "diskSize": "50Gi",
        "resources": {
          "requests": {
            "memory": "1.4Gi"
          }
        },
        "roles": ["master"]
      },
      {
        "component": "es-data",
        "replicas": 3,
        "diskSize": "50Gi",
        "resources": {
          "requests": {
            "memory": "4.8Gi"
          }
        },
        "roles": ["data"]
      },
      {
        "component": "es-ingest",
        "replicas": 1,
        "resources": {
          "requests": {
            "memory": "2.5Gi"
          }
        },
        "roles": ["ingest"],
        "persistence": {
          "emptyDir": {}
        }
      }
    ]
  }
}
`
	mergedOverrides = `
{
  "opensearchCluster": {
    "nodePools": [
      {
        "component": "es-master",
        "replicas": 3,
        "diskSize": "100Gi",
        "resources": {
          "requests": {
            "memory": "1.4Gi"
          }
        },
        "roles": ["master"]
      },
      {
        "component": "es-data",
        "replicas": 3,
        "diskSize": "50Gi",
        "resources": {
          "requests": {
            "memory": "4.8Gi"
          }
        },
        "roles": ["data"]
      },
      {
        "component": "data-ingest",
        "replicas": 5,
        "diskSize": "10Gi",
        "resources": {
          "requests": {
            "memory": "1Gi"
          }
        },
		"jvm": "Xvm512"
        "roles": ["data", "ingest"]
      },
      {
        "component": "es-ingest",
        "replicas": 1,
        "resources": {
          "requests": {
            "memory": "2.5Gi"
          }
        },
        "roles": ["ingest"],
        "persistence": {
          "emptyDir": {}
        }
      }
    ]
  }
}
`
)

func TestBuildNodePoolOverride(t *testing.T) {
	getControllerRuntimeClient = func() (client.Client, error) {
		return fake.NewClientBuilder().WithScheme(newScheme()).WithRuntimeObjects().Build(), nil
	}

	vz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					Nodes: []vzapi.OpenSearchNode{
						{
							Name:     "es-master",
							Replicas: common.Int32Ptr(3),
							Roles:    []vmov1.NodeRole{"master"},
							Storage:  &vzapi.OpenSearchNodeStorage{Size: "50Gi"},
						},
						{
							Name:     "data-ingest",
							Replicas: common.Int32Ptr(5),
							Resources: &v1.ResourceRequirements{Requests: map[v1.ResourceName]resource.Quantity{
								"memory": resource.MustParse("1Gi"),
							}},
							Roles:    []vmov1.NodeRole{"ingest", "data"},
							JavaOpts: "Xvm512",
							Storage:  &vzapi.OpenSearchNodeStorage{Size: "10Gi"},
						}},
				},
				OpenSearchOperator: &vzapi.OpenSearchOperatorComponent{
					InstallOverrides: vzapi.InstallOverrides{
						ValueOverrides: []vzapi.Overrides{
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(userNodePoolOverrideJson),
								},
							},
							{
								Values: &apiextensionsv1.JSON{
									Raw: []byte(defaultNodePoolOverrideJson),
								},
							},
						},
					},
				},
			},
		},
	}

	actualOverrides := GetOverrides(vz).([]vzapi.Overrides)
	expectedOverrides, _ := CreateValueOverride([]byte(mergedOverrides))
	assert.Equal(t, expectedOverrides, actualOverrides[0])
}

func CreateValueOverride(rawJSON []byte) (vzapi.Overrides, error) {
	return vzapi.Overrides{

		Values: &apiextensionsv1.JSON{
			Raw: rawJSON,
		},
	}, nil
}
