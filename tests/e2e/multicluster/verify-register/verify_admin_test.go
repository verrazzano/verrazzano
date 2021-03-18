// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package register_test

import (
	"fmt"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"os"
	"time"

	"github.com/onsi/ginkgo"
)

const waitTimeout = 10 * time.Minute
const pollingInterval = 10 * time.Second

var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

var _ = ginkgo.Describe("Multi Cluster Verify Register",
	func() {
		ginkgo.It("admin cluster has the expected secrets", func() {
			gomega.Eventually(func() bool {
				s, err := pkg.GetSecret("verrazzano-mc", fmt.Sprintf("verrazzano-cluster-%s-manifest", managedClusterName))
				return s != nil && err == nil
				s, err = pkg.GetSecret("verrazzano-mc", fmt.Sprintf("verrazzano-cluster-%s-agent", managedClusterName))
				return s != nil && err == nil
				s, err = pkg.GetSecret("verrazzano-mc", fmt.Sprintf("verrazzano-cluster-%s-registration", managedClusterName))
				return s != nil && err == nil
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("admin cluster has the expected metrics", func() {
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
