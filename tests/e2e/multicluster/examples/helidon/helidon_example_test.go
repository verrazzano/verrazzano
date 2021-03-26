// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mchelidon

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	clustersv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/clusters/v1alpha1"
	oamv1alpha1 "github.com/verrazzano/verrazzano/application-operator/apis/oam/v1alpha1"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

const (
	pollingInterval = 5 * time.Second
	waitTimeout     = 5 * time.Minute

	multiclusterNamespace = "verrazzano-mc"
	testNamespace         = "hello-helidon"

	projectName   = "hello-helidon"
	appConfigName = "hello-helidon-appconf"
	componentName = "hello-helidon-component"
	workloadName  = "hello-helidon-workload"
)

var expectedPodsHelloHelidon = []string{"hello-helidon-deployment"}
var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")

// set the kubeconfig to use the admin cluster kubeconfig and deploy the example resources
var _ = ginkgo.BeforeSuite(func() {
	os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))

	if err := pkg.CreateOrUpdateResourceFromFile("examples/multicluster/hello-helidon/verrazzano-project.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create hello-helidon project resource: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("examples/multicluster/hello-helidon/mc-hello-helidon-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create multi-cluster hello-helidon component resources: %v", err))
	}
	if err := pkg.CreateOrUpdateResourceFromFile("examples/multicluster/hello-helidon/mc-hello-helidon-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to create multi-cluster hello-helidon application resource: %v", err))
	}
})

var _ = ginkgo.Describe("Multi-cluster Verify Hello Helidon", func() {
	ginkgo.Context("Admin Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		ginkgo.It("Verify the project exists", func() {
			gomega.Eventually(projectExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify the MultiClusterApplicationConfiguration exists", func() {
			gomega.Eventually(mcAppConfExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify the MultiClusterComponent exists", func() {
			gomega.Eventually(mcComponentExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify the VerrazzanoHelidonWorkload does NOT exist", func() {
			gomega.Expect(componentWorkloadExists()).Should(gomega.BeFalse())
		})
	})

	ginkgo.Context("Managed Cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_KUBECONFIG"))
		})

		ginkgo.It("Verify the project exists", func() {
			gomega.Eventually(projectExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify the MultiClusterApplicationConfiguration exists", func() {
			gomega.Eventually(mcAppConfExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify the MultiClusterComponent exists", func() {
			gomega.Eventually(mcComponentExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify the VerrazzanoHelidonWorkload exists", func() {
			gomega.Eventually(componentWorkloadExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
		})

		ginkgo.It("Verify expected pods are running", func() {
			gomega.Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(gomega.BeTrue())
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

			ginkgo.It("Verify the project does not exist on this managed cluster", func() {
				os.Setenv("TEST_KUBECONFIG", kubeconfig)
				pkg.Log(pkg.Info, "Testing against cluster with kubeconfig: "+kubeconfig)
				gomega.Expect(projectExists()).Should(gomega.BeFalse())
			})

			ginkgo.It("Verify the MultiClusterApplicationConfiguration does not exist on this managed cluster", func() {
				os.Setenv("TEST_KUBECONFIG", kubeconfig)
				pkg.Log(pkg.Info, "Testing against cluster with kubeconfig: "+kubeconfig)
				gomega.Expect(mcAppConfExists()).Should(gomega.BeFalse())
			})

			ginkgo.It("Verify the MultiClusterComponent does not exist on this managed cluster", func() {
				os.Setenv("TEST_KUBECONFIG", kubeconfig)
				pkg.Log(pkg.Info, "Testing against cluster with kubeconfig: "+kubeconfig)
				gomega.Expect(mcComponentExists()).Should(gomega.BeFalse())
			})
		}
	})

	ginkgo.Context("Logging", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		indexName := "hello-helidon-hello-helidon-appconf-hello-helidon-component-hello-helidon-container"

		ginkgo.It("Verify Elasticsearch index exists on admin cluster", func() {
			gomega.Eventually(func() bool {
				return pkg.LogIndexFound(indexName)
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find log index for hello helidon")
		})

		ginkgo.It("Verify recent Elasticsearch log record exists on admin cluster", func() {
			gomega.Eventually(func() bool {
				return pkg.LogRecordFound(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					"oam.applicationconfiguration.namespace": "hello-helidon",
					"oam.applicationconfiguration.name":      "hello-helidon-appconf",
					"verrazzano.cluster.name":                clusterName,
				})
			}, waitTimeout, pollingInterval).Should(gomega.BeTrue(), "Expected to find a recent log record")
		})
	})

	// NOTE: This test is disabled until this bug is fixed: https://jira.oraclecorp.com/jira/browse/VZ-2448

	// ginkgo.Context("Metrics", func() {
	// 	ginkgo.BeforeEach(func() {
	// 		os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
	// 	})

	// 	ginkgo.It("Verify Prometheus metrics exist on admin cluster", func() {
	// 		gomega.Eventually(appMetricsExists, waitTimeout, pollingInterval).Should(gomega.BeTrue())
	// 	})
	// })

	ginkgo.Context("Delete resources on admin cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))
		})

		ginkgo.It("Delete all the things", func() {
			cleanUp()
		})

		ginkgo.It("Verify the project no longer exists on the admin cluster", func() {
			gomega.Eventually(projectExists, waitTimeout, pollingInterval).Should(gomega.BeFalse())
		})

		ginkgo.It("Verify the MultiClusterApplicationConfiguration no longer exists on the admin cluster", func() {
			gomega.Eventually(mcAppConfExists, waitTimeout, pollingInterval).Should(gomega.BeFalse())
		})

		ginkgo.It("Verify the MultiClusterComponent no longer exists on the admin cluster", func() {
			gomega.Eventually(mcComponentExists, waitTimeout, pollingInterval).Should(gomega.BeFalse())
		})
	})

	ginkgo.Context("Verify resources have been deleted on the managed cluster", func() {
		ginkgo.BeforeEach(func() {
			os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_KUBECONFIG"))
		})

		ginkgo.It("Verify the project no longer exists on the managed cluster", func() {
			gomega.Eventually(projectExists, waitTimeout, pollingInterval).Should(gomega.BeFalse())
		})

		// NOTE: These tests are disabled pending a fix for https://jira.oraclecorp.com/jira/browse/VZ-2454

		// ginkgo.It("Verify the MultiClusterApplicationConfiguration no longer exists on the managed cluster", func() {
		// 	gomega.Eventually(mcAppConfExists, waitTimeout, pollingInterval).Should(gomega.BeFalse())
		// })

		// ginkgo.It("Verify the MultiClusterComponent no longer exists on the managed cluster", func() {
		// 	gomega.Eventually(mcComponentExists, waitTimeout, pollingInterval).Should(gomega.BeFalse())
		// })

		// ginkgo.It("Verify the VerrazzanoHelidonWorkload no longer exists on the managed cluster", func() {
		// 	gomega.Eventually(componentWorkloadExists, waitTimeout, pollingInterval).Should(gomega.BeFalse())
		// })

		// ginkgo.It("Verify expected pods are no longer running", func() {
		// 	gomega.Eventually(helloHelidonPodsRunning, waitTimeout, pollingInterval).Should(gomega.BeFalse())
		// })
	})
})

