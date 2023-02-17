// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"

	vmov1 "github.com/verrazzano/verrazzano-monitoring-operator/pkg/apis/vmcontroller/v1"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
)

// updateFunc is passed into CreateOrUpdateVMI to create the necessary VMI resources
func updateFunc(ctx spi.ComponentContext, storage *common.ResourceRequestValues, vmi *vmov1.VerrazzanoMonitoringInstance, existingVMI *vmov1.VerrazzanoMonitoringInstance) error {
	vmi.Spec.OpensearchDashboards = newOpenSearchDashboards(ctx.EffectiveCR())
	return nil
}

func newOpenSearchDashboards(cr *vzapi.Verrazzano) vmov1.OpensearchDashboards {
	if cr.Spec.Components.Kibana == nil {
		return vmov1.OpensearchDashboards{}
	}
	kibanaValues := cr.Spec.Components.Kibana
	opensearchDashboards := vmov1.OpensearchDashboards{
		Enabled: kibanaValues.Enabled != nil && *kibanaValues.Enabled,
		Resources: vmov1.Resources{
			RequestMemory: "192Mi",
		},
	}
	// Set the Plugins to the VMI
	opensearchDashboards.Plugins = kibanaValues.Plugins

	if kibanaValues.Replicas != nil {
		opensearchDashboards.Replicas = *kibanaValues.Replicas
	}
	return opensearchDashboards
}
