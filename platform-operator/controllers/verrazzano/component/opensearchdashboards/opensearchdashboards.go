// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/types"
)

const kibanaDeployment = "vmi-system-kibana"

func getOSDDeployments() []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      kibanaDeployment,
			Namespace: ComponentNamespace,
		},
	}
}

// isOSDReady checks if the OpenSearch-Dashboards resources are ready
func isOSDReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	if vzconfig.IsOpenSearchDashboardsEnabled(ctx.EffectiveCR()) {
		if !ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), getOSDDeployments(), 1, prefix) {
			return false
		}
	}
	return common.IsVMISecretReady(ctx)
}

// doesOSDExist is the IsInstalled check
func doesOSDExist(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	deploy := []types.NamespacedName{{
		Name:      kibanaDeployment,
		Namespace: ComponentNamespace,
	}}
	return ready.DoDeploymentsExist(ctx.Log(), ctx.Client(), deploy, 1, prefix)
}
