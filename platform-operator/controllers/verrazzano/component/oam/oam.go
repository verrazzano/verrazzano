// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package oam

import (
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const oamOperatorDeploymentName = "oam-kubernetes-runtime"

// IsOAMReady checks if the OAM operator deployment is ready
func IsOAMReady(log *zap.SugaredLogger, c client.Client, _ string, namespace string) bool {
	deployments := []types.NamespacedName{
		{Name: oamOperatorDeploymentName, Namespace: namespace},
	}
	return status.DeploymentsReady(log, c, deployments, 1)
}
