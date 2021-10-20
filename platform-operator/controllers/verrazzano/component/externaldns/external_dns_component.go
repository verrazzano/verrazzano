// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.
package externaldns

import (
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"path/filepath"
)

type externalDNSComponent struct {
	helm.HelmComponent
}

func NewComponent() spi.Component {
	return externalDNSComponent{
		helm.HelmComponent{
			ReleaseName:             ComponentName,
			ChartDir:                filepath.Join(config.GetThirdPartyDir(), ComponentName),
			ChartNamespace:          "cert-manager",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(config.GetHelmOverridesDir(), "external-dns-values.yaml"),
		},
	}
}
