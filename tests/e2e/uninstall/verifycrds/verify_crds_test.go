// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verifycrds

import (
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Expected verrazzano.io CRDs after uninstall
var verrazzanoiocrds = map[string]bool{
	"ingresstraits.oam.verrazzano.io":                              false,
	"loggingtraits.oam.verrazzano.io":                              false,
	"metricsbindings.app.verrazzano.io":                            false,
	"metricstemplates.app.verrazzano.io":                           false,
	"metricstraits.oam.verrazzano.io":                              false,
	"multiclusterapplicationconfigurations.clusters.verrazzano.io": false,
	"multiclustercomponents.clusters.verrazzano.io":                false,
	"multiclusterconfigmaps.clusters.verrazzano.io":                false,
	"multiclustersecrets.clusters.verrazzano.io":                   false,
	"verrazzanocoherenceworkloads.oam.verrazzano.io":               false,
	"verrazzanohelidonworkloads.oam.verrazzano.io":                 false,
	"verrazzanomonitoringinstances.verrazzano.io":                  false,
	"verrazzanoprojects.clusters.verrazzano.io":                    false,
	"verrazzanoweblogicworkloads.oam.verrazzano.io":                false,
}

// These CRDs are not deleted when using vz uninstall but are deleted when deleting the platform-operator.yaml.
// Therefore, they may or may not be present after an uninstall.
var optionalverrazzanoiocrds = []string{
	"verrazzanomanagedclusters.clusters.verrazzano.io",
	"verrazzanos.install.verrazzano.io",
}

// Expected istio.io CRDs after uninstall
var istioiocrds = map[string]bool{
	"authorizationpolicies.security.istio.io":  false,
	"destinationrules.networking.istio.io":     false,
	"envoyfilters.networking.istio.io":         false,
	"gateways.networking.istio.io":             false,
	"istiooperators.install.istio.io":          false,
	"peerauthentications.security.istio.io":    false,
	"proxyconfigs.networking.istio.io":         false,
	"requestauthentications.security.istio.io": false,
	"serviceentries.networking.istio.io":       false,
	"sidecars.networking.istio.io":             false,
	"telemetries.telemetry.istio.io":           false,
	"virtualservices.networking.istio.io":      false,
	"wasmplugins.extensions.istio.io":          false,
	"workloadentries.networking.istio.io":      false,
	"workloadgroups.networking.istio.io":       false,
}

// Expected oam.dev CRDs after uninstall
var oamdevcrds = map[string]bool{
	"applicationconfigurations.core.oam.dev": false,
	"components.core.oam.dev":                false,
	"containerizedworkloads.core.oam.dev":    false,
	"healthscopes.core.oam.dev":              false,
	"manualscalertraits.core.oam.dev":        false,
	"scopedefinitions.core.oam.dev":          false,
	"traitdefinitions.core.oam.dev":          false,
	"workloaddefinitions.core.oam.dev":       false,
}

// Expected cert-manager.io CRDs after uninstall
var certmanageriocrds = map[string]bool{
	"certificaterequests.cert-manager.io": false,
	"certificates.cert-manager.io":        false,
	"challenges.acme.cert-manager.io":     false,
	"clusterissuers.cert-manager.io":      false,
	"issuers.cert-manager.io":             false,
	"orders.acme.cert-manager.io":         false,
}

// Expected monitoring.coreis.com CRDs after uninstall
var monitoringcoreoscomcrds = map[string]bool{
	"alertmanagerconfigs.monitoring.coreos.com": false,
	"alertmanagers.monitoring.coreos.com":       false,
	"podmonitors.monitoring.coreos.com":         false,
	"probes.monitoring.coreos.com":              false,
	"prometheuses.monitoring.coreos.com":        false,
	"prometheusrules.monitoring.coreos.com":     false,
	"servicemonitors.monitoring.coreos.com":     false,
	"thanosrulers.monitoring.coreos.com":        false,
}

// Expected MySQL Operator CRDs after uninstall
var mysqloperatorcrds = map[string]bool{
	"innodbclusters.mysql.oracle.com": false,
	"mysqlbackups.mysql.oracle.com":   false,
	"clusterkopfpeerings.zalando.org": false,
	"kopfpeerings.zalando.org":        false,
}
var t = framework.NewTestFramework("uninstall verify crds")

// This test verifies the CRDs found after an uninstall of Verrazzano are what is expected
var _ = t.Describe("Verify CRDs after uninstall.", Label("f:platform-lcm.unnstall"), func() {
	crds, err := pkg.ListCRDs()
	if err != nil {
		Fail(err.Error())
	}

	t.It("Check for expected verrazzano.io CRDs", func() {
		checkCrds(crds, verrazzanoiocrds, "verrazzano.io")
	})

	t.It("Check for expected istio.io CRDs", func() {
		checkCrds(crds, istioiocrds, "istio.io")
	})

	t.It("Check for expected oam.dev CRDs", func() {
		checkCrds(crds, oamdevcrds, "oam.dev")
	})

	t.It("Check for expected cert-manager.io CRDs", func() {
		checkCrds(crds, certmanageriocrds, "cert-manager.io")
	})

	t.It("Check for expected monitoring.coreos.com CRDs", func() {
		checkCrds(crds, monitoringcoreoscomcrds, "monitoring.coreos.com")
	})

	t.It("Check for expected domains.weblogic.oracle CRD", func() {
		checkCrds(crds, map[string]bool{"domains.weblogic.oracle": false}, "domains.weblogic.oracle")
	})

	t.It("Check for expected coherence.coherence.oracle.com CRD", func() {
		checkCrds(crds, map[string]bool{"coherence.coherence.oracle.com": false}, "coherence.coherence.oracle.com")
	})

	t.It("Check for expected MySQL Operator CRDs", func() {
		checkCrds(crds, mysqloperatorcrds, "mysql.oracle.com")
		checkCrds(crds, mysqloperatorcrds, "zalando.org")
	})

	t.It("Check for unexpected CRDs", func() {
		var crdsFound = make(map[string]bool)
		for _, crd := range crds.Items {
			// Anything other than these CRDs being checked are unexpected after an uninstall
			if strings.HasSuffix(crd.Name, "projectcalico.org") ||
				strings.HasSuffix(crd.Name, "verrazzano.io") ||
				strings.HasSuffix(crd.Name, "istio.io") ||
				strings.HasSuffix(crd.Name, "monitoring.coreos.com") ||
				strings.HasSuffix(crd.Name, "oam.dev") ||
				strings.HasSuffix(crd.Name, "cert-manager.io") ||
				strings.HasSuffix(crd.Name, "cluster.x-k8s.io") ||
				strings.HasSuffix(crd.Name, "cattle.io") ||
				strings.HasSuffix(crd.Name, "mysql.oracle.com") ||
				strings.HasSuffix(crd.Name, "zalando.org") ||
				crd.Name == "monitoringdashboards.monitoring.kiali.io" ||
				crd.Name == "domains.weblogic.oracle" ||
				crd.Name == "coherence.coherence.oracle.com" {
				crdsFound[crd.Name] = true
				continue
			}
			crdsFound[crd.Name] = false
		}

		unexpectedCrd := false
		for key, value := range crdsFound {
			if value == false {
				unexpectedCrd = true
				pkg.Log(pkg.Error, fmt.Sprintf("Unexpected CRD was found: %s", key))
			}
		}
		if unexpectedCrd {
			Fail("Unexpected CRDs were found in the cluster")
		}
	})
})

// checkCRds checks for both expected CRDs and unexpected CRDs for a given CRDs suffix (for example, verrazzano.io)
func checkCrds(crds *apiextv1.CustomResourceDefinitionList, expectdCrds map[string]bool, suffix string) {
	unexpectedCrd := false
	for _, crd := range crds.Items {
		_, ok := expectdCrds[crd.Name]
		if ok {
			expectdCrds[crd.Name] = true
		} else {
			if strings.HasSuffix(crd.Name, suffix) {
				optionalCrdFound := false
				for _, optionalcrd := range optionalverrazzanoiocrds {
					if crd.Name == optionalcrd {
						optionalCrdFound = true
						break
					}
				}
				if optionalCrdFound {
					continue
				}
				unexpectedCrd = true
				pkg.Log(pkg.Error, fmt.Sprintf("Unexpected CRD was found: %s", crd.Name))
			}
		}
	}

	crdNotFound := false
	for key, value := range expectdCrds {
		if !value {
			crdNotFound = true
			pkg.Log(pkg.Error, fmt.Sprintf("Expected CRD was not found: %s", key))
		}
	}

	if unexpectedCrd || crdNotFound {
		Fail(fmt.Sprintf("Failed to verify %s CRDs", suffix))
	}
}
