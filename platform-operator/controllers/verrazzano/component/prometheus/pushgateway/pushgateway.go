// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pushgateway

import (
	"context"
	"fmt"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/verrazzano/verrazzano/pkg/k8s/ready"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const deploymentName = "prometheus-pushgateway"

// isPushgatewayReady checks if the Prometheus Pushgateway deployment is ready
func (c prometheusPushgatewayComponent) isPushgatewayReady(ctx spi.ComponentContext) bool {
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return ready.DeploymentsAreReady(ctx.Log(), ctx.Client(), c.AvailabilityObjects.DeploymentNames, 1, prefix)
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
	ns := common.GetVerrazzanoMonitoringNamespace()
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), ns, func() error {
		common.MutateVerrazzanoMonitoringNamespace(ctx, ns)
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}

// GetOverrides returns install overrides for a component
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.PrometheusPushgateway != nil {
			return effectiveCR.Spec.Components.PrometheusPushgateway.ValueOverrides
		}
		return []vzapi.Overrides{}
	} else if effectiveCR, ok := object.(*installv1beta1.Verrazzano); ok {
		if effectiveCR.Spec.Components.PrometheusPushgateway != nil {
			return effectiveCR.Spec.Components.PrometheusPushgateway.ValueOverrides
		}
		return []installv1beta1.Overrides{}
	}

	return []vzapi.Overrides{}
}
