// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

//const (
//	userNodePoolOverrideJSON = `
//{
//  "openSearchCluster": {
//    "nodePools": [
//      {
//        "component": "es-master",
//        "diskSize": "100Gi"
//      }
//    ]
//  }
//}
//`
//	defaultNodePoolOverrideJSON = `
//{
//  "openSearchCluster": {
//    "nodePools": [
//      {
//        "component": "es-master",
//        "replicas": 3,
//        "diskSize": "50Gi",
//        "resources": {
//          "requests": {
//            "memory": "1.4Gi"
//          }
//        },
//        "roles": ["master"]
//      },
//      {
//        "component": "es-data",
//        "replicas": 3,
//        "diskSize": "50Gi",
//        "resources": {
//          "requests": {
//            "memory": "4.8Gi"
//          }
//        },
//        "roles": ["data"]
//      },
//      {
//        "component": "es-ingest",
//        "replicas": 1,
//        "resources": {
//          "requests": {
//            "memory": "2.5Gi"
//          }
//        },
//        "roles": ["ingest"],
//        "persistence": {
//          "emptyDir": {}
//        }
//      }
//    ]
//  }
//}
//`
//	mergedOverrides = `values:
//  openSearchCluster:
//    nodePools:
//    - component: es-master
//      diskSize: 100Gi
//      jvm: null
//      replicas: 3
//      resources:
//        requests:
//          memory: 1.4Gi
//      roles:
//      - master
//    - component: data-ingest
//      diskSize: 10Gi
//      jvm: Xvm512
//      replicas: 5
//      resources:
//        requests:
//          memory: 1Gi
//    - component: es-data
//      diskSize: 50Gi
//      replicas: 3
//      resources:
//        requests:
//          memory: 4.8Gi
//      roles:
//      - data
//    - component: es-ingest
//      persistence:
//        emptyDir: {}
//      replicas: 1
//      resources:
//        requests:
//          memory: 2.5Gi
//      roles:
//      - ingest
//`
//)

//func TestBuildNodePoolOverride(t *testing.T) {
//	fakeClient := fake.NewClientBuilder().WithScheme(newScheme()).WithRuntimeObjects().Build()
//	getControllerRuntimeClient = func() (client.Client, error) {
//		return fakeClient, nil
//	}
//
//	vz := &vzapi.Verrazzano{
//		Spec: vzapi.VerrazzanoSpec{
//			Components: vzapi.ComponentSpec{
//				Elasticsearch: &vzapi.ElasticsearchComponent{
//					Nodes: []vzapi.OpenSearchNode{
//						{
//							Name:     "es-master",
//							Replicas: common.Int32Ptr(3),
//							Roles:    []vmov1.NodeRole{"master"},
//							Storage:  &vzapi.OpenSearchNodeStorage{Size: "50Gi"},
//						},
//						{
//							Name:     "data-ingest",
//							Replicas: common.Int32Ptr(5),
//							Resources: &v1.ResourceRequirements{Requests: map[v1.ResourceName]resource.Quantity{
//								"memory": resource.MustParse("1Gi"),
//							}},
//							Roles:    []vmov1.NodeRole{"ingest", "data"},
//							JavaOpts: "Xvm512",
//							Storage:  &vzapi.OpenSearchNodeStorage{Size: "10Gi"},
//						}},
//				},
//				OpenSearchOperator: &vzapi.OpenSearchOperatorComponent{
//					InstallOverrides: vzapi.InstallOverrides{
//						ValueOverrides: []vzapi.Overrides{
//							{
//								Values: &apiextensionsv1.JSON{
//									Raw: []byte(userNodePoolOverrideJson),
//								},
//							},
//							{
//								Values: &apiextensionsv1.JSON{
//									Raw: []byte(defaultNodePoolOverrideJson),
//								},
//							},
//						},
//					},
//				},
//			},
//		},
//	}
//
//
//}

func CreateValueOverride(rawJSON []byte) ([]v1beta1.Overrides, error) {
	return []v1beta1.Overrides{
		{
			Values: &apiextensionsv1.JSON{
				Raw: rawJSON,
			}},
	}, nil
}
