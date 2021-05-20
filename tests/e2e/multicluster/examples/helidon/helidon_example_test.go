// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mchelidon

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster/examples"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longPollingInterval  = 20 * time.Second
	longWaitTimeout      = 10 * time.Minute
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	consistentlyDuration = 1 * time.Minute
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false

var _ = ginkgo.AfterEach(func() {
	// set failed to true if any of the tests has failed
	failed = failed || ginkgo.CurrentGinkgoTestDescription().Failed
})

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = ginkgo.BeforeSuite(func() {
	// deploy the VerrazzanoProject
	err := examples.DeployHelloHelidonProject(adminKubeconfig)
	if err != nil {
		ginkgo.Fail(err.Error())
	}
	// wait for the namespace to be created on the cluster before deploying app
	gomega.Eventually(func() bool {
		return examples.HelidonNamespaceExists(adminKubeconfig)
	}, waitTimeout, pollingInterval).Should(gomega.BeTrue())

	err = examples.DeployHelloHelidonApp(adminKubeconfig)
	if err != nil {
		ginkgo.Fail(err.Error())
	}
})

var _ = ginkgo.Describe("Multi-cluster verify hello-helidon", func() {
	ginkgo.Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		ginkgo.It("Has multi cluster resources", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyMCResources(adminKubeconfig, true, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has been created on admin cluster but not placed there
		// THEN expect that the app is not deployed to the admin cluster consistently for some length of time
		ginkgo.It("Does not have application placed", func() {
			gomega.Consistently(func() bool {
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false)
			}, consistentlyDuration, pollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Managed Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the managed cluster
		ginkgo.It("Has multi cluster resources", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyMCResources(managedKubeconfig, false, true)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the multi-cluster example application has been created on admin cluster and placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		ginkgo.It("Has application placed", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, true)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Remaining Managed Clusters", func() {
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
			ginkgo.It("Does not have multi cluster resources", func() {
				gomega.Eventually(func() bool {
					return examples.VerifyMCResources(kubeconfig, false, false)
				}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
			})
			ginkgo.It("Does not have application placed", func() {
				gomega.Eventually(func() bool {
					return examples.VerifyHelloHelidonInCluster(kubeconfig, false, false)
				}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
			})
		}
	})

	ginkgo.Context("Logging", func() {
		indexName := "verrazzano-namespace-hello-helidon"

		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect the Elasticsearch index for the app exists on the admin cluster Elasticsearch
		ginkgo.It("Verify Elasticsearch index exists on admin cluster", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFoundInCluster(indexName, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find log index for hello helidon")
		})

		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect recent Elasticsearch logs for the app exist on the admin cluster Elasticsearch
		ginkgo.It("Verify recent Elasticsearch log record exists on admin cluster", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFoundInCluster(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"kubernetes.labels.app_oam_dev\\/component": "hello-helidon-component",
					"kubernetes.labels.app_oam_dev\\/name":      "hello-helidon-appconf",
					"kubernetes.container_name":                 "hello-helidon-container",
				}, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
		})
	})

	// GIVEN an admin cluster and at least one managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect Prometheus metrics for the app to exist in Prometheus on the admin cluster
	ginkgo.Context("Metrics", func() {
		ginkgo.It("Verify Prometheus metrics exist on admin cluster", func() {
			gomega.Eventually(func() bool {
				return appMetricsExists(adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Change Placement of app to Admin Cluster", func() {
		ginkgo.It("Apply patch to change placement to admin cluster", func() {
			err := examples.ChangePlacementToAdminCluster(adminKubeconfig)
			if err != nil {
				ginkgo.Fail(err.Error())
			}
		})

		ginkgo.It("MC Resources should be removed from managed cluster", func() {
			gomega.Eventually(func() bool {
				// app should not be placed in the managed cluster
				return examples.VerifyMCResources(managedKubeconfig, false, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("App should be removed from managed cluster", func() {
			gomega.Eventually(func() bool {
				// app should not be placed in the managed cluster
				return examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("App should be placed in admin cluster", func() {
			gomega.Eventually(func() bool {
				// app should be placed in the admin cluster
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, true)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	// Ensure that if we change placement again, back to the original managed cluster, everything functions
	// as expected. This is needed because the change of placement to admin cluster and the change of placement to
	// a managed cluster are different, and we want to ensure we test the case where the destination cluster is
	// each of the 2 types - admin and managed
	ginkgo.Context("Return the app to Managed Cluster", func() {
		ginkgo.It("Apply patch to change placement back to managed cluster", func() {
			err := examples.ChangePlacementToManagedCluster(adminKubeconfig)
			if err != nil {
				ginkgo.Fail(err.Error())
			}
		})

		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has changed placement from admin back to managed cluster
		// THEN expect that the app is not deployed to the admin cluster
		ginkgo.It("Admin cluster does not have application placed", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(adminKubeconfig, true, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		// GIVEN a managed cluster
		// WHEN the multi-cluster example application has changed placement to this managed cluster
		// THEN expect that the app is now deployed to the cluster
		ginkgo.It("Managed cluster again has application placed", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonInCluster(managedKubeconfig, false, true)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})

	ginkgo.Context("Delete resources", func() {
		ginkgo.It("Delete resources on admin cluster", func() {
			err := cleanUp(adminKubeconfig)
			if err != nil {
				ginkgo.Fail(err.Error())
			}
		})

		ginkgo.It("Verify deletion on admin cluster", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify automatic deletion on managed cluster", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedInManagedCluster(managedKubeconfig)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Delete test namespace on managed cluster", func() {
			if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, managedKubeconfig); err != nil {
				ginkgo.Fail(fmt.Sprintf("Could not delete %s namespace in managed cluster: %v\n", examples.TestNamespace, err))
			}
		})

		ginkgo.It("Delete test namespace on admin cluster", func() {
			if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, adminKubeconfig); err != nil {
				ginkgo.Fail(fmt.Sprintf("Could not delete %s namespace in admin cluster: %v\n", examples.TestNamespace, err))
			}
		})
	})
})

var _ = ginkgo.AfterSuite(func() {
	if failed {
		pkg.ExecuteClusterDumpWithEnvVarConfig()
	}
})

func cleanUp(kubeconfigPath string) error {
	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/mc-hello-helidon-app.yaml", kubeconfigPath); err != nil {
		return fmt.Errorf("Failed to delete multi-cluster hello-helidon application resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/mc-hello-helidon-comp.yaml", kubeconfigPath); err != nil {
		return fmt.Errorf("Failed to delete multi-cluster hello-helidon component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/verrazzano-project.yaml", kubeconfigPath); err != nil {
		return fmt.Errorf("Failed to delete hello-helidon project resource: %v", err)
	}
	return nil
}

func appMetricsExists(kubeconfigPath string) bool {
	return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", "managed_cluster", clusterName, kubeconfigPath)
}
