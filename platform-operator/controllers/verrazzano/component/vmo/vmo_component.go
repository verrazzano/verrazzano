// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmo

import (
	helmcli "github.com/verrazzano/verrazzano/pkg/helm"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"github.com/verrazzano/verrazzano/platform-operator/internal/vzconfig"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = "verrazzano-monitoring-operator"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.VerrazzanoSystemNamespace

// vmoComponent represents a VMO component
type vmoComponent struct {
	helm.HelmComponent
}

// Verify that vmoComponent implements Component
var _ spi.Component = vmoComponent{}

// NewComponent returns a new VMO component
func NewComponent() spi.Component {
	return vmoComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			AppendOverridesFunc:     appendVmoOverrides,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			Dependencies:            []string{nginx.ComponentName},
		},
	}
}

// IsEnabled VMO enabled check for installation
func (c vmoComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	return vzconfig.IsVMOEnabled(effectiveCR)
}

// IsReady calls VMO isVmoReady function
func (c vmoComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return isVmoReady(context)
	}
	return false
}

// PreInstall VMO pre-install processing
func (c vmoComponent) PreInstall(context spi.ComponentContext) error {
	found, err := helmcli.IsReleaseInstalled(vzconst.Verrazzano, vzconst.VerrazzanoSystemNamespace)
	if err != nil {
		return context.Log().ErrorfNewErr("Failed searching for release: %v", err)
	}
	if found {
		return reassociateResources(context)
	}

	return nil
}

// PreUpgrade VMO pre-upgrade processing
func (c vmoComponent) PreUpgrade(context spi.ComponentContext) error {
	found, err := helmcli.IsReleaseInstalled(vzconst.Verrazzano, vzconst.VerrazzanoSystemNamespace)
	if err != nil {
		return context.Log().ErrorfNewErr("Failed searching for release: %v", err)
	}
	if found {
		if err := reassociateResources(context); err != nil {
			return err
		}
	}
	return common.ApplyCRDYaml(context, config.GetHelmVmoChartsDir())
}

// PostUpgrade VMO post-upgrade processing
func (c vmoComponent) PostUpgrade(context spi.ComponentContext) error {
	return common.ApplyCRDYaml(context, config.GetHelmVmoChartsDir())
}
