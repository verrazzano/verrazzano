// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package registry

import (
	"fmt"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/coherence"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/oam"
	"path/filepath"

	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/appoper"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/helm"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/istio"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/keycloak"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/spi"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/verrazzano"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/weblogic"
	"github.com/verrazzano/verrazzano/platform-operator/internal/config"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/verrazzano/verrazzano/application-operator/constants"
)

// GetComponents returns the list of components that are installable and upgradeable.
// The components will be processed in the order items in the array
func GetComponents() []spi.Component {
	overridesDir := config.GetHelmOverridesDir()
	helmChartsDir := config.GetHelmChartsDir()
	thirdPartyChartsDir := config.GetThirdPartyDir()

	return []spi.Component{
		// TODO: remove istio helm components
		helm.HelmComponent{
			ReleaseName:             "istio-base",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "istio/base"),
			ChartNamespace:          "istio-system",
			IgnoreNamespaceOverride: true,
			IgnoreImageOverrides:    true,
		},
		helm.HelmComponent{
			ReleaseName:             "istiod",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "istio/istio-control/istio-discovery"),
			ChartNamespace:          "istio-system",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(overridesDir, "istio-values.yaml"),
			AppendOverridesFunc:     istio.AppendIstioOverrides,
			ReadyStatusFunc:         istio.IstiodReadyCheck,
		},
		helm.HelmComponent{
			ReleaseName:             "istio-ingress",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "istio/gateways/istio-ingress"),
			ChartNamespace:          "istio-system",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(overridesDir, "istio-values.yaml"),
			AppendOverridesFunc:     istio.AppendIstioOverrides,
		},
		helm.HelmComponent{
			ReleaseName:             "istio-egress",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "istio/gateways/istio-egress"),
			ChartNamespace:          "istio-system",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(overridesDir, "istio-values.yaml"),
			AppendOverridesFunc:     istio.AppendIstioOverrides,
		},
		helm.HelmComponent{
			ReleaseName:             "istiocoredns",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "istio/istiocoredns"),
			ChartNamespace:          "istio-system",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(overridesDir, "istio-values.yaml"),
			AppendOverridesFunc:     istio.AppendIstioOverrides,
		},
		helm.HelmComponent{
			ReleaseName:             "ingress-controller",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "ingress-nginx"),
			ChartNamespace:          "ingress-nginx",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(overridesDir, "ingress-nginx-values.yaml"),
		},
		helm.HelmComponent{
			ReleaseName:             "cert-manager",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "cert-manager"),
			ChartNamespace:          "cert-manager",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(overridesDir, "cert-manager-values.yaml"),
		},
		helm.HelmComponent{
			ReleaseName:             "external-dns",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "external-dns"),
			ChartNamespace:          "cert-manager",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(overridesDir, "external-dns-values.yaml"),
		},
		helm.HelmComponent{
			ReleaseName:             "rancher",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "rancher"),
			ChartNamespace:          "cattle-system",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(overridesDir, "rancher-values.yaml"),
		},
		helm.HelmComponent{
			ReleaseName:             "verrazzano",
			ChartDir:                filepath.Join(helmChartsDir, "verrazzano"),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			ResolveNamespaceFunc:    verrazzano.ResolveVerrazzanoNamespace,
			PreUpgradeFunc:          verrazzano.VerrazzanoPreUpgrade,
		},
		helm.HelmComponent{
			ReleaseName:             "coherence-operator",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "coherence-operator"),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  "imagePullSecrets[0].name",
			ValuesFile:              filepath.Join(overridesDir, "coherence-values.yaml"),
			ReadyStatusFunc:         coherence.IsCoherenceOperatorReady,
		},
		helm.HelmComponent{
			ReleaseName:             "weblogic-operator",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "weblogic-operator"),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ImagePullSecretKeyname:  "imagePullSecrets[0].name",
			ValuesFile:              filepath.Join(overridesDir, "weblogic-values.yaml"),
			PreInstallFunc:          weblogic.WeblogicOperatorPreInstall,
			AppendOverridesFunc:     weblogic.AppendWeblogicOperatorOverrides,
			Dependencies:            []string{"istiod"},
			ReadyStatusFunc:         weblogic.IsWeblogicOperatorReady,
		},
		helm.HelmComponent{
			ReleaseName:             "oam-kubernetes-runtime",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "oam-kubernetes-runtime"),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ValuesFile:              filepath.Join(overridesDir, "oam-kubernetes-runtime-values.yaml"),
			ImagePullSecretKeyname:  "imagePullSecrets[0].name",
			ReadyStatusFunc:         oam.IsOAMReady,
		},
		helm.HelmComponent{
			ReleaseName:             "verrazzano-application-operator",
			ChartDir:                filepath.Join(helmChartsDir, "verrazzano-application-operator"),
			ChartNamespace:          constants.VerrazzanoSystemNamespace,
			IgnoreNamespaceOverride: true,
			SupportsOperatorInstall: true,
			ValuesFile:              filepath.Join(overridesDir, "verrazzano-application-operator-values.yaml"),
			AppendOverridesFunc:     appoper.AppendApplicationOperatorOverrides,
			ImagePullSecretKeyname:  "global.imagePullSecrets[0]",
			ReadyStatusFunc:         appoper.IsApplicationOperatorReady,
			Dependencies:            []string{"oam-kubernetes-runtime"},
		},
		helm.HelmComponent{
			ReleaseName:             "mysql",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "mysql"),
			ChartNamespace:          "keycloak",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(overridesDir, "mysql-values.yaml"),
		},
		helm.HelmComponent{
			ReleaseName:             "keycloak",
			ChartDir:                filepath.Join(thirdPartyChartsDir, "keycloak"),
			ChartNamespace:          "keycloak",
			IgnoreNamespaceOverride: true,
			ValuesFile:              filepath.Join(overridesDir, "keycloak-values.yaml"),
			AppendOverridesFunc:     keycloak.AppendKeycloakOverrides,
		},
		// istio upgrade code still in development so cannot have IstioComponent instance in the registry yet
		// istio.IstioComponent{},
	}
}

func FindComponent(releaseName string) (bool, spi.Component) {
	for _, comp := range GetComponents() {
		if comp.Name() == releaseName {
			return true, comp
		}
	}
	return false, &helm.HelmComponent{}
}

// ComponentDependenciesMet Checks if the declared dependencies for the component are ready and available
func ComponentDependenciesMet(log *zap.SugaredLogger, client client.Client, c spi.Component, dryRun bool) bool {
	trace, err := checkDependencies(log, client, c, dryRun, nil)
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
func checkDependencies(log *zap.SugaredLogger, client client.Client, c spi.Component, dryRun bool, trace map[string]bool) (map[string]bool, error) {
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
		if trace, err := checkDependencies(log, client, dependency, dryRun, trace); err != nil {
			return trace, err
		}
		if !dependency.IsReady(log, client, dependencyName, dryRun) {
			trace[dependencyName] = false // dependency is not ready
			continue
		}
		trace[dependencyName] = true // dependency is ready
	}
	return trace, nil
}
