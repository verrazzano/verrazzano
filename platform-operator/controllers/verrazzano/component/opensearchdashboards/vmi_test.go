// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"github.com/stretchr/testify/assert"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"testing"
)

var enabled = true

// TestNewVMIResources tests that new VMI resources can be created from a CR
// GIVEN a Verrazzano CR
//  WHEN I create new VMI resources
//  THEN the configuration in the CR is respected
func TestNewVMIResources(t *testing.T) {
	opensearchDashboards := newOpenSearchDashboards(&vzapi.Verrazzano{
		Spec: vzapi.VerrazzanoSpec{
			Profile: vzapi.Prod,
			Components: vzapi.ComponentSpec{
				DNS: dnsComponents.DNS,
				Kibana: &vzapi.KibanaComponent{
					MonitoringComponent: vzapi.MonitoringComponent{
						Enabled: &enabled,
					},
				},
			},
		},
	})
	assert.Equal(t, "192Mi", opensearchDashboards.Resources.RequestMemory)
}
