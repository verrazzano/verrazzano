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
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	consistentlyDuration = 1 * time.Minute
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

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
		indexName := "hello-helidon-hello-helidon-appconf-hello-helidon-component-hello-helidon-container"

		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect the Elasticsearch index for the app exists on the admin cluster Elasticsearch
		ginkgo.It("Verify Elasticsearch index exists on admin cluster", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFoundInCluster(indexName, adminKubeconfig)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find log index for hello helidon")
		})

		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect recent Elasticsearch logs for the app exist on the admin cluster Elasticsearch
		ginkgo.It("Verify recent Elasticsearch log record exists on admin cluster", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFoundInCluster(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"oam.applicationconfiguration.namespace": "hello-helidon",
					"oam.applicationconfiguration.name":      "hello-helidon-appconf",
					"verrazzano.cluster.name":                clusterName,
				}, adminKubeconfig)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
		})
	})

	// NOTE: This test is disabled until this bug is fixed: VZ-2448

	// ginkgo.Context("Metrics", func() {

	// GIVEN an admin cluster and at least one managed cluster
	// WHEN the example application has been deployed to the admin cluster
	// THEN expect Prometheus metrics for the app to exist in Prometheus on the admin cluster
	// 	ginkgo.It("Verify Prometheus metrics exist on admin cluster", func() {
	// 		gomega.Eventually(func() bool {
	//			return appMetricsExists(adminKubeconfig)
	// 		}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
	// 	})
	// })

	ginkgo.Context("Delete resources on admin cluster", func() {
		ginkgo.It("Delete all the things", func() {
			err := cleanUp()
			if err != nil {
				ginkgo.Fail(err.Error())
			}
		})

		ginkgo.It("Verify deletion on admin cluster", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, false)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
		ginkgo.It("Verify deletion on managed cluster", func() {
			gomega.Eventually(func() bool {
				return examples.VerifyHelloHelidonDeletedInManagedCluster(managedKubeconfig)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})
	})
})

var _ = ginkgo.AfterSuite(func() {
	if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, managedKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete hello-helidon namespace: %v\n", err))
	}

	if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, adminKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete %s namespace: %v\n", examples.TestNamespace, err))
	}
})

func cleanUp() error {
	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/mc-hello-helidon-app.yaml", adminKubeconfig); err != nil {
		return fmt.Errorf("Failed to delete multi-cluster hello-helidon application resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/mc-hello-helidon-comp.yaml", adminKubeconfig); err != nil {
		return fmt.Errorf("Failed to delete multi-cluster hello-helidon component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/verrazzano-project.yaml", adminKubeconfig); err != nil {
		return fmt.Errorf("Failed to delete hello-helidon project resource: %v", err)
	}
	return nil
}

func appMetricsExists(kubeconfigPath string) bool {
	return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", "managed_cluster", clusterName, kubeconfigPath)
}
