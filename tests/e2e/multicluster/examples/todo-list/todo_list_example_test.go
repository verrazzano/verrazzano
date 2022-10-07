// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package todo_list

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/weblogic"
	m1 "k8s.io/api/core/v1"
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
var beforeSuitePassed = false

var t = framework.NewTestFramework("todo_list")

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.BeforeSuite(func() {
	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	dbPass := pkg.GetRequiredEnvVarOrFail("DATABASE_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	// deploy the VerrazzanoProject
	start := time.Now()
	Eventually(func() error {
		return DeployTodoListProject(adminKubeconfig, sourceDir)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// wait for the namespace to be created on the cluster before deploying app
	Eventually(func() bool {
		return TodoListNamespaceExists(adminKubeconfig, testNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue())

	// create Docker repository secret
	Eventually(func() (*m1.Secret, error) {
		return pkg.CreateDockerSecretInCluster(testNamespace, "tododomain-repo-credentials", regServ, regUser, regPass, adminKubeconfig)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	// create WebLogic credentials secret
	Eventually(func() (*m1.Secret, error) {
		return pkg.CreateCredentialsSecretInCluster(testNamespace, "tododomain-weblogic-credentials", wlsUser, wlsPass, nil, adminKubeconfig)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	// create database credentials secret
	Eventually(func() (*m1.Secret, error) {
		return pkg.CreateCredentialsSecretInCluster(testNamespace, "tododomain-jdbc-tododb", wlsUser, dbPass, map[string]string{"weblogic.domainUID": "tododomain"}, adminKubeconfig)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	Eventually(func() error {
		return DeployTodoListApp(adminKubeconfig, sourceDir)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.Describe("In Multi-cluster, verify todo-list", Label("f:multicluster.mc-app-lcm"), func() {
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
				result, err := VerifyTodoListInCluster(adminKubeconfig, true, false, testProjectName, testNamespace)
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
				result, err := VerifyTodoListInCluster(managedKubeconfig, false, true, testProjectName, testNamespace)
				if err != nil {
					AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
				}
				return result
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
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
			It("Does not have multi cluster resources", func() {
				Eventually(func() bool {
					return VerifyMCResources(kubeconfig, false, false, testNamespace)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
			It("Does not have application placed", func() {
				Eventually(func() bool {
					result, err := VerifyTodoListInCluster(kubeconfig, false, false, testProjectName, testNamespace)
					if err != nil {
						AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
					}
					return result
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		}
	})

	t.Context("for WebLogic components", func() {
		// GIVEN the ToDoList app is deployed
		// WHEN the servers in the WebLogic domain is ready
		// THEN the domain.servers.status.health.overallHeath fields should be ok
		t.It("Verify 'todo-domain' overall health is ok", func() {
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

	t.Context("for Ingress", Label("f:mesh.ingress"), func() {
		var host = ""
		var err error
		// Get the host from the Istio gateway resource.
		// GIVEN the Istio gateway for the todo-list namespace
		// WHEN GetHostnameFromGateway is called
		// THEN return the host name found in the gateway.
		t.It("Get host from gateway.", func() {
			Eventually(func() (string, error) {
				host, err = k8sutil.GetHostnameFromGatewayInCluster(testNamespace, "", managedKubeconfig)
				return host, err
			}, waitTimeout, pollingInterval).Should(Not(BeEmpty()))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the ToDoList app is deployed
		// WHEN the UI is accessed
		// THEN the expected returned page should contain an expected value.
		t.It("Verify '/todo' UI endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/todo/", host)
				return pkg.GetWebPageInCluster(url, host, managedKubeconfig)
			}, waitTimeout, pollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains("Derek")))
		})
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
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for todo-list")
		})
	})

	t.Context("for Prometheus Metrics", Label("f:observability.monitoring.prom"), func() {

		t.It("Verify scrape_duration_seconds metrics exist for managed cluster", func() {
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
			Eventually(func() bool {
				m := make(map[string]string)
				m["namespace"] = testNamespace
				m[clusterNameMetricsLabel] = clusterName
				return pkg.MetricsExistInCluster("scrape_duration_seconds", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find base_jvm_uptime_seconds metric")
		})

		t.It("Verify DNE scrape_duration_seconds metrics does not exist for managed cluster", func() {
			clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
			Eventually(func() bool {
				m := make(map[string]string)
				m["namespace"] = testNamespace
				m[clusterNameMetricsLabel] = "DNE"
				return pkg.MetricsExistInCluster("scrape_duration_seconds", m, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeFalse(), "Not expected to find base_jvm_uptime_seconds metric")
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
	})

	t.Context("Delete resources", func() {
		t.It("on admin cluster", func() {
			Eventually(func() error {
				return cleanUp(adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Verify deletion on admin cluster", func() {
			Eventually(func() bool {
				return VerifyTodoListDeleteOnAdminCluster(adminKubeconfig, false, testNamespace, testProjectName)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Verify automatic deletion on managed cluster", func() {
			Eventually(func() bool {
				return VerifyTodoListDeleteOnManagedCluster(managedKubeconfig, testNamespace, testProjectName)
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
	if err := resource.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/mc-todo-list-application.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster todo-list application resource: %v", err)
	}

	if err := resource.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/todo-list-components.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster todo-list component resources: %v", err)
	}

	if err := resource.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/tododb-secret.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster todo-list component resources: %v", err)
	}

	if err := resource.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/weblogic-domain-secret.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster todo-list component resources: %v", err)
	}

	if err := resource.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/docker-registry-secret.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete multi-cluster todo-list component resources: %v", err)
	}

	if err := resource.DeleteResourceFromFileInCluster(fmt.Sprintf("examples/multicluster/%s/verrazzano-project.yaml", sourceDir), kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete todo-list project resource: %v", err)
	}
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	return nil
}
