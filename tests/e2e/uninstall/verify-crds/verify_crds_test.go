// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package verify_crds

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"strings"
)

var verrazzano_io_crds = map[string]bool{
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
	"verrazzanomanagedclusters.clusters.verrazzano.io":             false,
	"verrazzanomonitoringinstances.verrazzano.io":                  false,
	"verrazzanoprojects.clusters.verrazzano.io":                    false,
	"verrazzanos.install.verrazzano.io":                            false,
	"verrazzanoweblogicworkloads.oam.verrazzano.io":                false,
}

var istio_io_crds = map[string]bool{
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
var t = framework.NewTestFramework("uninstall verify crds")

// This test verifies the CRDs found after an uninstall of Verrazzano are what is expected
var _ = t.Describe("Verify CRDs after uninstall.", Label("f:platform-lcm.unnstall"), func() {
	crds, err := pkg.ListCRDs()
	if err != nil {
		Fail(err.Error())
	}

	t.It("Check for expected verrazzano.io CRDs", func() {
		unexpectedCrd := false
		for _, crd := range crds.Items {
			_, ok := verrazzano_io_crds[crd.Name]
			if ok {
				verrazzano_io_crds[crd.Name] = true
			} else {
				if strings.HasSuffix(crd.Name, ".verrazzano.io") {
					unexpectedCrd = true
					pkg.Log(pkg.Error, fmt.Sprintf("Unexpected verrazzano.io CRD was found: %s", crd.Name))
				}
			}
		}
		expectedCrdNotFound := checkExpectedCrds(verrazzano_io_crds)
		if unexpectedCrd || expectedCrdNotFound {
			Fail("Failed to verify verrazzano.io CRDs")
		}
	})

	t.It("Check for expected istio.io CRDs", func() {
		unexpectedCrd := false
		for _, crd := range crds.Items {
			_, ok := istio_io_crds[crd.Name]
			if ok {
				istio_io_crds[crd.Name] = true
			} else {
				if strings.HasSuffix(crd.Name, ".istio.io") {
					unexpectedCrd = true
					pkg.Log(pkg.Error, fmt.Sprintf("Unexpected istio.io CRD was found: %s", crd.Name))
				}
			}
		}
		expectedCrdNotFound := checkExpectedCrds(istio_io_crds)
		if unexpectedCrd || expectedCrdNotFound {
			Fail("Failed to verify istio.io CRDs")
		}
	})

	t.It("Check for unexpected CRDs", func() {
		var crdsFound = make(map[string]bool)
		for _, crd := range crds.Items {
			if strings.HasSuffix(crd.Name, ".projectcalico.org") ||
				strings.HasSuffix(crd.Name, ".verrazzano.io") ||
				strings.HasSuffix(crd.Name, ".istio.io") ||
				strings.HasSuffix(crd.Name, ".monitoring.coreos.com") ||
				strings.HasSuffix(crd.Name, ".oam.dev") ||
				strings.HasSuffix(crd.Name, ".cert-manager.io") ||
				strings.HasSuffix(crd.Name, ".cluster.x-k8s.io") ||
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

func checkExpectedCrds(crds map[string]bool) bool {
	crdNotFound := false
	for key, value := range crds {
		if !value {
			crdNotFound = true
			pkg.Log(pkg.Error, fmt.Sprintf("Expected CRD was not found: %s", key))
		}
	}
	return crdNotFound
}