var _ = ginkgo.AfterSuite(func() {
	cleanUp()

	os.Setenv("TEST_KUBECONFIG", os.Getenv("MANAGED_KUBECONFIG"))

	if err := pkg.DeleteNamespace(testNamespace); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete hello-helidon namespace: %v\n", err))
	}

	os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))

	if err := pkg.DeleteNamespace(testNamespace); err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not delete %s namespace: %v\n", testNamespace, err))
	}
})

func cleanUp() {
	os.Setenv("TEST_KUBECONFIG", os.Getenv("ADMIN_KUBECONFIG"))

	if err := pkg.DeleteResourceFromFile("examples/multicluster/hello-helidon/mc-hello-helidon-app.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete multi-cluster hello-helidon application resource: %v", err))
	}

	if err := pkg.DeleteResourceFromFile("examples/multicluster/hello-helidon/mc-hello-helidon-comp.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete multi-cluster hello-helidon component resources: %v", err))
	}

	if err := pkg.DeleteResourceFromFile("examples/multicluster/hello-helidon/verrazzano-project.yaml"); err != nil {
		ginkgo.Fail(fmt.Sprintf("Failed to delete hello-helidon project resource: %v", err))
	}
}

func resourceExists(gvr schema.GroupVersionResource, ns string, name string) bool {
	config := pkg.GetKubeConfig()
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		ginkgo.Fail(fmt.Sprintf("Could not create dynamic client: %v\n", err))
	}

	u, err := client.Resource(gvr).Namespace(ns).Get(context.TODO(), name, v1.GetOptions{})

	return u != nil && err == nil
}

func projectExists() bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.GroupVersion.Group,
		Version:  clustersv1alpha1.GroupVersion.Version,
		Resource: "verrazzanoprojects",
	}
	return resourceExists(gvr, multiclusterNamespace, projectName)
}

func mcAppConfExists() bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.GroupVersion.Group,
		Version:  clustersv1alpha1.GroupVersion.Version,
		Resource: "multiclusterapplicationconfigurations",
	}
	return resourceExists(gvr, testNamespace, appConfigName)
}

func mcComponentExists() bool {
	gvr := schema.GroupVersionResource{
		Group:    clustersv1alpha1.GroupVersion.Group,
		Version:  clustersv1alpha1.GroupVersion.Version,
		Resource: "multiclustercomponents",
	}
	return resourceExists(gvr, testNamespace, componentName)
}

func componentWorkloadExists() bool {
	gvr := schema.GroupVersionResource{
		Group:    oamv1alpha1.GroupVersion.Group,
		Version:  oamv1alpha1.GroupVersion.Version,
		Resource: "verrazzanohelidonworkloads",
	}
	return resourceExists(gvr, testNamespace, workloadName)
}

func helloHelidonPodsRunning() bool {
	return pkg.PodsRunning(testNamespace, expectedPodsHelloHelidon)
}

func appMetricsExists() bool {
	return pkg.MetricsExist("base_jvm_uptime_seconds", "managed_cluster", clusterName)
}
