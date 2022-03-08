// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherence

import (
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

// IsCoherenceOperatorReady checks if the COH operator deployment is ready
func isCoherenceOperatorReady(ctx spi.ComponentContext) bool {
	deployments := []status.PodReadyCheck{
		{
			NamespacedName: types.NamespacedName{
				Name:      ComponentName,
				Namespace: ComponentNamespace,
			},
			LabelSelector: labels.Set{"control-plane": "coherence"}.AsSelector(),
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}
