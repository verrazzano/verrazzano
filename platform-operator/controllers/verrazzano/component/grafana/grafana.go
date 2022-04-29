// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package grafana

import (
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/types"
)

const grafanaDeployment = "vmi-system-grafana"

// isGrafanaReady checks that the Grafana deployment has a minimum number of replicas available and
// the required secret is present
func isGrafanaReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())

	var deployments []types.NamespacedName

	deployments = append(deployments,
		types.NamespacedName{
			Name:      grafanaDeployment,
			Namespace: ComponentNamespace,
		})

	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}
