// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

var enabled = true

var monitoringComponent = vzapi.MonitoringComponent{
	Enabled: &enabled,
}

var vmiEnabledCR = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Components: vzapi.ComponentSpec{
			Kibana: &vzapi.KibanaComponent{
				MonitoringComponent: monitoringComponent,
			},
			Prometheus: &vzapi.PrometheusComponent{
				MonitoringComponent: monitoringComponent,
			},
			Grafana: &vzapi.GrafanaComponent{
				MonitoringComponent: monitoringComponent,
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
	r := &resourceRequestValues{
		Memory:  "",
		Storage: "50Gi",
	}
	grafana := newGrafana(&vmiEnabledCR, r, nil)
	assert.Equal(t, "48Mi", grafana.Resources.RequestMemory)
	assert.Equal(t, "50Gi", grafana.Storage.Size)

	prometheus := newPrometheus(&vmiEnabledCR, r, nil)
	assert.Equal(t, "128Mi", prometheus.Resources.RequestMemory)
	assert.Equal(t, "50Gi", prometheus.Storage.Size)

	opensearch, err := newOpenSearch(&vmiEnabledCR, r, nil)
	assert.NoError(t, err)
	assert.EqualValues(t, 1, opensearch.MasterNode.Replicas)
	assert.EqualValues(t, 2, opensearch.IngestNode.Replicas)
	assert.EqualValues(t, 3, opensearch.DataNode.Replicas)
	assert.Equal(t, "100Gi", opensearch.Storage.Size)

	opensearchDashboards := newOpenSearchDashboards(&vmiEnabledCR)
	assert.Equal(t, "192Mi", opensearchDashboards.Resources.RequestMemory)
}

// TestOpenSearchInvalidArgs tests trying to create an opensearch resource with invalid args
// GIVEN a Verrazzano CR with invalid install args
//  WHEN I create a new opensearch resource
//  THEN the opensearch resource fails to create
func TestOpenSearchInvalidArgs(t *testing.T) {
	r := &resourceRequestValues{}
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

	_, err := newOpenSearch(crBadArgs, r, nil)
	assert.Error(t, err)
}
