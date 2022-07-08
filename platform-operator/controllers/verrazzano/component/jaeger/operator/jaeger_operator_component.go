// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"context"
	"fmt"
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	// ComponentName is the name of the component
	ComponentName = "jaeger-operator"
	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoMonitoringNamespace
	// ComponentJSONName is the json name of the component in the CRD
	ComponentJSONName = "jaegerOperator"
	// ChartDir is the relative directory path for Jaeger Operator chart
	ChartDir = "jaegertracing/jaeger-operator"
)

type jaegerOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return jaegerOperatorComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ChartDir),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_3_0,
			ImagePullSecretKeyname:  "image.imagePullSecrets[0].name",
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "jaeger-operator-values.yaml"),
			Dependencies:            []string{certmanager.ComponentName},
			AppendOverridesFunc:     AppendOverrides,
			GetInstallOverridesFunc: GetOverrides,
		},
	}
}

// IsEnabled returns true only if the Jaeger Operator is explicitly enabled
// in the Verrazzano CR.
func (c jaegerOperatorComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.JaegerOperator
	if comp == nil || comp.Enabled == nil {
		return false
	}
	return *comp.Enabled
}

// IsReady checks if the Jaeger Operator deployment is ready
func (c jaegerOperatorComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isJaegerOperatorReady(ctx)
	}
	return false
}

// MonitorOverrides checks whether monitoring is enabled for install overrides sources
func (c jaegerOperatorComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.JaegerOperator == nil {
		return false
	}
	if ctx.EffectiveCR().Spec.Components.JaegerOperator.MonitorChanges != nil {
		return *ctx.EffectiveCR().Spec.Components.JaegerOperator.MonitorChanges
	}
	return true
}

// PreInstall updates resources necessary for the Jaeger Operator Component installation
func (c jaegerOperatorComponent) PreInstall(ctx spi.ComponentContext) error {
	return preInstall(ctx)
}

// ValidateInstall verifies the installation of the Verrazzano object
func (c jaegerOperatorComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return c.validateJaegerOperator(vz)
}

// ValidateUpgrade verifies the upgrade of the Verrazzano object
func (c jaegerOperatorComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("disabling component %s is not allowed", ComponentJSONName)
	}
	return c.validateJaegerOperator(new)
}

// PreUpgrade Jaeger component pre-upgrade processing
func (f jaegerOperatorComponent) PreUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Jaeger pre-upgrade")
	if err := removeDeployment(ctx); err != nil {
		return err
	}
	return nil
}

//Uprade jaegeroperator component for upgrade processing.
func (c jaegerOperatorComponent) Upgrade(ctx spi.ComponentContext) error {

	return c.HelmComponent.Install(ctx)
}

// IsInstalled checks if jaeger is installed
func (c jaegerOperatorComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, deployment)
	if errors.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		ctx.Log().Errorf("Failed to get %s/%s deployment: %v", ComponentNamespace, ComponentName, err)
		return false, err
	}
	return true, nil
}

// removeDeploymentAndService removes the Jaeger deployment during pre-upgrade.
// The match selector for jaeger operator deployment was changed in 1.34.1 from the previous jaeger version (1.32.0) that Verrazzano installed.
// The match selector is an immutable field so this was a workaround to avoid a failure during jaeger upgrade.
func removeDeployment(ctx spi.ComponentContext) error {
	deployment := &appsv1.Deployment{}
	if err := ctx.Client().Get(context.TODO(), types.NamespacedName{Namespace: ComponentNamespace, Name: ComponentName}, deployment); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to get deployment %s/%s: %v", ComponentNamespace, ComponentName, err)
	}
	// Remove the jaeger deployment only if the match selector is not what is expected.
	if deployment.Spec.Selector != nil && len(deployment.Spec.Selector.MatchExpressions) == 0 && len(deployment.Spec.Selector.MatchLabels) == 2 {
		instance, ok := deployment.Spec.Selector.MatchLabels["app.kubernetes.io/instance"]
		if ok && instance == ComponentName {
			name, ok := deployment.Spec.Selector.MatchLabels["app.kubernetes.io/name"]
			if ok && name == ComponentName {
				return nil
			}
		}
	}
	if err := ctx.Client().Delete(context.TODO(), deployment); err != nil {
		return ctx.Log().ErrorfNewErr("Failed to delete deployment %s/%s: %v", ComponentNamespace, ComponentName, err)
	}
	return nil
}
