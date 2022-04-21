// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"context"
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	globalconst "github.com/verrazzano/verrazzano/pkg/constants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

var enabled = true

var monitoringComponent = vzapi.MonitoringComponent{
	Enabled: &enabled,
}

var vmiEnabledCR = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Profile: vzapi.Prod,
		Components: vzapi.ComponentSpec{
			DNS: dnsComponents.DNS,
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
	opensearchDashboards := newOpenSearchDashboards(&vmiEnabledCR)
	assert.Equal(t, "192Mi", opensearchDashboards.Resources.RequestMemory)
}

// TestCreateVMI tests a new VMI resources is created in K8s according to the CR
// GIVEN a Verrazzano CR
// WHEN I create a new VMI resource
//  THEN the configuration in the CR is respected
func TestCreateVMI(t *testing.T) {
	ctx := spi.NewFakeContext(fake.NewClientBuilder().WithScheme(testScheme).Build(), &vmiEnabledCR, false)
	err := createVMIforOSD(ctx)
	assert.NoError(t, err)
	vmi := &vmov1.VerrazzanoMonitoringInstance{}
	namespacedName := types.NamespacedName{Name: system, Namespace: globalconst.VerrazzanoSystemNamespace}
	err = ctx.Client().Get(context.TODO(), namespacedName, vmi)
	assert.NoError(t, err)
	assert.Equal(t, "192Mi", vmi.Spec.Kibana.Resources.RequestMemory)
}
