// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package oam

import (
	"fmt"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "oam-kubernetes-runtime"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = constants.VerrazzanoSystemNamespace

type oamComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return oamComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "oam-kubernetes-runtime-values.yaml"),
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
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
	if c.IsEnabled(old) && !c.IsEnabled(new) {
		return fmt.Errorf("can not disable previously enabled oam")
	}
	return nil
}
