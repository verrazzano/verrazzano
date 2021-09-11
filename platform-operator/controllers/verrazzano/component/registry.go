// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package component

import (
	"fmt"
	"go.uber.org/zap"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
			readyStatusFunc:         istiodReadyCheck,
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
			supportsOperatorInstall: true,
			waitForInstall:          true,
			imagePullSecretKeyname:  "imagePullSecrets[0].name",
			valuesFile:              filepath.Join(overridesDir, "weblogic-values.yaml"),
			preInstallFunc:          weblogicOperatorPreInstall,
			appendOverridesFunc:     appendWeblogicOperatorOverrides,
			dependencies:            []string{"istiod"},
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

func FindComponent(releaseName string) (bool, Component) {
	for _, comp := range GetComponents() {
		if comp.Name() == releaseName {
			return true, comp
		}
	}
	return false, &helmComponent{}
}

// ComponentDependenciesMet Checks if the declared dependencies for the component are ready and available
func ComponentDependenciesMet(log *zap.SugaredLogger, client client.Client, c Component) bool {
	trace, err := checkDependencies(log, client, c, nil)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	if len(trace) == 0 {
		log.Infof("No dependencies declared for %s", c.Name())
		return true
	}
	log.Infof("Trace results for %s: %v", c.Name(), trace)
	for _, value := range trace {
		if !value {
			return false
		}
	}
	return true
}

// checkDependencies Check the ready state of any dependencies and check for cycles
func checkDependencies(log *zap.SugaredLogger, client client.Client, c Component, trace map[string]bool) (map[string]bool, error) {
	for _, dependencyName := range c.GetDependencies() {
		if trace == nil {
			trace = make(map[string]bool)
		}
		if _, ok := trace[dependencyName]; ok {
			return trace, fmt.Errorf("Illegal state, dependency cycle found for %s: %s", c.Name(), dependencyName)
		}
		found, dependency := FindComponent(dependencyName)
		if !found {
			return trace, fmt.Errorf("Illegal state, declared dependency not found for %s: %s", c.Name(), dependencyName)
		}
		if trace, err := checkDependencies(log, client, dependency, trace); err != nil {
			return trace, err
		}
		if !dependency.IsReady(log, client, dependencyName) {
			trace[dependencyName] = false // dependency is not ready
			continue
		}
		trace[dependencyName] = true // dependency is ready
	}
	return trace, nil
}
