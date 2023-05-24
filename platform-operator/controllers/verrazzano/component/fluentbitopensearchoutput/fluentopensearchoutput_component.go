// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentbitopensearchoutput

import (
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/fluentoperator"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"k8s.io/apimachinery/pkg/runtime"
	"path/filepath"
)

const (
	ComponentName      = "fluentbit-opensearch-output"
	ComponentJSONName  = "fluentbitOpensearchOutput"
	ComponentNamespace = constants.VerrazzanoSystemNamespace
)

type fluentbitOpensearchOutput struct {
	helm.HelmComponent
}

var _ spi.Component = fluentbitOpensearchOutput{}

func NewComponent() spi.Component {
	return fluentbitOpensearchOutput{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_6_0,
			GetInstallOverridesFunc:   getOverrides,
			Dependencies:              []string{fluentoperator.ComponentName},
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
		},
	}
}

func (c fluentbitOpensearchOutput) PreInstall(ctx spi.ComponentContext) error {
	// TODO: check for verrazzano-es-internal secret
	return c.HelmComponent.PreInstall(ctx)
}

func (c fluentbitOpensearchOutput) PreUpgrade(ctx spi.ComponentContext) error {
	// TODO: check for verrazzano-es-internal secret
	return c.HelmComponent.PreUpgrade(ctx)
}

func (c fluentbitOpensearchOutput) Reconcile(ctx spi.ComponentContext) error {
	installed, err := c.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		err = c.Install(ctx)
	}
	return err
}

// GetOverrides returns install overrides for a component
func getOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*v1alpha1.Verrazzano); ok {
		if effectiveCR.Spec.Components.FluentbitOpensearchOutput != nil {
			return effectiveCR.Spec.Components.FluentbitOpensearchOutput.ValueOverrides
		}
		return []v1alpha1.Overrides{}
	}
	effectiveCR := object.(*v1beta1.Verrazzano)
	if effectiveCR.Spec.Components.FluentbitOpensearchOutput != nil {
		return effectiveCR.Spec.Components.FluentbitOpensearchOutput.ValueOverrides
	}
	return []v1beta1.Overrides{}
}

func (c fluentbitOpensearchOutput) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.FluentbitOpensearchOutput != nil {
		if ctx.EffectiveCR().Spec.Components.FluentbitOpensearchOutput.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.FluentbitOpensearchOutput.MonitorChanges
		}
		return true
	}
	return false
}

func (c fluentbitOpensearchOutput) IsEnabled(cr runtime.Object) bool {
	return vzcr.IsFluentbitOpensearchOutputEnabled(cr)
}
