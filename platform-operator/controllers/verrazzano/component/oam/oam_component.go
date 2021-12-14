// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package oam

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

type oamComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return oamComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "oam-kubernetes-runtime-values.yaml"),
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ReadyStatusFunc:         IsOAMReady,
		},
	}
}

// IsEnabled OAM-specific enabled check for installation
func (c oamComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.OAM
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}
