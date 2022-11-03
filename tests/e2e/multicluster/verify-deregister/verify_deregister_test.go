// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package deregister_test

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"k8s.io/apimachinery/pkg/api/errors"
)

const waitTimeout = 10 * time.Minute
const pollingInterval = 10 * time.Second

const verrazzanoSystemNamespace = "verrazzano-system"

var externalEsURL = pkg.GetExternalOpenSearchURL(os.Getenv("ADMIN_KUBECONFIG"))

var t = framework.NewTestFramework("deregister_test")

var _ = t.AfterSuite(func() {})
var _ = t.BeforeSuite(func() {})
var _ = t.AfterEach(func() {})

var _ = t.Describe("Multi Cluster Verify Deregister", Label("f:multicluster.deregister"), func() {
	t.Context("Admin Cluster", func() {
		t.BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("ADMIN_KUBECONFIG"))
		})

		t.It("admin cluster Fluentd should point to the correct ES", func() {
			if pkg.UseExternalElasticsearch() {
				Eventually(func() bool {
					return pkg.AssertFluentdURLAndSecret(externalEsURL, "external-es-secret")
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected external ES in admin cluster fluentd Daemonset setting")
			} else {
				Eventually(func() bool {
					return pkg.AssertFluentdURLAndSecret(pkg.VmiESURL, pkg.VmiESInternalSecret)
				}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected VMI ES in admin cluster fluentd Daemonset setting")
			}
		})
	})

	t.Context("Managed Cluster", func() {
		t.BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("MANAGED_KUBECONFIG"))
		})

		t.It("should not have verrazzano-cluster-registration secret", func() {
			Eventually(func() bool {
				return missingSecret(verrazzanoSystemNamespace, "verrazzano-cluster-registration")
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected secret verrazzano-cluster-registration gone in managed cluster")
		})

		t.It("Fluentd should point to the correct ES", func() {
			Eventually(func() bool {
				return pkg.AssertFluentdURLAndSecret(pkg.VmiESURL, pkg.VmiESInternalSecret)
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
		pkg.Log(pkg.Info, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", name, namespace, err))
	}
	return false
}
