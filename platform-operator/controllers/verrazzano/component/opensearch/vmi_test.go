// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearch

import (
	"context"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var enabled = true

var vmiEnabledCR = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Profile: vzapi.Prod,
		Components: vzapi.ComponentSpec{
			DNS: dnsComponents.DNS,
			Ingress: &vzapi.IngressNginxComponent{
				Enabled: getBoolPtr(true),
			},
			Kibana: &vzapi.KibanaComponent{
				Enabled: &enabled,
			},
			Prometheus: &vzapi.PrometheusComponent{
				Enabled: &enabled,
			},
			Grafana: &vzapi.GrafanaComponent{
				Enabled: &enabled,
			},
			Elasticsearch: &vzapi.ElasticsearchComponent{

				ESInstallArgs: []vzapi.InstallArgs{
					{
						Name:  "nodes.master.replicas",
						Value: "1",
					},
					{
						Name:  "nodes.master.requests.memory",
						Value: "1G",
					},
					{
						Name:  "nodes.ingest.replicas",
						Value: "2",
					},
					{
						Name:  "nodes.ingest.requests.memory",
						Value: "2G",
					},
					{
						Name:  "nodes.data.replicas",
						Value: "3",
					},
					{
						Name:  "nodes.data.requests.memory",
						Value: "3G",
					},
					{
						Name:  "nodes.data.requests.storage",
						Value: "100Gi",
					},
				},
			},
		},
	},
}

// TestNewVMIResources tests that new VMI resources can be created from a CR
// GIVEN a Verrazzano CR
//  WHEN I create new VMI resources
//  THEN the configuration in the CR is respected
func TestNewVMIResources(t *testing.T) {
	r := &common.ResourceRequestValues{
		Memory:  "",
		Storage: "50Gi",
	}

	opensearch, err := newOpenSearch(&vmiEnabledCR, nil, r, nil, true, false)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, opensearch.MasterNode.Replicas)
	assert.EqualValues(t, 2, opensearch.IngestNode.Replicas)
	assert.EqualValues(t, 3, opensearch.DataNode.Replicas)
	assert.Equal(t, "100Gi", opensearch.DataNode.Storage.Size)
	assert.Equal(t, "50Gi", opensearch.MasterNode.Storage.Size)
}

// TestOpenSearchInvalidArgs tests trying to create an OpenSearch resource with invalid args
// GIVEN a Verrazzano CR with invalid install args
//  WHEN I create a new OpenSearch resource
//  THEN the OpenSearch resource fails to create
func TestOpenSearchInvalidArgs(t *testing.T) {
	r := &common.ResourceRequestValues{}
	crBadArgs := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					ESInstallArgs: []vzapi.InstallArgs{
						{
							Name:  "nodes.master.replicas",
							Value: "foobar!",
						},
					},
				},
			},
		},
	}

	_, err := newOpenSearch(crBadArgs, nil, r, nil, false, false)
	assert.Error(t, err)
}

// TestNewOpenSearchValuesAreCopied tests that VMI and policy values are copied over to the new OpenSearch
// GIVEN a Verrazzano CR and an existing VMI
//  WHEN I create a new OpenSearch resource
//  THEN the storage options from the existing VMi are preserved, and any policy values are copied.
func TestNewOpenSearchValuesAreCopied(t *testing.T) {
	age := "1d"
	r := &common.ResourceRequestValues{}
	testvz := &vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Components: vzapi.ComponentSpec{
				Elasticsearch: &vzapi.ElasticsearchComponent{
					ESInstallArgs: []vzapi.InstallArgs{},
					Policies: []vmov1.IndexManagementPolicy{
						{
							PolicyName:   "my-policy",
							IndexPattern: "pattern",
							MinIndexAge:  &age,
						},
					},
				},
			},
		},
	}
	pvcs := []string{"p1", "p2"}
	testvmi := &vmov1.VerrazzanoMonitoringInstance{
		Spec: vmov1.VerrazzanoMonitoringInstanceSpec{
			Elasticsearch: vmov1.Elasticsearch{
				MasterNode: vmov1.ElasticsearchNode{
					Replicas: 1,
				},
				DataNode: vmov1.ElasticsearchNode{
					Replicas: 1,
				},
				Storage: vmov1.Storage{
					Size:     "1Gi",
					PvcNames: pvcs,
				},
			},
		},
	}

	openSearch, err := newOpenSearch(testvz, nil, r, testvmi, false, false)
	assert.NoError(t, err)
	assert.Equal(t, "1Gi", openSearch.MasterNode.Storage.Size)
	assert.EqualValues(t, testvz.Spec.Components.Elasticsearch.Policies, openSearch.Policies)
	assert.EqualValues(t, pvcs, openSearch.DataNode.Storage.PvcNames)
	assert.Nil(t, openSearch.MasterNode.Storage.PvcNames)
}

// TestCreateOrUpdateVMI tests a new VMI resources is created in K8s according to the CR
// GIVEN a Verrazzano CR
// WHEN I create a new VMI resource
//  THEN the configuration in the CR is respected
func TestCreateOrUpdateVMI(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), &vmiEnabledCR, nil, false)
	err := common.CreateOrUpdateVMI(ctx, updateFunc)
	assert.NoError(t, err)
	vmi := &vmov1.VerrazzanoMonitoringInstance{}
	namespacedName := types.NamespacedName{Name: system, Namespace: globalconst.VerrazzanoSystemNamespace}
	err = ctx.Client().Get(context.TODO(), namespacedName, vmi)
	assert.NoError(t, err)
	assert.Equal(t, "vmi.system.default.blah", vmi.Spec.URI)
	assert.Equal(t, "verrazzano-ingress.default.blah", vmi.Spec.IngressTargetDNSName)
	assert.Equal(t, "100Gi", vmi.Spec.Elasticsearch.DataNode.Storage.Size)
	assert.EqualValues(t, 2, vmi.Spec.Elasticsearch.IngestNode.Replicas)
	assert.EqualValues(t, 1, vmi.Spec.Elasticsearch.MasterNode.Replicas)
	assert.EqualValues(t, 3, vmi.Spec.Elasticsearch.DataNode.Replicas)
}

