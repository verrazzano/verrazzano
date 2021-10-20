// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package appoper

import (
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path/filepath"
)

type applicationOperatorComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return applicationOperatorComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetHelmChartsDir(), ComponentName),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "verrazzano-application-operator-values.yaml"),
			AppendOverridesFunc:     AppendApplicationOperatorOverrides,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0]",
			ReadyStatusFunc:         IsApplicationOperatorReady,
			Dependencies:            []string{"oam-kubernetes-runtime"},
			PreUpgradeFunc:          ApplyCRDYaml,
		},
	}
}
