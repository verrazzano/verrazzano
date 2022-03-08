// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package weblogic

import (
	"fmt"
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "weblogic-operator"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

type weblogicComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return weblogicComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "weblogic-values.yaml"),
			PreInstallFunc:          WeblogicOperatorPreInstall,
			AppendOverridesFunc:     AppendWeblogicOperatorOverrides,
			Dependencies:            []string{istio.ComponentName},
		},
	}
}

// IsEnabled WebLogic-specific enabled check for installation
func (c weblogicComponent) IsEnabled(ctx spi.ComponentContext) bool {
	return isComponentEnabled(ctx.EffectiveCR())
}

func isComponentEnabled(vz *vzapi.Verrazzano) bool {
	comp := vz.Spec.Components.WebLogicOperator
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// ValidateInstall checks if the specified Verrazzano CR is valid for this component to be installed
func (c weblogicComponent) ValidateInstall(vz *vzapi.Verrazzano) error {
	return nil
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c weblogicComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	if isComponentEnabled(old) && !isComponentEnabled(new) {
		return fmt.Errorf("can not disable weblogicOperator")
	}
	return nil
}

// IsReady component check
func (c weblogicComponent) IsReady(ctx spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(ctx) {
		return isWeblogicOperatorReady(ctx)
	}
	return false
}
