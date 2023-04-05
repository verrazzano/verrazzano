// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dnsmc

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"
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

var beforeSuite = t.BeforeSuiteFunc(func() {
	adminCluster = multicluster.AdminCluster()
	managedClusters = multicluster.ManagedClusters()
	verifyRegistration()
})

var _ = BeforeSuite(beforeSuite)

var _ = t.Describe("Update managed-cluster dns", Serial, Ordered, Label("f:platform-lcm.update"), func() {
	t.Describe("multicluster dns verify", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("managed-cluster dns wildcard sslip.io config", func() {
			updateManagedClusterDNS()
			verifyPrometheusIngress()
			verifyThanosIngress()
			verifyThanosStore()
		})
	})
	t.Describe("multicluster dns verify", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("managed-cluster dns default nip.io config", func() {
			updateManagedClusterDNS()
			verifyPrometheusIngress()
			verifyThanosIngress()
			verifyThanosStore()
		})
	})
})

var oldPromIngs = map[string]string{}
var oldThanosIngs = map[string]string{}

// updateManagedClusterDNS switch dns config to sslip.io or nip.io
func updateManagedClusterDNS() {
	for _, managedCluster := range managedClusters {
		var oldDNS = managedCluster.GetCR(true).Spec.Components.DNS
		var newDNS *vzapi.DNSComponent
		if pkg.IsDefaultDNS(oldDNS) {
			newDNS = &vzapi.DNSComponent{Wildcard: &vzapi.Wildcard{Domain: pkg.SslipDomain}}
		}
		oldPromIngs[managedCluster.Name] = managedCluster.GetPrometheusIngress()
		oldThanosIngs[managedCluster.Name] = managedCluster.GetThanosIngress()
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

func verifyThanosIngress() {
	start := time.Now()
	for _, managedCluster := range managedClusters {
		oldThanosIng := oldThanosIngs[managedCluster.Name]
		gomega.Eventually(func() bool {
			newThanosIng := managedCluster.GetThanosIngress()
			pkg.Log(pkg.Info, fmt.Sprintf("Cluster %v ThanosIngress updated %v from %v to %v for %v", managedCluster.Name, newThanosIng != oldThanosIng, oldThanosIng, newThanosIng, time.Since(start)))
			return newThanosIng != oldThanosIng
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("%s of %s is not updated for %v", oldThanosIng, managedCluster.Name, time.Since(start)))
	}
}

func verifyThanosStore() {
	for _, managedCluster := range managedClusters {
		gomega.Eventually(func() (bool, error) {
			metricsTest, err := pkg.NewMetricsTest([]string{adminCluster.KubeConfigPath, managedCluster.KubeConfigPath}, adminCluster.KubeConfigPath, map[string]string{})
			if err != nil {
				t.Logs.Errorf("Failed to create metrics test object for cluster: %v", err)
				return false, err
			}

			queryStores, err := metricsTest.Source.GetTargets()
			if err != nil {
				t.Logs.Errorf("Failed to create get metrics target source: %v", err)
				return false, err
			}

			expectedName := fmt.Sprintf("%s:443", managedCluster.GetQueryIngress())
			for _, store := range queryStores {
				storeMap, ok := store.(map[string]interface{})
				if !ok {
					t.Logs.Infof("Thanos store empty, skipping entry")
					continue
				}
				name, ok := storeMap["name"]
				if !ok {
					t.Logs.Infof("Name not found for store, skipping entry")
					continue
				}
				nameString, nameOk := name.(string)
				if !nameOk {
					t.Logs.Infof("Name not valid format, skipping entry")
					continue
				}
				if ok {
					t.Logs.Infof("Found store in Thanos %s, want is equal to %s", nameString, expectedName)
				}
				if ok && nameString == expectedName {
					return true, nil
				}
			}
			return false, nil
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("store of %s is not ready", managedCluster.Name))
	}
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
