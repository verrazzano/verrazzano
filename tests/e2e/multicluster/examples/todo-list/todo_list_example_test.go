// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo_list

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	m1 "k8s.io/api/core/v1"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg/weblogic"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	longWaitTimeout      = 10 * time.Minute
	longPollingInterval  = 20 * time.Second
	consistentlyDuration = 1 * time.Minute
	sourceDir            = "todo-list"
	testNamespace        = "mc-todo-list"
	testProjectName      = "todo-list"
)

var clusterName = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")

// failed indicates whether any of the tests has failed
var failed = false

var _ = AfterEach(func() {
	failed = failed || CurrentGinkgoTestDescription().Failed
})

var _ = BeforeSuite(func() {
	// deploy the VerrazzanoProject
	Eventually(func() error {
		return DeployTodoListProject(adminKubeconfig, sourceDir)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// wait for the namespace to be created on the cluster before deploying app
	Eventually(func() bool {
		return TodoListNamespaceExists(adminKubeconfig, testNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue())

	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := pkg.GetRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	// create Docker repository secret
	Eventually(func() (*m1.Secret, error) {
		return pkg.CreateDockerSecret(testNamespace, "tododomain-repo-credentials", regServ, regUser, regPass)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	// create Weblogic credentials secret
	Eventually(func() (*m1.Secret, error) {
		return pkg.CreateCredentialsSecret(testNamespace, "tododomain-weblogic-credentials", wlsUser, wlsPass, nil)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	// create database credentials secret
	Eventually(func() (*m1.Secret, error) {
		return pkg.CreateCredentialsSecret(testNamespace, "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "tododomain"})
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	Eventually(func() error {
		return DeployTodoListApp(adminKubeconfig, sourceDir, testNamespace)
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
				return VerifyTodoListInCluster(adminKubeconfig, true, false, testProjectName, testNamespace)
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
				return VerifyTodoListInCluster(managedKubeconfig, false, true, testProjectName, testNamespace)
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
					return VerifyTodoListInCluster(kubeconfig, false, false, testProjectName, testNamespace)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		}
	})

	Context("Verify Weblogic app componenets", func() {
		// GIVEN the ToDoList app is deployed
		// WHEN the servers in the WebLogic domain is ready
		// THEN the domain.servers.status.health.overallHeath fields should be ok
		It("Verify Weblogic 'todo-domain' overall health is ok", func() {
			Eventually(func() bool {
				domain, err := weblogic.GetDomainInCluster(testNamespace, "todo-domain", managedKubeconfig)
				if err != nil {
					return false
				}
				healths, err := weblogic.GetHealthOfServers(domain)
				if err != nil || healths[0] != weblogic.Healthy {
					return false
				}
				return true
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
	})

	Context("Ingress.", func() {
		var host = ""
		var err error
		// Get the host from the Istio gateway resource.
		// GIVEN the Istio gateway for the todo-list namespace
		// WHEN GetHostnameFromGateway is called
		// THEN return the host name found in the gateway.
		It("Get host from gateway.", func() {
			Eventually(func() (string, error) {
				host, err = k8sutil.GetHostnameFromGatewayInCluster(testNamespace, "", managedKubeconfig)
				return host, err
			}, waitTimeout, pollingInterval).Should(Not(BeEmpty()))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the ToDoList app is deployed
		// WHEN the UI is accessed
		// THEN the expected returned page should contain an expected value.
		It("Verify '/todo' UI endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/todo/", host)
				return pkg.GetWebPageInCluster(url, host, managedKubeconfig)
			}, waitTimeout, pollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Derek")))
		})
	})

	Context("Logging", func() {
		indexName := "verrazzano-namespace-mc-todo-list"

		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect the Elasticsearch index for the app exists on the admin cluster Elasticsearch
		It("Verify Elasticsearch index exists on admin cluster", func() {
			Eventually(func() bool {
				return pkg.LogIndexFoundInCluster(indexName, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for todo-list")
		})
	})

	Context("Prometheus Metrics", func() {

		It("Verify scrape_duration_seconds metrics exist for managed cluster", func() {
			Eventually(func() bool {
				m := make(map[string]string)
				m["namespace"] = testNamespace
				m["managed_cluster"] = clusterName
				return pkg.MetricsExistInCluster("scrape_duration_seconds", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find base_jvm_uptime_seconds metric")
		})

		It("Verify DNE scrape_duration_seconds metrics does not exist for managed cluster", func() {
			Eventually(func() bool {
				m := make(map[string]string)
				m["namespace"] = testNamespace
				m["managed_cluster"] = "DNE"
				return pkg.MetricsExistInCluster("scrape_duration_seconds", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeFalse(), "Not expected to find base_jvm_uptime_seconds metric")
		})

		It("Verify container_cpu_cfs_periods_total metrics exist for managed cluster", func() {
			Eventually(func() bool {
				m := make(map[string]string)
				m["namespace"] = testNamespace
				m["managed_cluster"] = clusterName
				return pkg.MetricsExistInCluster("container_cpu_cfs_periods_total", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find container_cpu_cfs_periods_total metric")
		})

		It("Verify istio_request_bytes_bucket metrics exist for managed cluster", func() {
			Eventually(func() bool {
				m := make(map[string]string)
				m["namespace"] = testNamespace
				m["managed_cluster"] = clusterName
				return pkg.MetricsExistInCluster("istio_request_bytes_bucket", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find vendor_requests_count_total metric")

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
				return VerifyTodoListDeleteOnAdminCluster(adminKubeconfig, false, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		It("Verify automatic deletion on managed cluster", func() {
			Eventually(func() bool {
				return VerifyTodoListDeleteOnManagedCluster(managedKubeconfig, testNamespace, testProjectName)
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
		err := pkg.ExecuteClusterDumpWithEnvVarConfig()
		if err != nil {
			return
		}
	}
})

func cleanUp(kubeconfigPath string) error {
	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/mc-todo-list-application.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop application resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/todo-list-components.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/tododb-secret.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/weblogic-domain-secret.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/docker-registry-secret.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster sock-shop component resources: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete sock-shop project resource: %v", err)
	}
	return nil
}
