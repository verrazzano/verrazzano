// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package deregister_test

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"k8s.io/apimachinery/pkg/api/errors"
)

const waitTimeout = 10 * time.Minute
const pollingInterval = 10 * time.Second

const verrazzanoSystemNamespace = "verrazzano-system"

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")

var _ = Describe("Multi Cluster Verify Deregister", func() {
	Context("Managed Cluster", func() {
		BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("MANAGED_KUBECONFIG"))
		})

		It("managed cluster has the expected secrets", func() {
			Eventually(func() bool {
				return missingSecret(verrazzanoSystemNamespace, "verrazzano-cluster-registration")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find secret verrazzano-cluster-registration")
		})

		It("managed cluster Fluentd should point to the correct ES", func() {
			Eventually(func() bool {
				return pkg.AssertFluentdURLAndSecret(pkg.VmiESURL, pkg.VmiESSecret)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected VMI ES in managed cluster fluentd Daemonset setting")
		})
	})
})

func missingSecret(namespace, name string) bool {
	_, err := pkg.GetSecret(namespace, name)
	if err != nil {
		if errors.IsNotFound(err) {
			return true
		}
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", name, namespace, err))
	}
	return false
}
