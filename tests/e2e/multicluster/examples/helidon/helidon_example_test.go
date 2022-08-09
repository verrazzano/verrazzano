// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mchelidon

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"

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
	skipVerifications    = "Skip Verifications"
	skipDeletions        = "Skip Deletions"
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false
var beforeSuitePassed = false
var t = framework.NewTestFramework("mchelidon")

var _ = t.AfterEach(func() {
	// set failed to true if any of the tests has failed
	failed = failed || CurrentSpecReport().Failed()
})

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = t.BeforeSuite(func() {
	if !skipDeploy {
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
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}
	beforeSuitePassed = true
})

var _ = t.Describe("In Multi-cluster, verify hello-helidon", Label("f:multicluster.mc-app-lcm"), func() {
	t.Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		t.It("Has multi cluster resources", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return examples.VerifyMCResources(adminKubeconfig, true, false, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has been created on admin cluster but not placed there
		// THEN expect that the app is not deployed to the admin cluster consistently for some length of time
		t.It("Does not have application placed", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
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
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return examples.VerifyMCResources(managedKubeconfig, false, true, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the multi-cluster example application has been created on admin cluster and placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		t.It("Has application placed", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				result, err := examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, true, testProjectName, testNamespace)
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
				if skipVerify {
					Skip(skipVerifications)
				}
				Eventually(func() bool {
					return examples.VerifyMCResources(kubeconfig, false, false, testNamespace)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
			t.It("Does not have application placed", func() {
				if skipVerify {
					Skip(skipVerifications)
				}
				Eventually(func() bool {
					result, err := examples.VerifyHelloHelidonInCluster(kubeconfig, false, false, testProjectName, testNamespace)
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
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.LogIndexFoundInCluster(indexName, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for hello helidon")
		})
	})

	// GIVEN an admin cluster and at least one managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect Prometheus metrics for the app to exist in Prometheus on the admin cluster
	t.Context("for Prometheus Metrics", Label("f:observability.monitoring.prom"), func() {

		t.It("Verify base_jvm_uptime_seconds metrics exist for managed cluster", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
			Eventually(func() bool {
				m := make(map[string]string)
				m["app"] = testApp
				m[clusterNameMetricsLabel] = clusterName
				return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find base_jvm_uptime_seconds metric")
		})

		t.It("Verify DNE base_jvm_uptime_seconds metrics does not exist for managed cluster", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
			Eventually(func() bool {
				m := make(map[string]string)
				m["cluster"] = testNamespace
				m[clusterNameMetricsLabel] = "DNE"
				return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeFalse(), "Not expected to find base_jvm_uptime_seconds metric")
		})

		t.It("Verify vendor_requests_count_total metrics exist for managed cluster", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
			Eventually(func() bool {
				m := make(map[string]string)
				m["app"] = testApp
				m[clusterNameMetricsLabel] = clusterName
				return pkg.MetricsExistInCluster("vendor_requests_count_total", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find vendor_requests_count_total metric")
		})

		t.It("Verify container_cpu_cfs_periods_total metrics exist for managed cluster", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
			Eventually(func() bool {
				m := make(map[string]string)
				m["namespace"] = testNamespace
				m[clusterNameMetricsLabel] = clusterName
				return pkg.MetricsExistInCluster("container_cpu_cfs_periods_total", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find container_cpu_cfs_periods_total metric")
		})
	})

	t.Context("Change Placement of app to Admin Cluster", func() {
		t.It("Apply patch to change placement to admin cluster", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() error {
				return examples.ChangePlacementToAdminCluster(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("MC Resources should be removed from managed cluster", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				// app should not be placed in the managed cluster
				return examples.VerifyMCResources(managedKubeconfig, false, false, testNamespace)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("App should be removed from managed cluster", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				// app should not be placed in the managed cluster
				result, err := examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, false, testProjectName, testNamespace)
				if err != nil {
					AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
				}
				return result
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("App should be placed in admin cluster", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				// app should be placed in the admin cluster
				result, err := examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, true, testProjectName, testProjectName)
				if err != nil {
					AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
				}
				return result
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	// Ensure that if we change placement again, back to the original managed cluster, everything functions
	// as expected. This is needed because the change of placement to admin cluster and the change of placement to
	// a managed cluster are different, and we want to ensure we test the case where the destination cluster is
	// each of the 2 types - admin and managed
	t.Context("Return the app to Managed Cluster", func() {
		t.It("Apply patch to change placement back to managed cluster", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() error {
				return examples.ChangePlacementToManagedCluster(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has changed placement from admin back to managed cluster
		// THEN expect that the app is not deployed to the admin cluster
		t.It("Admin cluster does not have application placed", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				result, err := examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false, testProjectName, testNamespace)
				if err != nil {
					AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
				}
				return result
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN a managed cluster
		// WHEN the multi-cluster example application has changed placement to this managed cluster
		// THEN expect that the app is now deployed to the cluster
		t.It("Managed cluster again has application placed", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
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
			if skipUndeploy {
				Skip(skipDeletions)
			}
			Eventually(func() error {
				return cleanUp(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Verify deletion on admin cluster", func() {
			if skipUndeploy {
				Skip(skipDeletions)
			}
			Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, false, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Verify automatic deletion on managed cluster", func() {
			if skipUndeploy {
				Skip(skipDeletions)
			}
			Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedInManagedCluster(managedKubeconfig, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Delete test namespace on managed cluster", func() {
			if skipUndeploy {
				Skip(skipDeletions)
			}
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(examples.TestNamespace, managedKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Delete test namespace on admin cluster", func() {
			if skipUndeploy {
				Skip(skipDeletions)
			}
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(examples.TestNamespace, adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})
	})
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		pkg.ExecuteBugReport(testNamespace)
	}
})

func cleanUp(kubeconfigPath string) error {
	start := time.Now()
	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/mc-hello-helidon-app.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster hello-helidon application resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/hello-helidon-comp.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster hello-helidon component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete hello-helidon project resource: %v", err)
	}
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	return nil
}
