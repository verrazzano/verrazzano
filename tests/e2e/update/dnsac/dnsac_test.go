// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package dnsac

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg/update"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	corev1 "k8s.io/api/core/v1"
)

var (
	t               = framework.NewTestFramework("update dns")
	adminCluster    *multicluster.Cluster
	managedClusters []*multicluster.Cluster
	shortWait       = 5 * time.Minute
	longWait        = 10 * time.Minute
	pollingInterval = 5 * time.Second
	fluentdName     = "fluentd"
	adminFluentd    *vzapi.FluentdComponent
)

type DNSModifier struct {
	DNS *vzapi.DNSComponent
}

func (m *DNSModifier) ModifyCR(cr *vzapi.Verrazzano) {
	cr.Spec.Components.DNS = m.DNS
}

var _ = t.BeforeSuite(func() {
	cr := update.GetCR()
	adminFluentd = cr.Spec.Components.Fluentd
	adminCluster = multicluster.AdminCluster()
	managedClusters = multicluster.ManagedClusters()
	verifyRegistration()
})

var _ = t.Describe("Update admin-cluster dns", Serial, Ordered, Label("f:platform-lcm.update"), func() {
	t.Describe("multicluster dns verify", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("managed-cluster dns wildcard sslip.io config", func() {
			if systemOpenSearch() {
				oldEsIng := updateAdminClusterDNS()
				newEsIng := verifyOpenSearchIngress(oldEsIng)
				verifyManagedClusterFluentd(newEsIng)
			} else {
				pkg.Log(pkg.Info, fmt.Sprintf(
					"Skip %v dns update test as custom log collecter cannot be used to assert ednpoint URL update", adminCluster.Name))
			}
		})
	})
	t.Describe("multicluster dns verify", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("managed-cluster dns default nip.io config", func() {
			if systemOpenSearch() {
				oldEsIng := updateAdminClusterDNS()
				newEsIng := verifyOpenSearchIngress(oldEsIng)
				verifyManagedClusterFluentd(newEsIng)
			} else {
				pkg.Log(pkg.Info, fmt.Sprintf(
					"Skip %v dns update test as custom log collecter cannot be used to assert ednpoint URL update", adminCluster.Name))
			}
		})
	})
})

// updateAdminClusterDNS switch dns config to sslip.io or nip.io
func updateAdminClusterDNS() string {
	adminVZ := adminCluster.GetCR(true)
	var oldDNS = adminVZ.Spec.Components.DNS
	var newDNS *vzapi.DNSComponent
	var domainOld, domainNew = pkg.NipDomain, pkg.SslipDomain
	oldEsIng := pkg.GetSystemOpenSearchIngressURL(adminCluster.KubeConfigPath)
	if pkg.IsDefaultDNS(oldDNS) {
		newDNS = &vzapi.DNSComponent{Wildcard: &vzapi.Wildcard{Domain: pkg.SslipDomain}}
	} else {
		domainOld, domainNew = pkg.SslipDomain, pkg.NipDomain
	}
	m := &DNSModifier{DNS: newDNS}
	gomega.Expect(strings.Contains(oldEsIng, domainOld)).Should(gomega.BeTrue())
	gomega.Expect(strings.Contains(oldEsIng, domainNew)).Should(gomega.BeFalse())
	update.RetryUpdate(m, adminCluster.KubeConfigPath, false, pollingInterval, shortWait)
	return oldEsIng
}

func systemOpenSearch() bool {
	return !pkg.UseExternalElasticsearch() &&
		(adminFluentd == nil || reflect.DeepEqual(*adminFluentd, vzapi.FluentdComponent{}))
}

func verifyOpenSearchIngress(oldEsIng string) string {
	start := time.Now()
	gomega.Eventually(func() bool {
		newEsIng := pkg.GetSystemOpenSearchIngressURL(adminCluster.KubeConfigPath)
		return newEsIng != oldEsIng
	}, longWait, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("admin-cluster %s is not updated for %v", oldEsIng, time.Since(start)))
	return pkg.GetSystemOpenSearchIngressURL(adminCluster.KubeConfigPath)
}

func verifyManagedClusterFluentd(newEsIng string) {
	start := time.Now()
	for _, managedCluster := range managedClusters {
		gomega.Eventually(func() bool {
			fp := managedCluster.FindFluentdPod()
			if fp == nil || len(fp.Spec.Containers) == 0 {
				return false
			}
			esURL := findEsURL(fp)
			pkg.Log(pkg.Info, fmt.Sprintf("Cluster %v Fluentd OpenSearch URL updated: %v from %v to %v for %v \n", managedCluster.Name, strings.Contains(esURL, newEsIng), esURL, newEsIng, time.Since(start)))
			return strings.Contains(esURL, newEsIng)
		}, longWait, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("%s Fluentd is not updated for %v", managedCluster.Name, time.Since(start)))
	}
}

func findEsURL(fp *corev1.Pod) string {
	for _, c := range fp.Spec.Containers {
		if c.Name == fluentdName {
			for _, env := range c.Env {
				if env.Name == "ELASTICSEARCH_URL" {
					return env.Value
				}
			}
		}
	}
	return ""
}

func verifyRegistration() {
	for _, managedCluster := range managedClusters {
		reg, _ := adminCluster.GetRegistration(managedCluster.Name)
		if reg == nil {
			adminCluster.Register(managedCluster)
			gomega.Eventually(func() bool {
				reg, err := adminCluster.GetRegistration(managedCluster.Name)
				return reg != nil && err == nil
			}, longWait, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("%s is not registered", managedCluster.Name))
		}
	}
}
