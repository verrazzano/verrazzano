// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package operator

import (
	"fmt"
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/certmanager"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"

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

var certificates = []types.NamespacedName{
	{Namespace: constants.VerrazzanoSystemNamespace, Name: jaegerCertificateName},
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
			Certificates:            certificates,
			IngressNames: []types.NamespacedName{
				{
					Namespace: constants.VerrazzanoSystemNamespace,
					Name:      constants.JaegerIngress,
				},
			},
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

func (c jaegerOperatorComponent) PostInstall(ctx spi.ComponentContext) error {
	if err := c.createOrUpdateJaegerResources(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PostInstall(ctx)
}

func (c jaegerOperatorComponent) PostUpgrade(ctx spi.ComponentContext) error {
	if err := c.HelmComponent.PostUpgrade(ctx); err != nil {
		return err
	}
	return c.createOrUpdateJaegerResources(ctx)
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

// createOrUpdateKialiResources create or update related Kiali resources
func (c jaegerOperatorComponent) createOrUpdateJaegerResources(ctx spi.ComponentContext) error {
	jaegerCREnabled, err := isCreateJaegerInstance(ctx)
	if err != nil {
		return err
	}
	if vzconfig.IsNGINXEnabled(ctx.EffectiveCR()) && jaegerCREnabled {
		if err := createOrUpdateJaegerIngress(ctx, constants.VerrazzanoSystemNamespace); err != nil {
			return err
		}
	}
	return nil
}
