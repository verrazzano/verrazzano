// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package kiali

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/nginx"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path/filepath"
)

type kialiComponent struct {
	helm.HelmComponent
}

var _ spi.Component = kialiComponent{}

const kialiOverridesFile = "kiali-server-values.yaml"

func NewComponent() spi.Component {
	return kialiComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), kialiOverridesFile),
			Dependencies:            []string{nginx.ComponentName},
			AppendOverridesFunc:     AppendOverrides,
			MinVerrazzanoVersion:    constants.VerrazzanoVersion1_1_0,
		},
	}
}

// PostInstall Kiali-post-install processing, create or update the Kiali ingress
func (c kialiComponent) PostInstall(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Kiali post-install")
	if err := c.HelmComponent.PostInstall(ctx); err != nil {
		return err
	}
	return c.createOrUpdateKialiResources(ctx)
}

// PostUpgrade Kiali-post-upgrade processing, create or update the Kiali ingress
func (c kialiComponent) PostUpgrade(ctx spi.ComponentContext) error {
	ctx.Log().Debugf("Kiali post-upgrade")
	if err := c.HelmComponent.PostUpgrade(ctx); err != nil {
		return err
	}
	return c.createOrUpdateKialiResources(ctx)
}

// IsReady Kiali-specific ready-check
func (c kialiComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return isKialiReady(context, c.ReleaseName, c.ChartNamespace)
	}
	return false
}

// IsEnabled Kiali-specific enabled check for installation
func (c kialiComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.Kiali
	if comp != nil && comp.Enabled != nil {
		return *comp.Enabled
	}
	return c.HelmComponent.IsEnabledFunc(ctx)
}

// createOrUpdateKialiResources create or update related Kiali resources
func (c kialiComponent) createOrUpdateKialiResources(ctx spi.ComponentContext) error {
	if err := createOrUpdateKialiIngress(ctx, c.ChartNamespace); err != nil {
		return err
	}
	if err := createOrUpdateAuthPolicy(ctx); err != nil {
		return err
	}
	return nil
}
