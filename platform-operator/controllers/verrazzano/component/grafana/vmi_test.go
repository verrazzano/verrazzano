// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"github.com/verrazzano/verrazzano-monitoring-operator/pkg/resources"
	"testing"

	"github.com/stretchr/testify/assert"
	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

var enabled = true

var grafanaEnabledCR = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Profile: vzapi.Prod,
		Components: vzapi.ComponentSpec{
			Grafana: &vzapi.GrafanaComponent{
				Enabled:  &enabled,
				Replicas: resources.NewVal(2),
				Database: &vzapi.DatabaseInfo{
					Host: "testhost:3032",
					Name: "grafanadb",
				},
			},
		},
	},
}

// TestNewGrafana tests that a Grafana VMO config can be created from a CR
// GIVEN a Verrazzano CR
//
//	WHEN I create new Grafana resource
//	THEN the configuration in the CR is respected
func TestNewGrafana(t *testing.T) {
	vmi := vmov1.VerrazzanoMonitoringInstance{}
	r := &common.ResourceRequestValues{
		Memory:  "",
		Storage: "50Gi",
	}

	ctx := spi.NewFakeContext(nil, &grafanaEnabledCR, nil, false)
	updateFunc(ctx, r, &vmi, nil)
	assert.True(t, vmi.Spec.Grafana.Enabled)
	assert.Equal(t, "48Mi", vmi.Spec.Grafana.Resources.RequestMemory)
	assert.Equal(t, "50Gi", vmi.Spec.Grafana.Storage.Size)
	assert.Equal(t, vmi.Spec.Grafana.Replicas, int32(2))
	assert.NotNil(t, vmi.Spec.Grafana.Database, "Database is nil")
	assert.Equal(t, "grafana-db", vmi.Spec.Grafana.Database.PasswordSecret)
	assert.Equal(t, "testhost:3032", vmi.Spec.Grafana.Database.Host)
	assert.Equal(t, "grafanadb", vmi.Spec.Grafana.Database.Name)
}

// TestNewGrafanaWithExistingVMI tests that storage values in the VMI are not erased when a new Grafana is created
// GIVEN a Verrazzano CR and an existing VMO
//
//	WHEN I create a new Grafana resource
//	THEN the storage options from the existing VMO are preserved.
func TestNewGrafanaWithExistingVMI(t *testing.T) {
	vmi := vmov1.VerrazzanoMonitoringInstance{}
	existingVmi := vmov1.VerrazzanoMonitoringInstance{
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

	ctx := spi.NewFakeContext(nil, &grafanaEnabledCR, nil, false)
	updateFunc(ctx, nil, &vmi, &existingVmi)
	assert.True(t, vmi.Spec.Grafana.Enabled)
	assert.Equal(t, "50Gi", vmi.Spec.Grafana.Storage.Size)
	assert.Equal(t, []string{"my-pvc"}, vmi.Spec.Grafana.Storage.PvcNames)
}
