// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysqloperator

import (
	"path/filepath"

	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"

	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

const (
	// ComponentName is the name of the component
	ComponentName = "mysql-operator"

	// ComponentNamespace is the namespace of the component
	ComponentNamespace = constants.VerrazzanoSystemNamespace

	// ComponentJSONName is the json name of the component in the CRD
	ComponentJSONName = "mySQLOperator"
)

type mysqlOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return mysqlOperatorComponent{
		helm.HelmComponent{
			ReleaseName:               ComponentName,
			JSONName:                  ComponentJSONName,
			ChartDir:                  filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:            ComponentNamespace,
			IgnoreNamespaceOverride:   true,
			SupportsOperatorInstall:   true,
			SupportsOperatorUninstall: true,
			MinVerrazzanoVersion:      constants.VerrazzanoVersion1_4_0,
			ValuesFile:                filepath.Join(config.GetHelmOverridesDir(), "mysql-operator-values.yaml"),
			Dependencies:              []string{},
			GetInstallOverridesFunc:   GetOverrides,
		},
	}
}

// IsEnabled?? check if keycloak enabled?

// IsReady - component specific ready-check
func (c mysqlOperatorComponent) IsReady(context spi.ComponentContext) bool {
	if c.HelmComponent.IsReady(context) {
		return isReady(context)
	}
	return false
}

// IsInstalled returns true if the component is installed
func (c mysqlOperatorComponent) IsInstalled(ctx spi.ComponentContext) (bool, error) {
	return isInstalled(ctx), nil
}

// IsEnabled returns true if the component is enabled for install
// MySQLOperator must be enabled if Keycloak is enabled
func (c mysqlOperatorComponent) IsEnabled(effectiveCR *vzapi.Verrazzano) bool {
	// Must be enabled if Keycloak is enabled
	keycloakComp := effectiveCR.Spec.Components.Keycloak
	if keycloakComp == nil || keycloakComp.Enabled == nil {
		return true
	}

	// If Keycloak is not enabled, check if MySQLOperator is explicitly enabled
	if !*keycloakComp.Enabled {
		comp := effectiveCR.Spec.Components.MySQLOperator
		if comp != nil && comp.Enabled != nil {
			return *comp.Enabled
		}
	}
	return true
}
