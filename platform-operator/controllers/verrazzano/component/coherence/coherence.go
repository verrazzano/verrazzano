// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package coherence

import (
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	clipkg "sigs.k8s.io/controller-runtime/pkg/client"
)

// ComponentName is the name of the component
const ComponentName = "coherence-operator"

const coherenceOperatorDeploymentName = ComponentName

// IsCoherenceOperatorReady checks if the COH operator deployment is ready
func IsCoherenceOperatorReady(log *zap.SugaredLogger, c clipkg.Client, _ string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: coherenceOperatorDeploymentName, Namespace: namespace},
	}
	return status.DeploymentsReady(log, c, deployments, 1)
}

// IsEnabled returns true if the component is enabled, which is the default
func IsEnabled(comp *v1alpha1.CoherenceOperatorComponent) bool {
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}
