// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register_test

import (
	"fmt"
	"os"
	"time"

	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	"github.com/onsi/ginkgo"
)

const waitTimeout = 10 * time.Minute
const pollingInterval = 10 * time.Second

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")

var _ = ginkgo.Describe("Multi Cluster Verify Register", func() {
	ginkgo.Context("Admin Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		ginkgo.It("admin cluster has the expected secrets", func() {
			gomega.Eventually(func() bool {
				s, err := pkg.GetSecret("verrazzano-mc", fmt.Sprintf("verrazzano-cluster-%s-manifest", managedClusterName))
				if s == nil || err != nil {
					return false
				}
				s, err = pkg.GetSecret("verrazzano-mc", fmt.Sprintf("verrazzano-cluster-%s-agent", managedClusterName))
				if s == nil || err != nil {
					return false
				}
				s, err = pkg.GetSecret("verrazzano-mc", fmt.Sprintf("verrazzano-cluster-%s-registration", managedClusterName))
				if s == nil || err != nil {
					return false
				}
				return true
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("admin cluster has the expected filebeat logs", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound("vmo-local-filebeat-"+time.Now().Format("2006.01.02"),
					time.Now().Add(-24*time.Hour),
					map[string]string{
						"fields.verrazzano.cluster.name": managedClusterName})
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a filebeat log record")
		})

		ginkgo.It("admin cluster has the expected journal logs", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound("vmo-local-journalbeat-"+time.Now().Format("2006.01.02"),
					time.Now().Add(-24*time.Hour),
					map[string]string{
						"fields.verrazzano.cluster.name": managedClusterName})
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a journalbeat log record")
		})
	})

	ginkgo.Context("Managed Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_KUBECONFIG"))
		})

		ginkgo.It("managed cluster has the expected secrets", func() {
			gomega.Eventually(func() bool {
				s, err := pkg.GetSecret("verrazzano-system", "verrazzano-cluster-agent")
				if s == nil || err != nil {
					return false
				}
				s, err = pkg.GetSecret("verrazzano-system", "verrazzano-cluster-registration")
				if s == nil || err != nil {
					return false
				}
				return true
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})
})
