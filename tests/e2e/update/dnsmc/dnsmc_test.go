// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dnsmc

import (
	"fmt"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
)

var (
	t               = framework.NewTestFramework("update dns")
	adminCluster    *multicluster.Cluster
	managedClusters []*multicluster.Cluster
	waitTimeout     = 10 * time.Minute
	pollingInterval = 5 * time.Second
)

type DNSModifier struct {
	DNS *vzapi.DNSComponent
}

func (m *DNSModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.DNS = m.DNS
}

var _ = t.BeforeSuite(func() {
	adminCluster = multicluster.AdminCluster()
	managedClusters = multicluster.ManagedClusters()
	verifyRegistration()
})

var _ = t.Describe("Update managed-cluster dns", Serial, Ordered, Label("f:platform-lcm.update"), func() {
	t.Describe("multicluster dns verify", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("managed-cluster dns wildcard sslip.io config", func() {
			updateManagedClusterDNS()
			verifyPrometheusIngress()
			verifyScrapeTargets()
		})
	})
	t.Describe("multicluster dns verify", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("managed-cluster dns default nip.io config", func() {
			updateManagedClusterDNS()
			verifyPrometheusIngress()
			verifyScrapeTargets()
		})
	})
})

var oldPromIngs = map[string]string{}

// updateManagedClusterDNS switch dns config to sslip.io or nip.io
func updateManagedClusterDNS() {
	for _, managedCluster := range managedClusters {
		var oldDNS = managedCluster.GetCR(true).Spec.Components.DNS
		var newDNS *vzapi.DNSComponent
		if pkg.IsDefaultDNS(oldDNS) {
			newDNS = &vzapi.DNSComponent{Wildcard: &vzapi.Wildcard{Domain: pkg.SslipDomain}}
		}
		oldPromIngs[managedCluster.Name] = managedCluster.GetPrometheusIngress()
		m := &DNSModifier{DNS: newDNS}
		update.RetryUpdate(m, managedCluster.KubeConfigPath, false, pollingInterval, waitTimeout)
	}
}
func verifyPrometheusIngress() {
	start := time.Now()
	for _, managedCluster := range managedClusters {
		oldPromIng := oldPromIngs[managedCluster.Name]
		gomega.Eventually(func() bool {
			newPromIng := managedCluster.GetPrometheusIngress()
			pkg.Log(pkg.Info, fmt.Sprintf("Cluster %v PrometheusIngress updated %v from %v to %v for %v", managedCluster.Name, newPromIng != oldPromIng, oldPromIng, newPromIng, time.Since(start)))
			return newPromIng != oldPromIng
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("%s of %s is not updated for %v", oldPromIng, managedCluster.Name, time.Since(start)))
	}
}

func verifyScrapeTargets() {
	start := time.Now()
	for _, managedCluster := range managedClusters {
		gomega.Eventually(func() bool {
			managedClusterPromIng := managedCluster.GetPrometheusIngress()
			target := findScrapeTarget(managedCluster.Name)
			pkg.Log(pkg.Info, fmt.Sprintf("Cluster %v ScrapeTarget updated to %v %v %v for %v", managedCluster.Name, managedClusterPromIng, target["scrapeUrl"], target["health"], time.Since(start)))
			scrape, ok := target["scrapeUrl"]
			if ok && !strings.Contains(scrape.(string), managedClusterPromIng) {
				pkg.Log(pkg.Error, fmt.Sprintf("ScrapeTargets:%v scrapeUrl:%v expecting:%v time:%v \n", len(target), scrape, managedClusterPromIng, time.Since(start)))
				return false
			}
			health, ok := target["health"]
			if ok {
				pkg.Log(pkg.Error, fmt.Sprintf("ScrapeTargets:%v health:%v ok:%v time:%v \n", len(target), health, ok, time.Since(start)))
				return health.(string) == "up"
			}
			return false
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("scrape target of %s is not ready", managedCluster.Name))
	}
}

func findScrapeTarget(name string) map[string]interface{} {
	targets, err := pkg.ScrapeTargets()
	if err != nil {
		return map[string]interface{}{}
	}
	for _, item := range targets {
		t := item.(map[string]interface{})
		scrapePool := t["scrapePool"].(string)
		if scrapePool == name {
			return t
		}
	}
	return map[string]interface{}{}
}

func verifyRegistration() {
	for _, managedCluster := range managedClusters {
		reg, _ := adminCluster.GetRegistration(managedCluster.Name)
		if reg == nil {
			adminCluster.Register(managedCluster)
			gomega.Eventually(func() bool {
				reg, err := adminCluster.GetRegistration(managedCluster.Name)
				return reg != nil && err == nil
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("%s is not registered", managedCluster.Name))
		}
	}
}
