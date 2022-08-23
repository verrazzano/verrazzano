// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"fmt"
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	installv1beta1 "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1beta1"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "oam-kubernetes-runtime"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "oam"

type oamComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return oamComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "oam-kubernetes-runtime-values.yaml"),
			ImagePullSecretKeyname:    secret.DefaultImagePullSecretKeyName,
			Dependencies:              []string{},
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

// IsEnabled OAM-specific enabled check for installation
func (c oamComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.OAM
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// IsReady component check
func (c oamComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isOAMReady(ctx)
	}
	return false
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c oamComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	// Block all changes for now, particularly around storage changes
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("Disabling component %s is not allowed", ComponentJSONName)
	}
	return c.HelmComponent.ValidateUpdate(old, new)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c oamComponent) ValidateUpdateV1Beta1(old *installv1beta1.Verrazzano, new *installv1beta1.Verrazzano) error {
	return nil
}

// PreUpgrade OAM-pre-upgrade processing
func (c oamComponent) PreUpgrade(ctx spi.ComponentContext) error {
	return common.ApplyCRDYaml(ctx, config.GetHelmOamChartsDir())
}

// PostInstall runs post-install processing for the OAM component
func (c oamComponent) PostInstall(ctx spi.ComponentContext) error {
	if err := ensureClusterRoles(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PostInstall(ctx)
}

func (c oamComponent) PostUninstall(ctx spi.ComponentContext) error {
	return deleteOAMClusterRoles(ctx.Client(), ctx.Log())
}

// PostUpgrade runs post-upgrade processing for the OAM component
func (c oamComponent) PostUpgrade(ctx spi.ComponentContext) error {
	if err := ensureClusterRoles(ctx); err != nil {
		return err
	}
	return c.HelmComponent.PostUpgrade(ctx)
}

// MonitorOverrides checks whether monitoring of install overrides is enabled or not
func (c oamComponent) MonitorOverrides(ctx spi.ComponentContext) bool {
	if ctx.EffectiveCR().Spec.Components.OAM != nil {
		if ctx.EffectiveCR().Spec.Components.OAM.MonitorChanges != nil {
			return *ctx.EffectiveCR().Spec.Components.OAM.MonitorChanges
		}
		return true
	}
	return false
}
