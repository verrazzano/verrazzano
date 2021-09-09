// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"path/filepath"

	"github.com/verrazzano/verrazzano/application-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
)

// GetComponents returns the list of components that are installable and upgradeable.
// The components will be processed in the order items in the array
func GetComponents() []Component {
	overridesDir := config.GetHelmOverridesDir()
	helmChartsDir := config.GetHelmChartsDir()
	thirdPartyChartsDir := config.GetThirdPartyDir()

	return []Component{
		// TODO: remove istio helm components
		helmComponent{
			releaseName:             "istio-base",
			chartDir:                filepath.Join(thirdPartyChartsDir, "istio/base"),
			chartNamespace:          "istio-system",
			ignoreNamespaceOverride: true,
			ignoreImageOverrides:    true,
		},
		helmComponent{
			releaseName:             "istiod",
			chartDir:                filepath.Join(thirdPartyChartsDir, "istio/istio-control/istio-discovery"),
			chartNamespace:          "istio-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "istio-values.yaml"),
			appendOverridesFunc:     appendIstioOverrides,
		},
		helmComponent{
			releaseName:             "istio-ingress",
			chartDir:                filepath.Join(thirdPartyChartsDir, "istio/gateways/istio-ingress"),
			chartNamespace:          "istio-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "istio-values.yaml"),
			appendOverridesFunc:     appendIstioOverrides,
		},
		helmComponent{
			releaseName:             "istio-egress",
			chartDir:                filepath.Join(thirdPartyChartsDir, "istio/gateways/istio-egress"),
			chartNamespace:          "istio-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "istio-values.yaml"),
			appendOverridesFunc:     appendIstioOverrides,
		},
		helmComponent{
			releaseName:             "istiocoredns",
			chartDir:                filepath.Join(thirdPartyChartsDir, "istio/istiocoredns"),
			chartNamespace:          "istio-system",
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "istio-values.yaml"),
			appendOverridesFunc:     appendIstioOverrides,
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
		helmComponent{
			releaseName:             "verrazzano",
			chartDir:                filepath.Join(helmChartsDir, "verrazzano"),
			chartNamespace:          constants.VerrazzanoSystemNamespace,
			ignoreNamespaceOverride: true,
			resolveNamespaceFunc:    resolveVerrazzanoNamespace,
			preUpgradeFunc:          verrazzanoPreUpgrade,
		},
		helmComponent{
			releaseName:             "coherence-operator",
			chartDir:                filepath.Join(thirdPartyChartsDir, "coherence-operator"),
			chartNamespace:          constants.VerrazzanoSystemNamespace,
			ignoreNamespaceOverride: true,
			supportsOperatorInstall: true,
			waitForInstall:          true,
			imagePullSecretKeyname:  "imagePullSecrets[0].name",
			valuesFile:              filepath.Join(overridesDir, "coherence-values.yaml"),
		},
		helmComponent{
			releaseName:             "weblogic-operator",
			chartDir:                filepath.Join(thirdPartyChartsDir, "weblogic-operator"),
			chartNamespace:          constants.VerrazzanoSystemNamespace,
			ignoreNamespaceOverride: true,
			valuesFile:              filepath.Join(overridesDir, "weblogic-values.yaml"),
		},
		helmComponent{
			releaseName:             "oam-kubernetes-runtime",
			chartDir:                filepath.Join(thirdPartyChartsDir, "oam-kubernetes-runtime"),
			chartNamespace:          constants.VerrazzanoSystemNamespace,
			ignoreNamespaceOverride: true,
			supportsOperatorInstall: true,
			waitForInstall:          true,
			valuesFile:              filepath.Join(overridesDir, "oam-kubernetes-runtime-values.yaml"),
			imagePullSecretKeyname:  "imagePullSecrets[0].name",
		},
		helmComponent{
			releaseName:             "verrazzano-application-operator",
			chartDir:                filepath.Join(helmChartsDir, "verrazzano-application-operator"),
			chartNamespace:          constants.VerrazzanoSystemNamespace,
			ignoreNamespaceOverride: true,
			supportsOperatorInstall: true,
			waitForInstall:          true,
			valuesFile:              filepath.Join(overridesDir, "verrazzano-application-operator-values.yaml"),
			appendOverridesFunc:     appendApplicationOperatorOverrides,
			imagePullSecretKeyname:  "global.imagePullSecrets[0]",
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
			appendOverridesFunc:     appendKeycloakOverrides,
		},
		istioComponent{
			componentName: "istio",
		},
	}
}
