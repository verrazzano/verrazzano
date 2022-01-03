// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package weblogic

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

type weblogicComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return weblogicComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "weblogic-values.yaml"),
			PreInstallFunc:          WeblogicOperatorPreInstall,
			AppendOverridesFunc:     AppendWeblogicOperatorOverrides,
			Dependencies:            []string{istio.ComponentName},
			ReadyStatusFunc:         IsWeblogicOperatorReady,
		},
	}
}

// IsEnabled WebLogic-specific enabled check for installation
func (c weblogicComponent) IsEnabled(ctx spi.ComponentContext) bool {
	comp := ctx.EffectiveCR().Spec.Components.WebLogicOperator
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}
