// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mcweblogic

import (
	"net/http"
	"os"
	"strconv"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8s/resource"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/weblogic"
	m1 "k8s.io/api/core/v1"

	"fmt"
	"time"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"

	"github.com/verrazzano/verrazzano/tests/e2e/multicluster"
)

const (
	// wait intervals
	pollingInterval      = 5 * time.Second
	waitTimeout          = 5 * time.Minute
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	consistentlyDuration = 1 * time.Minute
	longWaitTimeout      = 10 * time.Minute
	longPollingInterval  = 20 * time.Second

	// application specific constants
	appNamespace   = "mc-hello-wls"
	appConfigName  = "hello-appconf"
	projectName    = "hello-wls"
	componentName  = "hello-domain"
	appURL         = "hello/weblogic/greetings/message"
	welcomeMessage = "Hello WebLogic"
	ns             = "namespace"
	adminServer    = "AdminServer"
	adminServerPod = "hellodomain-adminserver"
	wlDomain       = "hellodomain"
	wlServer       = "weblogic-server"

	// kubernetes secrets
	dockerSecret = "hellodomain-repo-credentials"
	domainSecret = "hellodomain-weblogic-credentials"

	// metrics
	scrapeDuration   = "scrape_duration_seconds"
	serverState      = "wls_server_state_val"
	cpuLoad          = "wls_jvm_process_cpu_load"
	pendingSendBytes = "envoy_cluster_http2_pending_send_bytes"
	receivedBytes    = "istio_tcp_received_bytes_total"

	// various labels
	k8sLabelDomainUID     = "kubernetes.labels.weblogic_domainUID"
	k8sLabelWLServerName  = "kubernetes.labels.weblogic_serverName"
	k8sPodName            = "kubernetes.pod_name"
	k8sLabelContainerName = "kubernetes.container_name"
	labelDomainName       = "weblogic_domainName"
	labelPodName          = "pod_name"

	// application resources
	appConfiguration                  = "tests/testdata/test-applications/weblogic/hello-weblogic/hello-wls-mc-app.yaml"
	compConfiguration                 = "tests/testdata/test-applications/weblogic/hello-weblogic/hello-wls-comp.yaml"
	projectConfiguration              = "tests/testdata/test-applications/weblogic/hello-weblogic/verrazzano-project.yaml"
	wlDomainSecretConfiguration       = "tests/testdata/test-applications/weblogic/hello-weblogic/weblogic-domain-secret.yaml"
	dockerRegistrySecretConfiguration = "tests/testdata/test-applications/weblogic/hello-weblogic/docker-registry-secret.yaml"
)

var (
	appComp           = []string{"hello-domain"}
	appPod            = []string{"hellodomain-adminserver"}
	clusterName       = os.Getenv("MANAGED_CLUSTER_NAME")
	adminKubeconfig   = os.Getenv("ADMIN_KUBECONFIG")
	managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")
	failed            = false
	beforeSuitePassed = false
)