// TestCreateOrUpdateVMINoNGINX tests a new VMI resources is created in K8s according to the CR
// GIVEN a Verrazzano CR
// WHEN I create a new VMI resource and NGINX is not enabled
//  THEN the configuration in the CR is respected
func TestCreateOrUpdateVMINoNGINX(t *testing.T) {
	vmiEnabledCR.Spec.Components.Ingress.Enabled = getBoolPtr(false)
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), &vmiEnabledCR, nil, false)
	err := common.CreateOrUpdateVMI(ctx, updateFunc)
	assert.NoError(t, err)
	vmi := &vmov1.VerrazzanoMonitoringInstance{}
	namespacedName := types.NamespacedName{Name: system, Namespace: globalconst.VerrazzanoSystemNamespace}
	err = ctx.Client().Get(context.TODO(), namespacedName, vmi)
	assert.NoError(t, err)
	assert.Empty(t, vmi.Spec.URI)
	assert.Empty(t, vmi.Spec.IngressTargetDNSName)
	assert.Equal(t, "100Gi", vmi.Spec.Elasticsearch.DataNode.Storage.Size)
	assert.EqualValues(t, 2, vmi.Spec.Elasticsearch.IngestNode.Replicas)
	assert.EqualValues(t, 1, vmi.Spec.Elasticsearch.MasterNode.Replicas)
	assert.EqualValues(t, 3, vmi.Spec.Elasticsearch.DataNode.Replicas)
}

// TestHasDataNodeStorageOverride tests the detection of data node storage overrides
// GIVEN a Verrazzano CR
// WHEN I check for data node storage overrides
//  THEN hasNodeStorageOverride returns true or false depending on the CR values
func TestHasDataNodeStorageOverride(t *testing.T) {
	var tests = []struct {
		name        string
		cr          *vzapi.Verrazzano
		hasOverride bool
	}{
		{
			"no override when not enabled",
			&vzapi.Verrazzano{},
			false,
		},
		{
			"detects override when override is used",
			&vmiEnabledCR,
			true,
		},
		{
			"no detected override when none are present",
			&vzapi.Verrazzano{
				Spec: vzapi.VerrazzanoSpec{
					Components: vzapi.ComponentSpec{
						Elasticsearch: &vzapi.ElasticsearchComponent{
							Enabled: &enabled},
					},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.hasOverride, hasNodeStorageOverride(tt.cr, "nodes.data.requests.storage"))
		})
	}
}

func TestNodeAdapter(t *testing.T) {
	vmiStorage := "50Gi"
	vmi := &vmov1.VerrazzanoMonitoringInstance{
		Spec: vmov1.VerrazzanoMonitoringInstanceSpec{
			Elasticsearch: vmov1.Elasticsearch{
				Enabled: true,
				Nodes: []vmov1.ElasticsearchNode{
					{
						Name:     "a",
						Replicas: 3,
						Storage: &vmov1.Storage{
							Size: vmiStorage,
						},
						Roles: []vmov1.NodeRole{
							vmov1.MasterRole,
						},
						Resources: vmov1.Resources{
							RequestMemory: "48Mi",
						},
					},
					{
						Name:     "b",
						Replicas: 2,
						Storage: &vmov1.Storage{
							Size: "100Gi",
							PvcNames: []string{
								"1", "2",
							},
						},
						Roles: []vmov1.NodeRole{
							vmov1.DataRole,
							vmov1.IngestRole,
						},
						Resources: vmov1.Resources{
							RequestMemory: "48Mi",
						},
					},
				},
			},
		},
	}
	bNode := vzapi.OpenSearchNode{
		Name:     "b",
		Replicas: 2,
		Roles: []vmov1.NodeRole{
			vmov1.DataRole,
			vmov1.IngestRole,
		},
		Resources: &corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				"memory": resource.MustParse("48Mi"),
			},
		},
		Storage: &vzapi.OpenSearchNodeStorage{
			Size: "100Gi",
		},
	}
	nodes := []vzapi.OpenSearchNode{
		{
			Name:     "a",
			Replicas: 3,
			Roles: []vmov1.NodeRole{
				vmov1.MasterRole,
			},
			Resources: &corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"memory": resource.MustParse("48Mi"),
				},
			},
		},
		bNode,
	}

	adaptedNodes := nodeAdapter(vmi, nodes, map[string]vzapi.OpenSearchNode{bNode.Name: bNode}, &common.ResourceRequestValues{Storage: vmiStorage})
	compareNodes := func(n1, n2 *vmov1.ElasticsearchNode) {
		assert.Equal(t, n1.Name, n2.Name)
		assert.Equal(t, n1.Replicas, n2.Replicas)
		assert.EqualValues(t, n1.Roles, n2.Roles)
		if n1.Storage != nil {
			assert.NotNil(t, n2.Storage)
			assert.Equal(t, n1.Storage.Size, n2.Storage.Size)
		}
		assert.Equal(t, n1.Resources.RequestMemory, n2.Resources.RequestMemory)
	}
	compareNodes(&vmi.Spec.Elasticsearch.Nodes[0], &adaptedNodes[0])
	compareNodes(&vmi.Spec.Elasticsearch.Nodes[1], &adaptedNodes[1])
}
