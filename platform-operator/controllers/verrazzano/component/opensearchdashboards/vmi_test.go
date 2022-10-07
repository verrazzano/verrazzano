// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

var enabled = true

var enabledCR = vzapi.Verrazzano{
	Spec: vzapi.VerrazzanoSpec{
		Profile: vzapi.Prod,
		Components: vzapi.ComponentSpec{
			DNS: dnsComponents.DNS,
			Kibana: &vzapi.KibanaComponent{
				Enabled: &enabled,
			},
		},
	},
}

// TestNewVMIResources tests that new VMI resources can be created from a CR
// GIVEN a Verrazzano CR
//
//	WHEN I create new VMI resources
//	THEN the configuration in the CR is respected
func TestNewVMIResources(t *testing.T) {
	opensearchDashboards := newOpenSearchDashboards(enabledCR.DeepCopy())
	assert.Equal(t, "192Mi", opensearchDashboards.Resources.RequestMemory)
}

// TestNewVMIResourcesWithReplicas tests that new VMI resources can be created from a CR with replicas configured
// GIVEN a Verrazzano CR
//
//	WHEN I create new VMI resources
//	THEN the configuration in the CR is respected with replicas configured
func TestNewVMIResourcesWithReplicas(t *testing.T) {
	cr := enabledCR.DeepCopy()
	var replicas int32 = 2
	cr.Spec.Components.Kibana.Replicas = &replicas
	osd := newOpenSearchDashboards(cr)
	assert.Equal(t, replicas, osd.Replicas)

}
