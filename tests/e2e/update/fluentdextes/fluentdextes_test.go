// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package fluentdextes

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/constants"
	mcconst "github.com/verrazzano/verrazzano/pkg/mcconstants"
	vzapi "github.com/verrazzano/verrazzano/platform-operator/apis/verrazzano/v1alpha1"
	poconst "github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/update"
	"github.com/verrazzano/verrazzano/tests/e2e/update/fluentd"
	corev1 "k8s.io/api/core/v1"
)

var (
	t                = framework.NewTestFramework("update fluentd external opensearch")
	extOpensearchURL string
	extOpensearchSec string
	adminCluster     *multicluster.Cluster
	managedClusters  []*multicluster.Cluster
	orignalFluentd   *vzapi.FluentdComponent
	waitTimeout      = 10 * time.Minute
	pollingInterval  = 5 * time.Second
)

var _ = t.BeforeSuite(func() {
	cr := update.GetCR()
	orignalFluentd = cr.Spec.Components.Fluentd
	if orignalFluentd != nil { //External Collector is enabled
		extOpensearchURL = orignalFluentd.ElasticsearchURL
		extOpensearchSec = orignalFluentd.ElasticsearchSecret
	}
	adminCluster = multicluster.AdminCluster()
	managedClusters = multicluster.ManagedClusters()
})

var _ = t.AfterSuite(func() {
	if extOpensearchURL != "" && extOpensearchURL != pkg.VmiESURL && extOpensearchSec != "" {
		start := time.Now()
		gomega.Eventually(func() bool {
			return fluentd.ValidateDaemonset(extOpensearchURL, extOpensearchSec, "")
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("DaemonSet %s is not ready for %v", extOpensearchURL, time.Since(start)))
	}
})

var _ = t.Describe("Update Fluentd", Label("f:platform-lcm.update"), func() {
	t.Describe("Update to default Opensearch", Label("f:platform-lcm.fluentd-default-opensearch"), func() {
		t.It("default Opensearch", func() {
			if orignalFluentd != nil { //External Collector is enabled
				m := &fluentd.FluentdModifier{Component: vzapi.FluentdComponent{}}

				start := time.Now()
				fluentd.ValidateUpdate(m, "")

				gomega.Eventually(func() bool {
					return fluentd.ValidateDaemonset(pkg.VmiESURL, pkg.VmiESInternalSecret, "")
				}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("DaemonSet %s is not ready for %v", pkg.VmiESURL, time.Since(start)))
			}
		})
	})
	t.Describe("multicluster verify", Label("f:platform-lcm.multicluster-verify"), func() {
		t.It("default ca-bundle", func() {
			verifyCaSync("")
		})
	})
	t.Describe("Update to external Opensearch", Label("f:platform-lcm.fluentd-external-opensearch"), func() {
		t.It("external Opensearch", func() {
			pkg.Log(pkg.Info, fmt.Sprintf("Update fluentd to use %v and %v", extOpensearchURL, extOpensearchSec))
			if orignalFluentd != nil { //External Collector is enabled
				m := &fluentd.FluentdModifier{Component: *orignalFluentd}
				update.RetryUpdate(m, adminCluster.KubeConfigPath, false, pollingInterval, waitTimeout)

				start := time.Now()
				gomega.Eventually(func() bool {
					return fluentd.ValidateDaemonset(extOpensearchURL, extOpensearchSec, "")
				}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("DaemonSet %s is not ready for %v", extOpensearchURL, time.Since(start)))
				verifyCaSync(extOpensearchSec)
			}
		})
	})
})

func verifyCaSync(esSec string) {
	extEsCa := ""
	if esSec != "" && esSec != pkg.VmiESInternalSecret {
		bytes, _ := adminCluster.GetSecretData(poconst.VerrazzanoInstallNamespace, esSec, mcconst.FluentdOSCaBundleKey)
		if len(bytes) > 0 {
			extEsCa = string(bytes)
		}
	}
	for _, managedCluster := range managedClusters {
		reg := getRegistration(managedCluster)
		if reg != nil {
			gomega.Eventually(func() bool {
				return verifyCaBundles(reg, managedCluster, esSec, extEsCa)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("CA bundle in %s is not synced", esSec))
		}
	}
}

func getRegistration(managedCluster *multicluster.Cluster) *corev1.Secret {
	reg, _ := adminCluster.GetRegistration(managedCluster.Name)
	if reg == nil {
		adminCluster.Register(managedCluster)
		gomega.Eventually(func() bool {
			reg, _ := adminCluster.GetRegistration(managedCluster.Name)
			return reg != nil
		}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), fmt.Sprintf("%s is not registered", managedCluster.Name))
		reg, _ = adminCluster.GetRegistration(managedCluster.Name)
	}
	return reg
}

func verifyCaBundles(reg *corev1.Secret, managedCluster *multicluster.Cluster, esSec, extEsCa string) bool {
	admEsCa, regEsCa := caBundles(reg)
	mngEsCa := ""
	if extEsCa == "" {
		extEsCa = admEsCa
	}
	bytes, _ := managedCluster.
		GetSecretData(constants.VerrazzanoSystemNamespace, "verrazzano-cluster-registration", mcconst.OSCaBundleKey)
	pkg.Log(pkg.Info, fmt.Sprintf("Opensearch ca-bundle synced to registration:%v managed-cluster:%v", extEsCa == regEsCa, extEsCa == mngEsCa))
	if len(bytes) == 0 {
		//if the managed-cluster is NOT registered, verify only the ca in registration
		if extEsCa != regEsCa {
			pkg.Log(pkg.Info, fmt.Sprintf("Opensearch ca-bundle in %s is not synced to %v registration", esSec, managedCluster.Name))
			return false
		}
		return extEsCa == regEsCa
	}
	mngEsCa = string(bytes)
	if extEsCa != mngEsCa {
		pkg.Log(pkg.Info, fmt.Sprintf("ManagedCluster %v verrazzano-cluster-registration is not synced", managedCluster.Name))
		return false
	}
	return extEsCa == mngEsCa
}

func caBundles(reg *corev1.Secret) (string, string) {
	admEsCa, regEsCa := "", ""
	for k, v := range reg.Data {
		if k == mcconst.OSCaBundleKey {
			regEsCa = string(v)
		}
		if k == mcconst.AdminCaBundleKey {
			admEsCa = string(v)
		}
	}
	return admEsCa, regEsCa
}
