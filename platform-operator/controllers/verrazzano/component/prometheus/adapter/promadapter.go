// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package adapter

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// isPrometheusAdapterReady checks if the Prometheus Adapter deployment is ready
func isPrometheusAdapterReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      ComponentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// PreInstall implementation for the Prometheus Adapter Component
func preInstall(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Prometheus Adapter preInstall dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Prometheus Adapter", ComponentNamespace)
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}
