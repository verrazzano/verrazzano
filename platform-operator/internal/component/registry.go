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
	overridesDir := filepath.Join(config.Get().HelmConfigDir, "overrides")
	vzChartsDir := filepath.Join(config.Get().HelmConfigDir, "charts")
	thirdPartyChartsDir := config.Get().ThirdpartyChartsDir

	return []Component{
		helmComponent{
			releaseName:             "istio",
			chartDir:                filepath.Join(thirdPartyChartsDir, "istio"),
			chartNamespace:          "istio-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "istio-values.yaml"),
			preUpgradeFunc:          PreUpgrade,
		},
		helmComponent{
			releaseName:             "ingress-controller",
			chartDir:                filepath.Join(thirdPartyChartsDir, "ingress-nginx"),
			chartNamespace:          "ingress-nginx",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "ingress-nginx-values.yaml"),
		},
		helmComponent{
			releaseName:             "cert-manager",
			chartDir:                filepath.Join(thirdPartyChartsDir, "cert-manager"),
			chartNamespace:          "cert-manager",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "cert-manager-values.yaml"),
		},
		helmComponent{
			releaseName:             "external-dns",
			chartDir:                filepath.Join(thirdPartyChartsDir, "external-dns"),
			chartNamespace:          "cert-manager",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "external-dns-values.yaml"),
		},
		helmComponent{
			releaseName:             "rancher",
			chartDir:                filepath.Join(thirdPartyChartsDir, "rancher"),
			chartNamespace:          "cattle-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "rancher-values.yaml"),
		},
		Verrazzano{},
		helmComponent{
			releaseName:             "verrazzano-application-operator",
			chartDir:                filepath.Join(vzChartsDir, "verrazzano-application-operator"),
			chartNamespace:          "verrazzano-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "verrazzano-application-operator-values.yaml"),
		},
		helmComponent{
			releaseName:             "coherence-operator",
			chartDir:                filepath.Join(thirdPartyChartsDir, "coherence-operator"),
			chartNamespace:          "verrazzano-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "coherence-values.yaml"),
		},
		helmComponent{
			releaseName:             "weblogic-operator",
			chartDir:                filepath.Join(thirdPartyChartsDir, "weblogic-operator"),
			chartNamespace:          "verrazzano-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "weblogic-values.yaml"),
		},
		helmComponent{
			releaseName:             "mysql",
			chartDir:                filepath.Join(thirdPartyChartsDir, "mysql"),
			chartNamespace:          "keycloak",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "mysql-values.yaml"),
		},
		helmComponent{
			releaseName:             "keycloak",
			chartDir:                filepath.Join(thirdPartyChartsDir, "keycloak"),
			chartNamespace:          "keycloak",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "keycloak-values.yaml"),
		},
	}
}
