// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package sock_shop

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8s/resource"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"

	. "github.com/onsi/ginkgo/v2"
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
	testCluster          = "SockShop"
	testApp              = "carts-coh"
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false
var beforeSuitePassed = false

var t = framework.NewTestFramework("sock_shop")

var _ = t.AfterEach(func() {
	// set failed to true if any of the tests has failed
	failed = failed || CurrentSpecReport().Failed()
})

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = t.BeforeSuite(func() {
	// deploy the VerrazzanoProject
	start := time.Now()
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
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("In Multi-cluster, verify sock-shop", Label("f:multicluster.mc-app-lcm"), func() {
	t.Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		t.It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return VerifyMCResources(adminKubeconfig, true, false, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has been created on admin cluster but not placed there
		// THEN expect that the app is not deployed to the admin cluster consistently for some length of time
		t.It("Does not have application placed", func() {
			Consistently(func() bool {
				result, err := VerifySockShopInCluster(adminKubeconfig, true, false, testProjectName, testNamespace)
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
				return VerifyMCResources(managedKubeconfig, false, true, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the multi-cluster example application has been created on admin cluster and placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		t.It("Has application placed", func() {
			Eventually(func() bool {
				result, err := VerifySockShopInCluster(managedKubeconfig, false, true, testProjectName, testNamespace)
				if err != nil {
					AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
				}
				return result
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	t.Context("Remaining Managed Clusters", func() {
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
			t.It("Does not have multi cluster resources", func() {
				Eventually(func() bool {
					return VerifyMCResources(kubeconfig, false, false, testNamespace)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
			t.It("Does not have application placed", func() {
				Eventually(func() bool {
					result, err := VerifySockShopInCluster(kubeconfig, false, false, testProjectName, testNamespace)
					if err != nil {
						AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
					}
					return result
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		}
	})

	t.Context("for Logging", Label("f:observability.logging.es"), func() {
		indexName, err := pkg.GetOpenSearchAppIndexWithKC(testNamespace, adminKubeconfig)
		Expect(err).To(BeNil())
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect the Elasticsearch index for the app exists on the admin cluster Elasticsearch
		t.It("Verify Elasticsearch index exists on admin cluster", func() {
			Eventually(func() bool {
				return pkg.LogIndexFoundInCluster(indexName, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for sock-shop")
		})
	})

	// GIVEN an admin cluster and at least one managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect Prometheus metrics for the app to exist in Prometheus on the admin cluster
	t.Context("for Prometheus Metrics", Label("f:observability.monitoring.prom"), func() {

		// Coherence metric fix available only from 1.3.0
		if ok, _ := pkg.IsVerrazzanoMinVersion("1.3.0", adminKubeconfig); ok {
			t.It("Verify base_jvm_uptime_seconds metrics exist for managed cluster", func() {
				clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
				Eventually(func() bool {
					m := make(map[string]string)
					m["app"] = testApp
					m[clusterNameMetricsLabel] = clusterName
					return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", m, adminKubeconfig)
				}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find base_jvm_uptime_seconds metric")
			})

			t.It("Verify DNE base_jvm_uptime_seconds metrics does not exist for managed cluster", func() {
				clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
				Eventually(func() bool {
					m := make(map[string]string)
					m["app"] = testApp
					m[clusterNameMetricsLabel] = "DNE"
					return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", m, adminKubeconfig)
				}, longWaitTimeout, longPollingInterval).Should(BeFalse(), "Not expected to find base_jvm_uptime_seconds metric")
			})

			t.It("Verify vendor_requests_count_total metrics exist for managed cluster", func() {
				clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
				Eventually(func() bool {
					m := make(map[string]string)
					m["app"] = testApp
					m[clusterNameMetricsLabel] = clusterName
					return pkg.MetricsExistInCluster("vendor_requests_count_total", m, adminKubeconfig)
				}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find vendor_requests_count_total metric")
			})

			t.It("Verify container_cpu_cfs_periods_total metrics exist for managed cluster", func() {
				clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
				Eventually(func() bool {
					m := make(map[string]string)
					m["namespace"] = testNamespace
					m[clusterNameMetricsLabel] = clusterName
					return pkg.MetricsExistInCluster("container_cpu_cfs_periods_total", m, adminKubeconfig)
				}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find container_cpu_cfs_periods_total metric")
			})

			t.It("Verify coherence metrics exist for managed cluster", func() {
				clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
				Eventually(func() bool {
					m := make(map[string]string)
					m["cluster"] = testCluster
					m[clusterNameMetricsLabel] = clusterName
					return pkg.MetricsExistInCluster("vendor:coherence_service_messages_local", m, adminKubeconfig)
				}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find coherence metric")
			})
		}
	})

	t.Context("Delete resources", func() {
		t.It("on admin cluster", func() {
			Eventually(func() error {
				return cleanUp(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Verify deletion on admin cluster", func() {
			Eventually(func() bool {
				return VerifySockShopDeleteOnAdminCluster(adminKubeconfig, false, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Verify automatic deletion on managed cluster", func() {
			Eventually(func() bool {
				return VerifySockShopDeleteOnManagedCluster(managedKubeconfig, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Delete test namespace on managed cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(testNamespace, managedKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Delete test namespace on admin cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(testNamespace, adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})
	})
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		err := pkg.ExecuteBugReport(testNamespace)
		if err != nil {
			return
		}
	}
})

func cleanUp(kubeconfigPath string) error {
	start := time.Now()
	file, err := pkg.FindTestDataFile(fmt.Sprintf("examples/multicluster/%s/sock-shop-app.yaml", sourceDir))
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInCluster(file, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop application resource: %v", err)
	}

	file, err = pkg.FindTestDataFile(fmt.Sprintf("examples/multicluster/%s/sock-shop-comp.yaml", sourceDir))
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInCluster(file, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop component resources: %v", err)
	}

	file, err = pkg.FindTestDataFile(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir))
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInCluster(file, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete sock-shop project resource: %v", err)
	}
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	return nil
}
