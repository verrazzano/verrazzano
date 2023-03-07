// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package opensearchoperator

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"k8s.io/apimachinery/pkg/types"
)

const opensearchOperatorDeploymentName = "opensearch-operator-controller-manager"

func (o opensearchOperatorComponent) isReady(ctx spi.ComponentContext) bool {
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), getDeploymentList(), 1, getPrefix(ctx))
}

func getDeploymentList() []types.NamespacedName {
	return []types.NamespacedName{
		{
			Name:      opensearchOperatorDeploymentName,
			Namespace: ComponentNamespace,
		},
	}
}

func getPrefix(ctx spi.ComponentContext) string {
	return fmt.Sprintf("Component %s", ctx.GetComponent())
}
