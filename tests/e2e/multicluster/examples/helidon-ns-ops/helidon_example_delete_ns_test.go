// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcnshelidon

import (
	"fmt"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework/metrics"

	"os"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster/examples"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	consistentlyDuration = 1 * time.Minute
	sourceDir            = "hello-helidon-ns"
	testNamespace        = "hello-helidon-ns"
	testProjectName      = "hello-helidon-ns"
)

var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false
var beforeSuitePassed = false

var t = framework.NewTestFramework("mcnshelidon")

var _ = t.AfterEach(func() {
	// set failed to true if any of the tests has failed
	failed = failed || CurrentSpecReport().Failed()
})

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = t.BeforeSuite(func() {
	// deploy the VerrazzanoProject
	start := time.Now()
	Eventually(func() error {
		return examples.DeployHelloHelidonProject(adminKubeconfig, sourceDir)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// wait for the namespace to be created on the cluster before deploying app
	Eventually(func() bool {
		return examples.HelidonNamespaceExists(adminKubeconfig, sourceDir)
	}, waitTimeout, pollingInterval).Should(BeTrue())

	Eventually(func() error {
		return examples.DeployHelloHelidonApp(adminKubeconfig, sourceDir)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("In Multi-cluster, verify delete ns of hello-helidon-ns", Label("f:multicluster.mc-app-lcm"), func() {
	t.Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		t.It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return examples.VerifyMCResources(adminKubeconfig, true, false, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has been created on admin cluster but not placed there
		// THEN expect that the app is not deployed to the admin cluster consistently for some length of time
		t.It("Does not have application placed", func() {
			Consistently(func() bool {
				result, err := examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false, testProjectName, testNamespace)
				if err != nil {
					AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
				}
				return result
			}, consistentlyDuration, pollingInterval).Should(BeTrue())
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
		t.It("Delete project on admin cluster", func() {
			Eventually(func() error {
				return deleteProject(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Delete test namespace on managed cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(testNamespace, managedKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Verify deletion on managed cluster", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedInManagedCluster(managedKubeconfig, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Delete test namespace on admin cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(testNamespace, adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Verify deletion on admin cluster", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, false, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

	})
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteBugReport(testNamespace)
	}
})

func deleteProject(kubeconfigPath string) error {
	start := time.Now()
	file, err := pkg.FindTestDataFile("examples/multicluster/hello-helidon-ns/verrazzano-project.yaml")
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInCluster(file, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete hello-helidon project resource: %v", err)
	}
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	return nil
}
