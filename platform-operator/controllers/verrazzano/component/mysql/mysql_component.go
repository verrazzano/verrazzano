// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// ComponentName is the name of the component
const ComponentName = "mysql"

// ComponentNamespace is the namespace of the component
const ComponentNamespace = vzconst.KeycloakNamespace

// ComponentJSONName is the josn name of the verrazzano component in CRD
const ComponentJSONName = "keycloak.mysql"

// mysqlComponent represents an MySQL component
type mysqlComponent struct {
	helm.HelmComponent
}

// Verify that mysqlComponent implements Component
var _ spi.Component = mysqlComponent{}

// NewComponent returns a new MySQL component
func NewComponent() spi.Component {
	return mysqlComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			JSONName:                ComponentJSONName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          ComponentNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "mysql-values.yaml"),
			AppendOverridesFunc:     appendMySQLOverrides,
			Dependencies:            []string{istio.ComponentName},
		},
	}
}

// IsReady calls MySQL isMySQLReady function
func (c mysqlComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return isMySQLReady(context)
	}
	return false
}

// IsEnabled mysql-specific enabled check for installation
// If keycloak is enabled, mysql is enabled; disabled otherwise
func (c mysqlComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	comp := effectiveCR.Spec.Components.Keycloak
	if comp == nil || comp.Enabled == nil {
		return true
	}
	return *comp.Enabled
}

// PreInstall calls MySQL preInstall function
func (c mysqlComponent) PreInstall(ctx spi.ComponentContext) error {
	return preInstall(ctx, c.ChartNamespace)
}

// PostInstall calls MySQL postInstall function
func (c mysqlComponent) PostInstall(ctx spi.ComponentContext) error {
	return postInstall(ctx)
}

// ValidateUpdate checks if the specified new Verrazzano CR is valid for this component to be updated
func (c mysqlComponent) ValidateUpdate(old *vzapi.Verrazzano, new *vzapi.Verrazzano) error {
	return nil
}
