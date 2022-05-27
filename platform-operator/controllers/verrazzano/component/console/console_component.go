// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package console

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path/filepath"
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
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			AppendOverridesFunc:     AppendOverrides,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
		},
	}
}

// IsEnabled console-specific enabled check for installation
func (c consoleComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Console
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c consoleComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Do not allow any changes except to enable the component post-install
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// IsReady component check
func (c consoleComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isConsoleReady(ctx)
	}
	return false
}

// PreInstall - actions to perform prior to installing this component
func (c consoleComponent) PreInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debug("Verrazzano Console pre-install")
	if err := preHook(ctx); err != nil {
		return err
	}

	// Temporary work around for installer bug of calling pre-install after a component is installed
	installed, err := c.IsInstalled(ctx)
	if err != nil {
		return err
	}
	if installed {
		ctx.Log().Oncef("Component %s already installed, skipping PreInstall checks", ComponentName)
		return nil
	}

	return nil
}

// PreUpgrade performs any required pre upgrade operations
func (c consoleComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return preHook(ctx)
}
