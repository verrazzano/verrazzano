// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// GetComponents returns the list of components that are installable and upgradeable.
// The components will be processed in the order items in the array
func GetComponents() []Component {
	thirdpartyDir := config.Get().ThirdpartyChartsDir
	componentDir := filepath.Join(config.Get().VerrazzanoInstallDir, "components")

	return []Component{
		helmComponent{
			releaseName:             "istio",
			chartDir:                filepath.Join(thirdpartyDir, "istio"),
			chartNamespace:          "istio-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "istio-values.yaml"),
			preUpgradeFunc:          PreUpgrade,
		},
		Nginx{},
		helmComponent{
			releaseName:             "cert-manager",
			chartDir:                filepath.Join(thirdpartyDir, "cert-manager"),
			chartNamespace:          "cert-manager",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "cert-manager-values.yaml"),
		},
		helmComponent{
			releaseName:             "external-dns",
			chartDir:                filepath.Join(thirdpartyDir, "external-dns"),
			chartNamespace:          "cert-manager",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "external-dns-values.yaml"),
		},
		helmComponent{
			releaseName:             "rancher",
			chartDir:                filepath.Join(thirdpartyDir, "rancher"),
			chartNamespace:          "cattle-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "rancher-values.yaml"),
		},
		Verrazzano{},
		helmComponent{
			releaseName:             "coherence-operator",
			chartDir:                filepath.Join(config.Get().ThirdpartyChartsDir, "coherence-operator"),
			chartNamespace:          "verrazzano-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "coherence-values.yaml"),
		},
		helmComponent{
			releaseName:             "weblogic-operator",
			chartDir:                filepath.Join(thirdpartyDir, "weblogic-operator"),
			chartNamespace:          "verrazzano-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "weblogic-values.yaml"),
		},
		helmComponent{
			releaseName:             "mysql",
			chartDir:                filepath.Join(thirdpartyDir, "mysql"),
			chartNamespace:          "keycloak",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "mysql-values.yaml"),
		},
		helmComponent{
			releaseName:             "keycloak",
			chartDir:                filepath.Join(thirdpartyDir, "keycloak"),
			chartNamespace:          "keycloak",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "keycloak-values.yaml"),
		},
	}
}
