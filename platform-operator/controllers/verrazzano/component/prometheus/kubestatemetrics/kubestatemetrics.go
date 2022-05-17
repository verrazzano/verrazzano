// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kubestatemetrics

import (
	"context"
	"fmt"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/prometheus"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const deploymentName = "kube-state-metrics"

// isDeploymentReady checks if the kube-state-metrics deployment is ready
func isDeploymentReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      deploymentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// PreInstall implementation for the Kube State Metrics Component
func preInstall(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Kube State Metrics PreInstall dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for Kube State Metrics", ComponentNamespace)
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &prometheus.VerrazzanoMonitoringNamespace, func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}
