// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/operator/internal/config"
)

// GetComponents returns the list of components that are installable and upgradeable.
// The components will be processed in the order items in the array
func GetComponents() []Component {
	overrideDir := filepath.Join(config.Get().HelmConfigDir, "overrides")
	chartDir := filepath.Join(config.Get().HelmConfigDir, "charts")

	return []Component{
		helmComponent{
			releaseName:             "istio",
			chartDir:                filepath.Join(chartDir, "istio"),
			chartNamespace:          "istio-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overrideDir, "istio-values.yaml"),
			preUpgradeFunc:          PreUpgrade,
		},
		Nginx{},
		helmComponent{
			releaseName:             "cert-manager",
			chartDir:                filepath.Join(chartDir, "cert-manager"),
			chartNamespace:          "cert-manager",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overrideDir, "cert-manager-values.yaml"),
		},
		helmComponent{
			releaseName:             "external-dns",
			chartDir:                filepath.Join(chartDir, "external-dns"),
			chartNamespace:          "cert-manager",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overrideDir, "external-dns-values.yaml"),
		},
		helmComponent{
			releaseName:             "rancher",
			chartDir:                filepath.Join(chartDir, "rancher"),
			chartNamespace:          "cattle-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overrideDir, "rancher-values.yaml"),
		},
		Verrazzano{},
		helmComponent{
			releaseName:             "verrazzano-application-operator",
			chartDir:                filepath.Join(chartDir, "verrazzano-application-operator"),
			chartNamespace:          "verrazzano-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overrideDir, "verrazzano-application-operator-values.yaml"),
		},
		helmComponent{
			releaseName:             "coherence-operator",
			chartDir:                filepath.Join(chartDir, "coherence-operator"),
			chartNamespace:          "verrazzano-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overrideDir, "coherence-values.yaml"),
		},
		helmComponent{
			releaseName:             "weblogic-operator",
			chartDir:                filepath.Join(chartDir, "weblogic-operator"),
			chartNamespace:          "verrazzano-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overrideDir, "weblogic-values.yaml"),
		},
		helmComponent{
			releaseName:             "mysql",
			chartDir:                filepath.Join(chartDir, "mysql"),
			chartNamespace:          "keycloak",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overrideDir, "mysql-values.yaml"),
		},
		helmComponent{
			releaseName:             "keycloak",
			chartDir:                filepath.Join(chartDir, "keycloak"),
			chartNamespace:          "keycloak",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overrideDir, "keycloak-values.yaml"),
		},
	}
}
