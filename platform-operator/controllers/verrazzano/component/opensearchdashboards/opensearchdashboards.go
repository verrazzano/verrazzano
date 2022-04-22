// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchdashboards

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/types"
)

const kibanaDeployment = "vmi-system-kibana"

// checkOpenSearchDashboardsStatus checks performs checks on the OpenSearch-Dashboards resources
func checkOpenSearchDashboardsStatus(ctx spi.ComponentContext, deploymentFunc status.DeploymentFunc) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	var deployments []types.NamespacedName

	if vzconfig.IsKibanaEnabled(ctx.EffectiveCR()) {
		deployments = append(deployments,
			types.NamespacedName{
				Name:      kibanaDeployment,
				Namespace: ComponentNamespace,
			})
	}

	if !deploymentFunc(ctx.Log(), ctx.Client(), deployments, 1, prefix) {
		return false
	}

	return common.IsVMISecretReady(ctx)
}
