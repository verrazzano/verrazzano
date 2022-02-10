// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherence

import (
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "coherence-operator"

const coherenceOperatorDeploymentName = ComponentName

// IsCoherenceOperatorReady checks if the COH operator deployment is ready
func IsCoherenceOperatorReady(ctx spi.ComponentContext, _ string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: coherenceOperatorDeploymentName, Namespace: namespace},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}
