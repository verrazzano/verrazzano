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

// areOpenSearchDashboardsInstalled checks if OpenSearch-Dashboards has been installed yet
func areOpenSearchDashboardsInstalled(ctx spi.ComponentContext) (bool, error) {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	var deployments []types.NamespacedName

	if vzconfig.IsKibanaEnabled(ctx.EffectiveCR()) {
		deployments = append(deployments,
			types.NamespacedName{
				Name:      kibanaDeployment,
				Namespace: ComponentNamespace,
			})
	}

	deploymentsExist, err := status.DoDeploymentsExist(ctx.Log(), ctx.Client(), deployments, prefix)
	if !deploymentsExist {
		return false, err
	}

	return common.IsVMISecretReady(ctx), nil
}

// areOpenSearchDashboardsReady VMI components ready-check
func areOpenSearchDashboardsReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	var deployments []types.NamespacedName

	if vzconfig.IsKibanaEnabled(ctx.EffectiveCR()) {
		deployments = append(deployments,
			types.NamespacedName{
				Name:      kibanaDeployment,
				Namespace: ComponentNamespace,
			})
	}

	if !status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix) {
		return false
	}

	return common.IsVMISecretReady(ctx)
}
