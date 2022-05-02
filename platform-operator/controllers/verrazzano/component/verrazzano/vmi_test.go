// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verrazzano

import (
	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
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
			Prometheus: &vzapi.PrometheusComponent{
				MonitoringComponent: monitoringComponent,
			},
			Grafana: &vzapi.GrafanaComponent{
				MonitoringComponent: monitoringComponent,
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
	grafana := newGrafana(&vmiEnabledCR, r, nil)
	assert.Equal(t, "48Mi", grafana.Resources.RequestMemory)
	assert.Equal(t, "50Gi", grafana.Storage.Size)

	prometheus := newPrometheus(&vmiEnabledCR, r, nil)
	assert.Equal(t, "128Mi", prometheus.Resources.RequestMemory)
	assert.Equal(t, "50Gi", prometheus.Storage.Size)
}

// TestNewGrafanaWithExistingVMI tests that storage values in the VMI are not erased when a new Grafana is created
// GIVEN a Verrazzano CR and an existing VMO
//  WHEN I create a new Grafana resource
//  THEN the storage options from the existing VMO are preserved.
func TestNewGrafanaWithExistingVMI(t *testing.T) {
	existingVmo := vmov1.VerrazzanoMonitoringInstance{
		Spec: vmov1.VerrazzanoMonitoringInstanceSpec{
			Grafana: vmov1.Grafana{
				Enabled: true,
				Storage: vmov1.Storage{
					Size: "100Gi",
					PvcNames: []string{
						"my-pvc",
					},
				},
			},
		},
	}

	grafana := newGrafana(&vmiEnabledCR, nil, &existingVmo)
	assert.NotNil(t, grafana)
	assert.Equal(t, "100Gi", grafana.Storage.Size)
	assert.Equal(t, []string{"my-pvc"}, grafana.Storage.PvcNames)
}

// TestNewPrometheusWithDefaultStorage tests that the default storage of Prometheus is 50Gi
// GIVEN a Verrazzano CR
// WHEN I create a new Prometheus resource
//  THEN the storage is 50Gi
func TestNewPrometheusWithDefaultStorage(t *testing.T) {
	prometheus := newPrometheus(&vmiEnabledCR, nil, nil)
	assert.Equal(t, "50Gi", prometheus.Storage.Size)
}

// TestPrometheusWithStorageOverride tests that storage overrides are applied to Prometheus
// GIVEN a Verrazzano CR and a storage override of 100Gi
// WHEN I create a new Prometheus resource
//  THEN the storage is 100Gi
func TestPrometheusWithStorageOverride(t *testing.T) {
	prometheus := newPrometheus(&vmiEnabledCR, &common.ResourceRequestValues{Storage: "100Gi"}, nil)
	assert.Equal(t, "100Gi", prometheus.Storage.Size)
}
