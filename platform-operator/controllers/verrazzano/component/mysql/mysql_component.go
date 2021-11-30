// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = "mysql"

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
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          vzconst.KeycloakNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "mysql-values.yaml"),
			AppendOverridesFunc:     appendMySQLOverrides,
			Dependencies:            []string{istio.ComponentName},
			ReadyStatusFunc:         isReady,
		},
	}
}

// isReady MySQL-specific ready-check
func (c mysqlComponent) IsReady(context spi.ComponentContext) bool {
	return isReady(context, c.ReleaseName, c.ChartNamespace)
}

// isEnabled Kiali-specific enabled check for installation
func (c mysqlComponent) IsEnabled(ctx spi.ComponentContext) bool {
	return isEnabled(ctx)
}

// postInstall Kiali-post-install processing, create or update the Kiali ingress
func (c mysqlComponent) PreInstall(ctx spi.ComponentContext) error {
	return preInstall(ctx, c.ChartNamespace)
}

// postInstall Kiali-post-install processing, create or update the Kiali ingress
func (c mysqlComponent) PostInstall(ctx spi.ComponentContext) error {
	return postInstall(ctx)
}
