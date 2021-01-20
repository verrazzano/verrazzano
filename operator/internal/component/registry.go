// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/operator/internal/config"
	"path/filepath"
)

// GetComponents returns the list of components that are installable and upgradeable.
// The components will be processed in the order items in the array
func GetComponents() []Component {
	componentDir := filepath.Join(config.Get().VerrazzanoInstallDir, "components")
	helmConfigDir := filepath.Join(config.Get().HelmConfigDir, "components")

	return []Component{
		Verrazzano{},
		helmComponent{
			releaseName:             "verrazzano-application-operator",
			chartDir:                filepath.Join(helmConfigDir, "charts/verrazzano-application-operator"),
			chartNamespace:          "verrazzano-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(helmConfigDir, "overrides/verrazzano-application-operator.yaml"),
		},
		Nginx{},
		helmComponent{
			releaseName:             "cert-manager",
			chartDir:                filepath.Join(config.Get().ThirdpartyChartsDir, "cert-manager"),
			chartNamespace:          "cert-manager",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "cert-manager-values.yaml"),
		},
		helmComponent{
			releaseName:             "external-dns",
			chartDir:                filepath.Join(config.Get().ThirdpartyChartsDir, "external-dns"),
			chartNamespace:          "cert-manager",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "external-dns-values.yaml"),
		},
		helmComponent{
			releaseName:             "keycloak",
			chartDir:                filepath.Join(config.Get().ThirdpartyChartsDir, "keycloak"),
			chartNamespace:          "keycloak",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "keycloak-values.yaml"),
		},
		helmComponent{
			releaseName:             "rancher",
			chartDir:                filepath.Join(config.Get().ThirdpartyChartsDir, "rancher"),
			chartNamespace:          "cattle-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "rancher-values.yaml"),
		},
		helmComponent{
			releaseName:             "istio",
			chartDir:                filepath.Join(config.Get().ThirdpartyChartsDir, "istio"),
			chartNamespace:          "istio-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "istio-values.yaml"),
			preUpgradeFunc:          PreUpgrade,
		},
		helmComponent{
			releaseName:             "coherence-operator",
			chartDir:                filepath.Join(config.Get().ThirdpartyChartsDir, "coherence-operator"),
			chartNamespace:          "verrazzano-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "coherence-values.yaml"),
		},
	}
}
