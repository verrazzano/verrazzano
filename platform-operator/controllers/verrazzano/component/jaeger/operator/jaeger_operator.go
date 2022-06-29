// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/bom"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/k8s/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

const (
	deploymentName = "jaeger-operator"
)

// Jaeger Subcomponent names in Verrazzano BOM
var (
	subComponentNames = []string{"jaeger-agent", "jaeger-collector", "jaeger-query", "jaeger-ingester"}
)

// isjaegerOperatorReady checks if the Jaeger Operator deployment is ready
func isJaegerOperatorReady(ctx spi.ComponentContext) bool {
	deployments := []types.NamespacedName{
		{
			Name:      deploymentName,
			Namespace: ComponentNamespace,
		},
	}
	prefix := fmt.Sprintf("Component %s", ctx.GetComponent())
	return status.DeploymentsAreReady(ctx.Log(), ctx.Client(), deployments, 1, prefix)
}

// PreInstall implementation for the Jaeger Operator Component
func preInstall(ctx spi.ComponentContext) error {
	// Do nothing if dry run
	if ctx.IsDryRun() {
		ctx.Log().Debug("Jaeger Operator PreInstall dry run")
		return nil
	}

	// Create the verrazzano-monitoring namespace if not already created
	return ensureVerrazzanoMonitoringNamespace(ctx)
}

// AppendOverrides appends Helm value overrides for the Jaeger Operator component's Helm chart
func AppendOverrides(compContext spi.ComponentContext, _ string, _ string, _ string, kvs []bom.KeyValue) ([]bom.KeyValue, error) {
	// Append Jaeger images from the subcomponents in the bom
	compContext.Log().Debug("Appending the image overrides for the Jaeger Operator component")
	return appendJaegerImageOverrides(compContext, kvs, subComponentNames)
}

// validateJaegerOperator checks scenarios in which the Verrazzano CR violates install verification
// due to Jaeger Operator specifications
func (c jaegerOperatorComponent) validateJaegerOperator(vz *vzapi.Verrazzano) error {
	// Validate install overrides
	if vz.Spec.Components.JaegerOperator != nil {
		if err := vzapi.ValidateInstallOverrides(vz.Spec.Components.JaegerOperator.ValueOverrides); err != nil {
			return err
		}
	}
	return nil
}

// GetOverrides returns the list of install overrides for a component
func GetOverrides(effectiveCR *vzapi.Verrazzano) []vzapi.Overrides {
	if effectiveCR.Spec.Components.JaegerOperator != nil {
		return effectiveCR.Spec.Components.JaegerOperator.ValueOverrides
	}
	return []vzapi.Overrides{}
}

func ensureVerrazzanoMonitoringNamespace(ctx spi.ComponentContext) error {
	// Create the verrazzano-monitoring namespace
	ctx.Log().Debugf("Creating namespace %s for the Jaeger Operator", ComponentNamespace)
	namespace := corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ComponentNamespace}}
	if _, err := controllerruntime.CreateOrUpdate(context.TODO(), ctx.Client(), &namespace, func() error {
		return nil
	}); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to create or update the %s namespace: %v", ComponentNamespace, err)
	}
	return nil
}

// appendJaegerImageOverrides takes a list of subcomponent image names and appends it to the given Helm overrides
func appendJaegerImageOverrides(ctx spi.ComponentContext, kvs []bom.KeyValue, subcomponents []string) ([]bom.KeyValue, error) {
	bomFile, err := bom.NewBom(config.GetDefaultBOMFilePath())
	if err != nil {
		return kvs, ctx.Log().ErrorNewErr("Failed to get the bom file for the Jaeger image overrides: ", err)
	}

	for _, subcomponent := range subcomponents {
		imageOverrides, err := bomFile.BuildImageOverrides(subcomponent)
		if err != nil {
			return kvs, ctx.Log().ErrorfNewErr("Failed to build the Jaeger image overrides for subcomponent %s: ", subcomponent, err)
		}
		kvs = append(kvs, imageOverrides...)
	}
	ctx.Log().Infof("Appending jaeger image overrides %v", kvs)
	return kvs, nil
}
