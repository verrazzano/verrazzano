// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/types"
)

// ComponentName is the name of the component
const ComponentName = "oam-kubernetes-runtime"

const oamOperatorDeploymentName = ComponentName

// IsOAMReady checks if the OAM operator deployment is ready
func IsOAMReady(context spi.ComponentContext, _ string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: oamOperatorDeploymentName, Namespace: namespace},
	}
	prefix := fmt.Sprintf("Component %s", context.GetComponent())
	return status.DeploymentsReady(context.Log(), context.Client(), deployments, 1, prefix)
}
