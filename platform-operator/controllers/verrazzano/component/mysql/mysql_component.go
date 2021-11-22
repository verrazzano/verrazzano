// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mysql

import (
	vzconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/secret"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path/filepath"
)

// ComponentName is the name of the component
const ComponentName = "mysql"

// MySQLComponent represents an MySQL component
type MySQLComponent struct {
	helm.HelmComponent
}

// Verify that MySQLComponent implements Component
var _ spi.Component = MySQLComponent{}

// NewComponent returns a new MySQL component
func NewComponent() spi.Component {
	return MySQLComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          vzconst.KeycloakNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  secret.DefaultImagePullSecretKeyName,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "mysql-values.yaml"),
			PreInstallFunc:          PreInstall,
			AppendOverridesFunc:     AppendMySQLOverrides,
			PostInstallFunc:         PostInstall,
			Dependencies:            []string{},
			ReadyStatusFunc:         IsReady,
		},
	}
}
