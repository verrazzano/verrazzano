// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pushgateway

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"strconv"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const deploymentName = "prometheus-pushgateway"

// isPushgatewayReady checks if the Prometheus Pushgateway deployment is ready
func isPushgatewayReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      deploymentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// PreInstall implementation for the Prometheus Pushgateway Component
func preInstall(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Prometheus Pushgateway PreInstall dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Prometheus Pushgateway Component", ComponentNamespace)
	namespace := v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}

// AppendOverrides appends Helm value overrides for the Prometheus Operator Helm chart
func AppendOverrides(ctx spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	ctx.Log().Debug("Appending overrides for the Prometheus Pushgateway components")

	// If prometheus is enabled, enabled the service monitor to add the scrape config for
	// pushgateway through service monitor configuration
	kvs = append(kvs, bom.KeyValue{
		Key:   "serviceMonitor.enabled",
		Value: strconv.FormatBool(vzconfig.IsPrometheusEnabled(ctx.EffectiveCR())),
	})

	return kvs, nil
}

// GetHelmOverrides appends Helm value overrides for the Prometheus Operator Helm chart
func GetHelmOverrides(ctx spi.ComponentContext) []vzapi.Overrides {
	return ctx.EffectiveCR().Spec.Components.PrometheusPushgateway.ValueOverrides
}