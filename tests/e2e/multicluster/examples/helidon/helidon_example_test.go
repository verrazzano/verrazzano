// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mchelidon

import (
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster/examples"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longPollingInterval  = 20 * time.Second
	longWaitTimeout      = 20 * time.Minute
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	consistentlyDuration = 1 * time.Minute
	sourceDir            = "hello-helidon"
	testNamespace        = "hello-helidon"
	testProjectName      = "hello-helidon"
	testApp              = "hello-helidon"
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false

var _ = AfterEach(func() {
	// set failed to true if any of the tests has failed
	failed = failed || CurrentSpecReport().Failed()
})

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = BeforeSuite(func() {
	// deploy the VerrazzanoProject
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
})

var _ = Describe("Multi-cluster verify hello-helidon", func() {
	Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return examples.VerifyMCResources(adminKubeconfig, true, false, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has been created on admin cluster but not placed there
		// THEN expect that the app is not deployed to the admin cluster consistently for some length of time
		It("Does not have application placed", func() {
			Consistently(func() bool {
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false, testProjectName, testNamespace)
			}, consistentlyDuration, pollingInterval).Should(BeTrue())
		})
	})

	Context("Managed Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the managed cluster
		It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return examples.VerifyMCResources(managedKubeconfig, false, true, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the multi-cluster example application has been created on admin cluster and placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		It("Has application placed", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, true, testProjectName, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	Context("Remaining Managed Clusters", func() {
		clusterCountStr := os.Getenv("CLUSTER_COUNT")
		if clusterCountStr == "" {
			// skip tests
			return
		}
		clusterCount, err := strconv.Atoi(clusterCountStr)
		if err != nil {
			// skip tests
			return
		}

		kubeconfigDir := os.Getenv("KUBECONFIG_DIR")
		for i := 3; i <= clusterCount; i++ {
			kubeconfig := kubeconfigDir + "/" + fmt.Sprintf("%d", i) + "/kube_config"
			It("Does not have multi cluster resources", func() {
				Eventually(func() bool {
					return examples.VerifyMCResources(kubeconfig, false, false, testNamespace)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
			It("Does not have application placed", func() {
				Eventually(func() bool {
					return examples.VerifyHelloHelidonInCluster(kubeconfig, false, false, testProjectName, testNamespace)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		}
	})

	Context("Logging", func() {
		indexName := "verrazzano-namespace-hello-helidon"

		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect the Elasticsearch index for the app exists on the admin cluster Elasticsearch
		It("Verify Elasticsearch index exists on admin cluster", func() {
			Eventually(func() bool {
				return pkg.LogIndexFoundInCluster(indexName, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for hello helidon")
		})

		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect recent Elasticsearch logs for the app exist on the admin cluster Elasticsearch
		It("Verify recent Elasticsearch log record exists on admin cluster", func() {
			Eventually(func() bool {
				return pkg.LogRecordFoundInCluster(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "hello-helidon-component",
					"kubernetes.labels.app_oam_dev\\/name":      "hello-helidon-appconf",
					"kubernetes.container_name":                 "hello-helidon-container",
				}, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
		})
	})

	// GIVEN an admin cluster and at least one managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect Prometheus metrics for the app to exist in Prometheus on the admin cluster
	Context("Prometheus Metrics", func() {

		It("Verify base_jvm_uptime_seconds metrics exist for managed cluster", func() {
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel()
			Eventually(func() bool {
				m := make(map[string]string)
				m["app"] = testApp
				m[clusterNameMetricsLabel] = clusterName
				return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find base_jvm_uptime_seconds metric")
		})

		It("Verify DNE base_jvm_uptime_seconds metrics does not exist for managed cluster", func() {
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel()
			Eventually(func() bool {
				m := make(map[string]string)
				m["cluster"] = testNamespace
				m[clusterNameMetricsLabel] = "DNE"
				return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeFalse(), "Not expected to find base_jvm_uptime_seconds metric")
		})

		It("Verify vendor_requests_count_total metrics exist for managed cluster", func() {
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel()
			Eventually(func() bool {
				m := make(map[string]string)
				m["app"] = testApp
				m[clusterNameMetricsLabel] = clusterName
				return pkg.MetricsExistInCluster("vendor_requests_count_total", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find vendor_requests_count_total metric")
		})

		It("Verify container_cpu_cfs_periods_total metrics exist for managed cluster", func() {
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel()
			Eventually(func() bool {
				m := make(map[string]string)
				m["namespace"] = testNamespace
				m[clusterNameMetricsLabel] = clusterName
				return pkg.MetricsExistInCluster("container_cpu_cfs_periods_total", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find container_cpu_cfs_periods_total metric")
		})
	})

	Context("Change Placement of app to Admin Cluster", func() {
		It("Apply patch to change placement to admin cluster", func() {
			Eventually(func() error {
				return examples.ChangePlacementToAdminCluster(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("MC Resources should be removed from managed cluster", func() {
			Eventually(func() bool {
				// app should not be placed in the managed cluster
				return examples.VerifyMCResources(managedKubeconfig, false, false, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("App should be removed from managed cluster", func() {
			Eventually(func() bool {
				// app should not be placed in the managed cluster
				return examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, false, testProjectName, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("App should be placed in admin cluster", func() {
			Eventually(func() bool {
				// app should be placed in the admin cluster
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, true, testProjectName, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	// Ensure that if we change placement again, back to the original managed cluster, everything functions
	// as expected. This is needed because the change of placement to admin cluster and the change of placement to
	// a managed cluster are different, and we want to ensure we test the case where the destination cluster is
	// each of the 2 types - admin and managed
	Context("Return the app to Managed Cluster", func() {
		It("Apply patch to change placement back to managed cluster", func() {
			Eventually(func() error {
				return examples.ChangePlacementToManagedCluster(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has changed placement from admin back to managed cluster
		// THEN expect that the app is not deployed to the admin cluster
		It("Admin cluster does not have application placed", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false, testProjectName, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN a managed cluster
		// WHEN the multi-cluster example application has changed placement to this managed cluster
		// THEN expect that the app is now deployed to the cluster
		It("Managed cluster again has application placed", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, true, testProjectName, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	Context("Delete resources", func() {
		It("Delete resources on admin cluster", func() {
			Eventually(func() error {
				return cleanUp(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("Verify deletion on admin cluster", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, false, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("Verify automatic deletion on managed cluster", func() {
			Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedInManagedCluster(managedKubeconfig, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("Delete test namespace on managed cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(examples.TestNamespace, managedKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("Delete test namespace on admin cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(examples.TestNamespace, adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})
	})
})

var _ = AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
})

func cleanUp(kubeconfigPath string) error {
	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/mc-hello-helidon-app.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster hello-helidon application resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/hello-helidon-comp.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster hello-helidon component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete hello-helidon project resource: %v", err)
	}
	return nil
}
