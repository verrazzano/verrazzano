// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package argocd_test

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster/examples"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	dump "github.com/verrazzano/verrazzano/tests/e2e/pkg/test/clusterdump"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"
)

const (
	waitTimeout     = 5 * time.Minute
	pollingInterval = 10 * time.Second
	testNamespace   = "hello-helidon"
	testProjectName = "hello-helidon"
	argoCDNamespace = "argocd"
)

const (
	argoCdHelidonApplicationFile = "tests/e2e/config/scripts/hello-helidon-argocd-mc.yaml"
)

var managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

var t = framework.NewTestFramework("argocd_test")

var beforeSuite = t.BeforeSuiteFunc(func() {
	// Get the Hello Helidon Argo CD application yaml file
	// Deploy the Argo CD application in the admin cluster
	// This should internally deploy the helidon app to the managed cluster
	start := time.Now()
	Eventually(func() error {
		file, err := pkg.FindTestDataFile(argoCdHelidonApplicationFile)
		if err != nil {
			return err
		}
		return resource.CreateOrUpdateResourceFromFileInGeneratedNamespace(file, argoCDNamespace)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred(), "Failed to create Argo CD Application Project file")
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))

	beforeSuitePassed = true
})

var _ = BeforeSuite(beforeSuite)

var failed = false
var beforeSuitePassed = false

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.Describe("Multi Cluster Argo CD Validation", Label("f:platform-lcm.install"), func() {
	t.Context("Admin Cluster", func() {
		t.BeforeEach(func() {
			os.Setenv(k8sutil.EnvVarTestKubeConfig, os.Getenv("ADMIN_KUBECONFIG"))
		})

		t.It("has the expected secrets", func() {
			secretName := fmt.Sprintf("%s-argocd-cluster-secret", managedClusterName)
			Eventually(func() bool {
				return findSecret(argoCDNamespace, secretName)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find secret "+secretName)
		})
	})

	t.Context("Managed Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the managed cluster
		t.It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return examples.VerifyMCResources(managedKubeconfig, false, true, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the multi-cluster example application has been created on admin cluster and placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		t.It("Has application placed", func() {
			Eventually(func() bool {
				result, err := examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, true, testProjectName, testNamespace)
				if err != nil {
					AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
				}
				return result
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("Delete resources", func() {
		t.It("Delete resources on admin cluster", func() {
			Eventually(func() error {
				return deleteArgoCDApplication(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Verify deletion on admin cluster", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, false, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Verify automatic deletion on managed cluster", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedInManagedCluster(managedKubeconfig, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Delete test namespace on managed cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(examples.TestNamespace, managedKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Delete test namespace on admin cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(examples.TestNamespace, adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})
	})

})

var afterSuite = t.AfterSuiteFunc(func() {
	if failed || !beforeSuitePassed {
		dump.ExecuteBugReport(testNamespace)
	}
})

var _ = AfterSuite(afterSuite)

func deleteArgoCDApplication(kubeconfigPath string) error {
	start := time.Now()
	file, err := pkg.FindTestDataFile(argoCdHelidonApplicationFile)
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInCluster(file, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete Argo CD hello-helidon application: %v", err)
	}

	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	return nil
}

func findSecret(namespace, name string) bool {
	s, err := pkg.GetSecret(namespace, name)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Failed to get secret %s in namespace %s with error: %v", name, namespace, err))
		return false
	}
	return s != nil
}
