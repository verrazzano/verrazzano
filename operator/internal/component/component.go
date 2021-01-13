// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/operator/internal/config"
)

// Component interface defines the methods implemented by components
type Component interface {
	// Name returns the name of the Verrazzano component
	Name() string

	// Upgrade will upgrade the Verrazzano component specified in the CR.Version field
	Upgrade(namespace string) error
}

// GetComponents returns the list of components that are installable and upgradeable.
// The components will be processed in the order items in the array
func GetComponents() []Component {
	componentDir := filepath.Join(config.Get().VerrazzanoInstallDir, "components")

	return []Component{
		Verrazzano{},
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
	}
}
