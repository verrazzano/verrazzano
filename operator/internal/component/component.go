// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"github.com/verrazzano/verrazzano/operator/internal/config"
	"path/filepath"
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
			releaseName:             "external-dns",
			chartDir:                filepath.Join(config.Get().ThirdpartyChartsDir, "external-dns"),
			chartNamespace:          "cert-manager",
			allowsNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "external-dns-values.yaml"),
		},
		helmComponent{
			releaseName:             "istio",
			chartDir:                filepath.Join(config.Get().ThirdpartyChartsDir, "istio"),
			chartNamespace:          "istio",
			allowsNamespaceOverride: true,
			valuesFile:              filepath.Join(componentDir, "external-dns-values.yaml"),
		},
	}
}
