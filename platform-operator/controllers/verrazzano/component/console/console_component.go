// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package console

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/networkpolicies"
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/authproxy"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ComponentName      = "verrazzano-console"
	ComponentJSONName  = ComponentName
	ComponentNamespace = constants.VerrazzanoSystemNamespace
)

type consoleComponent struct {
	helm.HelmComponent
}

// Verify that ConsoleComponent implements Component
var _ spi.Component = consoleComponent{}

// NewComponent returns a new consoleComponent
func NewComponent() spi.Component {
	return consoleComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			Dependencies:              []string{networkpolicies.ComponentName, authproxy.ComponentName},
			AppendOverridesFunc:       AppendOverrides,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_4_0,
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

// IsEnabled console-specific enabled check for installation
func (c consoleComponent) IsEnabled(effectiveCR runtime.Object) bool {
	return vzconfig.IsConsoleEnabled(effectiveCR)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c consoleComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c consoleComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return nil
}

// IsReady component check
func (c consoleComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isConsoleReady(ctx)
	}
	return false
}

func (c consoleComponent) IsAvailable(context spi.ComponentContext) (reason string, available bool) {
	available = c.IsReady(context)
	if available {
		return fmt.Sprintf("%s is available", c.Name()), true
	}
	return fmt.Sprintf("%s is unavailable: failed readiness checks", c.Name()), false
}

// PreInstall - actions to perform prior to installing this component
func (c consoleComponent) PreInstall(ctx spi.ComponentContext) error {
	return preHook(ctx)
}

// PreUpgrade performs any required pre upgrade operations
func (c consoleComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return preHook(ctx)
}

// GetOverrides gets the install overrides for the console
func GetOverrides(object runtime.Object) interface{} {
	if effectiveCR, ok := object.(*vzapi.Verrazzano); ok {
		if effectiveCR.Spec.Components.Console != nil {
			return effectiveCR.Spec.Components.Console.ValueOverrides
		}
		return []vzapi.Overrides{}
	}
	effectiveCR := object.(*installv1beta1.Verrazzano)
	if effectiveCR.Spec.Components.Console != nil {
		return effectiveCR.Spec.Components.Console.ValueOverrides
	}
	return []installv1beta1.Overrides{}
}

// MonitorOverrides checks whether monitoring of install overrides for the console is enabled or not
func (c consoleComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.Console != nil {
		if ctx.EffectiveCR().Spec.Components.Console.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.Console.MonitorChanges
		}
		return true
	}
	return false
}