var t = framework.NewTestFramework("mcweblogic")

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.BeforeSuite(func() {

	start := time.Now()

	wlsUser := "weblogic"
	wlsPass := pkg.GetRequiredEnvVarOrFail("WEBLOGIC_PSW")
	regServ := pkg.GetRequiredEnvVarOrFail("OCR_REPO")
	regUser := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_USR")
	regPass := pkg.GetRequiredEnvVarOrFail("OCR_CREDS_PSW")

	// deploy the VerrazzanoProject
	Eventually(func() error {
		return multicluster.DeployVerrazzanoProject(projectConfiguration, adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	// wait for the namespace to be created on the cluster before deploying app
	Eventually(func() bool {
		return multicluster.TestNamespaceExists(adminKubeconfig, appNamespace)
	}, waitTimeout, pollingInterval).Should(BeTrue())

	// create Docker repository secret
	Eventually(func() (*m1.Secret, error) {
		return pkg.CreateDockerSecretInCluster(appNamespace, dockerSecret, regServ, regUser, regPass, adminKubeconfig)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	// create WebLogic credentials secret
	Eventually(func() (*m1.Secret, error) {
		return pkg.CreateCredentialsSecretInCluster(appNamespace, domainSecret, wlsUser, wlsPass, nil, adminKubeconfig)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	Eventually(func() error {
		return multicluster.DeployCompResource(compConfiguration, appNamespace, adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	Eventually(func() error {
		return multicluster.DeployAppResource(appConfiguration, appNamespace, adminKubeconfig)
	}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

	beforeSuitePassed = true
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		err := pkg.ExecuteBugReport(appNamespace)
		if err != nil {
			return
		}
	}
})

var _ = t.Describe("In Multi-cluster, verify WebLogic application", Label("f:multicluster.mc-app-lcm"), func() {

	t.Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		t.It("Has multi cluster resources", func() {
			Eventually(func() bool {
				return multicluster.VerifyMCResources(adminKubeconfig, true, false, appNamespace, appConfigName, appComp)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		// GIVEN an admin cluster
		// WHEN the multi-cluster example application has been created on admin cluster but not placed there
		// THEN expect that the app is not deployed to the admin cluster consistently for some length of time
		t.It("Does not have application placed", func() {
			Consistently(func() bool {
				result, err := multicluster.VerifyAppResourcesInCluster(adminKubeconfig, true, false, projectName, appNamespace, appPod)
				if err != nil {
					AbortSuite(fmt.Sprintf("Verification of application resources failed for admin cluster, error: %v", err))
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
				return multicluster.VerifyMCResources(managedKubeconfig, false, true, appNamespace, appConfigName, appComp)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the multi-cluster example application has been created on admin cluster and placed in managed cluster
		// THEN expect that the app is deployed to the managed cluster
		t.It("Has application placed", func() {
			Eventually(func() bool {
				result, err := multicluster.VerifyAppResourcesInCluster(managedKubeconfig, false, true, projectName, appNamespace, appPod)
				if err != nil {
					AbortSuite(fmt.Sprintf("Verification of application resources failed for managed cluster, error: %v", err))
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
					return multicluster.VerifyMCResources(kubeconfig, false, false, appNamespace, appConfigName, appComp)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
			It("Does not have application placed", func() {
				Eventually(func() bool {
					result, err := multicluster.VerifyAppResourcesInCluster(kubeconfig, false, false, projectName, appNamespace, appPod)
					if err != nil {
						AbortSuite(fmt.Sprintf("Verification of application resources failed for the cluster with kubeconfig: %v, error: %v", kubeconfig, err))
					}
					return result
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
		}
	})

	t.Context("for WebLogic components", func() {
		// GIVEN the app is deployed
		// WHEN the servers in the WebLogic domain is ready
		// THEN the domain.servers.status.health.overallHeath fields should be ok
		t.It("Verify 'hello-domain' overall health is ok", func() {
			Eventually(func() bool {
				domain, err := weblogic.GetDomainInCluster(appNamespace, componentName, managedKubeconfig)
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
		// GIVEN the Istio gateway for the app namespace
		// WHEN GetHostnameFromGateway is called
		// THEN return the host name found in the gateway.
		t.It("Get host from gateway.", func() {
			Eventually(func() (string, error) {
				host, err = k8sutil.GetHostnameFromGatewayInCluster(appNamespace, "", managedKubeconfig)
				return host, err
			}, waitTimeout, pollingInterval).Should(Not(BeEmpty()))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the WebLogic app is deployed
		// WHEN the UI is accessed
		// THEN the expected returned page should contain an expected value.
		t.It("Verify sample endpoint is working.", func() {
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/%s", host, appURL)
				return pkg.GetWebPageInCluster(url, host, managedKubeconfig)
			}, waitTimeout, pollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyEquals(welcomeMessage)))
		})
	})

	t.Context("for Logging", Label("f:observability.logging.es"), func() {
		indexName, err := pkg.GetOpenSearchAppIndexWithKC(appNamespace, adminKubeconfig)
		Expect(err).To(BeNil())
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect the Elasticsearch index for the app exists on the admin cluster Elasticsearch
		t.It("Verify Elasticsearch index exists on admin cluster", func() {
			Eventually(func() bool {
				return pkg.LogIndexFoundInCluster(indexName, adminKubeconfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for WebLogic application")
		})

		t.It("Verify recent hellodomain-adminserver log record exists", func() {
			Eventually(func() bool {
				return pkg.LogRecordFoundInCluster(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					k8sLabelDomainUID:     wlDomain,
					k8sLabelWLServerName:  adminServer,
					k8sPodName:            adminServerPod,
					k8sLabelContainerName: wlServer,
				}, adminKubeconfig)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
		})
	})

	t.Context("for Prometheus Metrics", Label("f:observability.monitoring.prom"), func() {

		t.It("Retrieve application Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
					Eventually(func() bool {
						m := make(map[string]string)
						m[ns] = appNamespace
						m[clusterNameMetricsLabel] = clusterName
						return pkg.MetricsExistInCluster(scrapeDuration, m, adminKubeconfig)
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
					Eventually(func() bool {
						m := make(map[string]string)
						m[ns] = appNamespace
						m[clusterNameMetricsLabel] = clusterName
						m[labelDomainName] = wlDomain
						return pkg.MetricsExistInCluster(serverState, m, adminKubeconfig)
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
					Eventually(func() bool {
						m := make(map[string]string)
						m[ns] = appNamespace
						m[clusterNameMetricsLabel] = clusterName
						m[labelDomainName] = wlDomain
						return pkg.MetricsExistInCluster(cpuLoad, m, adminKubeconfig)
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
			)
		})

		t.It("Retrieve Istio Prometheus scraped metrics", func() {
			pkg.Concurrently(
				func() {
					clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
					Eventually(func() bool {
						m := make(map[string]string)
						m[ns] = appNamespace
						m[clusterNameMetricsLabel] = clusterName
						m[labelPodName] = adminServerPod
						return pkg.MetricsExistInCluster(pendingSendBytes, m, adminKubeconfig)
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
				func() {
					clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
					Eventually(func() bool {
						m := make(map[string]string)
						m[ns] = appNamespace
						m[clusterNameMetricsLabel] = clusterName
						m["destination_canonical_service"] = componentName
						return pkg.MetricsExistInCluster(receivedBytes, m, adminKubeconfig)
					}, longWaitTimeout, longPollingInterval).Should(BeTrue())
				},
			)
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
				return multicluster.VerifyDeleteOnAdminCluster(adminKubeconfig, false, appNamespace, projectName, appConfigName, appPod)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Verify automatic deletion on managed cluster", func() {
			Eventually(func() bool {
				return multicluster.VerifyDeleteOnManagedCluster(managedKubeconfig, appNamespace, projectName, appConfigName, appPod)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Delete test namespace on managed cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(appNamespace, managedKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Delete test namespace on admin cluster", func() {
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(appNamespace, adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})
	})
})

func cleanUp(kubeconfigPath string) error {
	start := time.Now()
	file, err := pkg.FindTestDataFile(appConfiguration)
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeconfigPath, appNamespace); err != nil {
		return fmt.Errorf("failed to delete application resource: %v", err)
	}
	file, err = pkg.FindTestDataFile(compConfiguration)
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeconfigPath, appNamespace); err != nil {
		return fmt.Errorf("failed to delete component resource: %v", err)
	}
	file, err = pkg.FindTestDataFile(wlDomainSecretConfiguration)
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeconfigPath, appNamespace); err != nil {
		return fmt.Errorf("failed to delete WebLogic domain secret: %v", err)
	}
	file, err = pkg.FindTestDataFile(dockerRegistrySecretConfiguration)
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInClusterInGeneratedNamespace(file, kubeconfigPath, appNamespace); err != nil {
		return fmt.Errorf("failed to delete docker registry secret: %v", err)
	}

	file, err = pkg.FindTestDataFile(projectConfiguration)
	if err != nil {
		return err
	}
	if err := resource.DeleteResourceFromFileInCluster(file, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete project resource: %v", err)
	}
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	return nil
}
