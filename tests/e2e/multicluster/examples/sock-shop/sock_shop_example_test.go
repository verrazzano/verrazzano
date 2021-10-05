// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package sock_shop

import (
	"fmt"
	"os"
	"strconv"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longPollingInterval  = 20 * time.Second
	longWaitTimeout      = 10 * time.Minute
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	consistentlyDuration = 1 * time.Minute
	sourceDir            = "sock-shop"
	testNamespace        = "mc-sockshop"
	testProjectName      = "sockshop"
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false

var _ = AfterEach(func() {
	// set failed to true if any of the tests has failed
	failed = failed || CurrentGinkgoTestDescription().Failed
})

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = BeforeSuite(func() {
	// deploy the VerrazzanoProject
	Eventually(func() error {
		return DeploySockShopProject(adminKubeconfig, sourceDir)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// wait for the namespace to be created on the cluster before deploying app
	Eventually(func() bool {
		return SockShopNamespaceExists(adminKubeconfig, testNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue())

	Eventually(func() error {
		return DeploySockShopApp(adminKubeconfig, sourceDir)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
})

var _ = Describe("Multi-cluster verify sock-shop", func() {
	Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return VerifyMCResources(adminKubeconfig, true, false, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has been created on admin cluster but not placed there
		// THEN expect that the app is not deployed to the admin cluster consistently for some length of time
		It("Does not have application placed", func() {
			Consistently(func() bool {
				return VerifySockShopInCluster(adminKubeconfig, true, false, testProjectName, testNamespace)
			}, consistentlyDuration, pollingInterval).Should(BeTrue())
		})
	})

	Context("Managed Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the managed cluster
		It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return VerifyMCResources(managedKubeconfig, false, true, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the multi-cluster example application has been created on admin cluster and placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		It("Has application placed", func() {
			Eventually(func() bool {
				return VerifySockShopInCluster(managedKubeconfig, false, true, testProjectName, testNamespace)
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
					return VerifyMCResources(kubeconfig, false, false, testNamespace)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
			It("Does not have application placed", func() {
				Eventually(func() bool {
					return VerifySockShopInCluster(kubeconfig, false, false, testProjectName, testNamespace)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		}
	})

	//Context("Logging", func() {
	//	indexName := "verrazzano-namespace-sock-shop"
	//
	//	// GIVEN an admin cluster and at least one managed cluster
	//	// WHEN the example application has been deployed to the admin cluster
	//	// THEN expect the Elasticsearch index for the app exists on the admin cluster Elasticsearch
	//	It("Verify Elasticsearch index exists on admin cluster", func() {
	//		Eventually(func() bool {
	//			return pkg.LogIndexFoundInCluster(indexName, adminKubeconfig)
	//		}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for sock shop")
	//	})
	//
	//	// GIVEN an admin cluster and at least one managed cluster
	//	// WHEN the example application has been deployed to the admin cluster
	//	// THEN expect recent Elasticsearch logs for the app exist on the admin cluster Elasticsearch
	//	It("Verify recent Elasticsearch log record exists on admin cluster", func() {
	//		Eventually(func() bool {
	//			return pkg.LogRecordFoundInCluster(indexName, time.Now().Add(-24*time.Hour), map[string]string{
	//				"kubernetes.labels.app_oam_dev\\/component": "sock-shop-component",
	//				"kubernetes.labels.app_oam_dev\\/name":      "sock-shop-appconf",
	//				"kubernetes.container_name":                 "sock-shop-container",
	//			}, adminKubeconfig)
	//		}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
	//	})
	//})

	// GIVEN an admin cluster and at least one managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect Prometheus metrics for the app to exist in Prometheus on the admin cluster
	Context("Metrics", func() {
		It("Verify Prometheus metrics exist on admin cluster", func() {
			Eventually(func() bool {
				return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", "managed_cluster", clusterName, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})

	Context("Delete resources", func() {
		It("Delete resources on admin cluster", func() {
			Eventually(func() error {
				return cleanUp(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		//It("Verify deletion on admin cluster", func() {
		//	Eventually(func() bool {
		//		return VerifySockShopDeleteOnAdminCluster(adminKubeconfig, false, testNamespace, testProjectName)
		//	}, waitTimeout, pollingInterval).Should(BeTrue())
		//})

		It("Verify automatic deletion on managed cluster", func() {
			Eventually(func() bool {
				return VerifySockShopDeleteOnManagedCluster(managedKubeconfig, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("Delete test namespace on managed cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(testNamespace, managedKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		It("Delete test namespace on admin cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(testNamespace, adminKubeconfig)
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
	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/sock-shop-app.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop application resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/sock-shop-comp.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete sock-shop project resource: %v", err)
	}
	return nil
}
