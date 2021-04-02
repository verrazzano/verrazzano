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
	pollingInterval = 5 * time.Second
	waitTimeout     = 5 * time.Minute
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = ginkgo.BeforeSuite(func() {
	examples.DeployHelloHelidon(adminKubeconfig)
})

var _ = ginkgo.Describe("Multi-cluster Verify Hello Helidon", func() {
	ginkgo.Context("Admin Cluster", func() {
		examples.VerifyHelloHelidonInAdminCluster(adminKubeconfig, false)
	})

	ginkgo.Context("Managed Cluster", func() {
		examples.VerifyHelloHelidonInManagedCluster(managedKubeconfig, true)
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

			examples.VerifyHelloHelidonInManagedCluster(kubeconfig, false)
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
			cleanUp()
		})

		examples.VerifyHelloHelidonDeletedAdminCluster(adminKubeconfig, false)
	})

	ginkgo.Context("Verify resources have been deleted on the managed cluster", func() {
		examples.VerifyHelloHelidonDeletedInCluster(managedKubeconfig, true)
	})
})

var _ = ginkgo.AfterSuite(func() {
	cleanUp()

	if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, managedKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete hello-helidon namespace: %v\n", err))
	}

	if err := pkg.DeleteNamespaceInCluster(examples.TestNamespace, adminKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete %s namespace: %v\n", examples.TestNamespace, err))
	}
})

func cleanUp() {
	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/mc-hello-helidon-app.yaml", adminKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete multi-cluster hello-helidon application resource: %v", err))
	}

	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/mc-hello-helidon-comp.yaml", adminKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete multi-cluster hello-helidon component resources: %v", err))
	}

	if err := pkg.DeleteResourceFromFileInCluster("examples/multicluster/hello-helidon/verrazzano-project.yaml", adminKubeconfig); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete hello-helidon project resource: %v", err))
	}
}

func appMetricsExists(kubeconfigPath string) bool {
	return pkg.MetricsExistInCluster("base_jvm_uptime_seconds", "managed_cluster", clusterName, kubeconfigPath)
}
