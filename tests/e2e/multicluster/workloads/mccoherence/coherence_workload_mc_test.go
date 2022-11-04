// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mccoherence

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/multicluster"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
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
	helloCoherence = "hello-coherence"
	appNamespace   = "mc-hello-coherence"
	appConfigName  = "hello-appconf"
	projectName    = helloCoherence
	componentName  = helloCoherence

	cohClusterName   = "HelloCoherence"
	podName          = "hello-coh-0"
	containerName    = "coherence"
	appName          = "hello-coh"
	appEndPoint      = "catalogue"
	expectedResponse = "A perfect example of a swivel chair trained calf"

	// metrics
	jvmUptime            = "base_jvm_uptime_seconds"
	vendorRequestsCount  = "vendor_requests_count_total"
	memUsageBytes        = "container_memory_usage_bytes"
	clusterSize          = "clusterSize"
	serviceMessagesLocal = "vendor:coherence_service_messages_local"

	// various labels
	k8sLabelOAMComp              = "kubernetes.labels.app_oam_dev\\/component"
	k8sLabelOAMCompKeyword       = "kubernetes.labels.app_oam_dev\\/component.keyword"
	k8sLabelOAMApp               = "kubernetes.labels.app_oam_dev\\/name"
	k8sLabelContainerName        = "kubernetes.container_name"
	k8sLabelContainerNameKeyword = "kubernetes.container_name.keyword"
	k8sLabelCoherenceCluster     = "kubernetes.labels.coherenceCluster"
	k8sPodName                   = "kubernetes.pod_name"
	labelApp                     = "app"
	labelNS                      = "namespace"
	labelCluster                 = "cluster"
	skipVerifications            = "Skip Verifications"
	skipDeletions                = "Skip Deletions"

	// application resources
	appConfiguration     = "tests/testdata/test-applications/coherence/hello-coherence/hello-coherence-mc-app.yaml"
	compConfiguration    = "tests/testdata/test-applications/coherence/hello-coherence/hello-coherence-comp.yaml"
	projectConfiguration = "tests/testdata/test-applications/coherence/hello-coherence/verrazzano-project.yaml"
)

var (
	appComp           = []string{helloCoherence}
	appPod            = []string{"hello-coh-"}
	clusterName       = os.Getenv("MANAGED_CLUSTER_NAME")
	adminKubeconfig   = os.Getenv("ADMIN_KUBECONFIG")
	managedKubeconfig = os.Getenv("MANAGED_KUBECONFIG")
	failed            = false
	beforeSuitePassed = false
)

var t = framework.NewTestFramework("mccoherence")

var _ = t.AfterEach(func() {
	failed = failed || CurrentSpecReport().Failed()
})

var _ = t.BeforeSuite(func() {
	if !skipDeploy {
		start := time.Now()

		// deploy the VerrazzanoProject
		Eventually(func() error {
			return multicluster.DeployVerrazzanoProject(projectConfiguration, adminKubeconfig)
		}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

		// wait for the namespace to be created on the cluster before deploying app
		Eventually(func() bool {
			return multicluster.TestNamespaceExists(adminKubeconfig, appNamespace)
		}, waitTimeout, pollingInterval).Should(BeTrue())

		Eventually(func() error {
			return multicluster.DeployCompResource(compConfiguration, appNamespace, adminKubeconfig)
		}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())

		Eventually(func() error {
			return multicluster.DeployAppResource(appConfiguration, appNamespace, adminKubeconfig)
		}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
	}
	beforeSuitePassed = true
})

var _ = t.AfterSuite(func() {
	if failed || !beforeSuitePassed {
		err := pkg.ExecuteBugReport(appNamespace)
		if err != nil {
			return
		}
	}
})

var _ = t.Describe("In Multi-cluster, verify Coherence application", Label("f:multicluster.mc-app-lcm"), func() {

	t.Context("Admin Cluster", func() {
		// GIVEN an admin cluster and at least one managed cluster
		// WHEN the example application has been deployed to the admin cluster
		// THEN expect that the multi-cluster resources have been created on the admin cluster
		t.It("Has multi cluster resources", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return multicluster.VerifyMCResources(adminKubeconfig, true, false, appNamespace, appConfigName, appComp)
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
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return multicluster.VerifyMCResources(managedKubeconfig, false, true, appNamespace, appConfigName, appComp)
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
			t.It("Does not have multi cluster resources", func() {
				if skipVerify {
					Skip(skipVerifications)
				}
				Eventually(func() bool {
					return multicluster.VerifyMCResources(kubeconfig, false, false, appNamespace, appConfigName, appComp)
				}, waitTimeout, pollingInterval).Should(BeTrue())
			})
			t.It("Does not have application placed", func() {
				if skipVerify {
					Skip(skipVerifications)
				}
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

	t.Context("for Ingress", Label("f:mesh.ingress"), func() {
		var host = ""
		var err error
		// Get the host from the Istio gateway resource.
		// GIVEN the Istio gateway for the app namespace
		// WHEN GetHostnameFromGateway is called
		// THEN return the host name found in the gateway.
		t.It("Get host from gateway.", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() (string, error) {
				host, err = k8sutil.GetHostnameFromGatewayInCluster(appNamespace, "", managedKubeconfig)
				return host, err
			}, waitTimeout, pollingInterval).Should(Not(BeEmpty()))
		})

		// Verify the application REST endpoint is working.
		// GIVEN the Coherence app is deployed
		// WHEN the UI is accessed
		// THEN the expected returned page should contain an expected value.
		t.It("Verify '/catalogue' endpoint is working.", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() (*pkg.HTTPResponse, error) {
				url := fmt.Sprintf("https://%s/%s", host, appEndPoint)
				return pkg.GetWebPageInCluster(url, host, managedKubeconfig)
			}, waitTimeout, pollingInterval).Should(And(pkg.HasStatus(http.StatusOK), pkg.BodyContains(expectedResponse)))
		})
	})

	t.Context("for Logging", Label("f:observability.logging.es"), func() {
		indexName, err := pkg.GetOpenSearchAppIndexWithKC(appNamespace, adminKubeconfig)
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
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find log index for Coherence application")
		})

		t.It("Verify recent log record exists", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.LogRecordFoundInCluster(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					k8sLabelOAMComp:       componentName,
					k8sLabelOAMApp:        appConfigName,
					k8sLabelContainerName: componentName,
				}, adminKubeconfig)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
		})

		t.It("Verify recent application log record", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.LogRecordFoundInCluster(indexName, time.Now().Add(-24*time.Hour), map[string]string{
					k8sLabelCoherenceCluster:     cohClusterName,
					k8sLabelOAMCompKeyword:       componentName,
					k8sPodName:                   podName,
					k8sLabelContainerNameKeyword: containerName,
				}, adminKubeconfig)
			}, shortWaitTimeout, shortPollingInterval).Should(BeTrue(), "Expected to find a recent log record")
		})
	})

	t.Context("for Prometheus Metrics", Label("f:observability.monitoring.prom"), func() {
		// Coherence metric fix available only from 1.3.0
		if ok, _ := pkg.IsVerrazzanoMinVersion("1.3.0", adminKubeconfig); ok {
			t.It("Retrieve application Prometheus scraped metrics", func() {
				if skipVerify {
					Skip(skipVerifications)
				}
				pkg.Concurrently(
					func() {
						clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
						Eventually(func() bool {
							m := make(map[string]string)
							m[labelApp] = appName
							m[clusterNameMetricsLabel] = clusterName
							return pkg.MetricsExistInCluster(jvmUptime, m, adminKubeconfig)
						}, longWaitTimeout, longPollingInterval).Should(BeTrue())
					},
					func() {
						clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
						Eventually(func() bool {
							m := make(map[string]string)
							m[labelApp] = appName
							m[clusterNameMetricsLabel] = clusterName
							return pkg.MetricsExistInCluster(vendorRequestsCount, m, adminKubeconfig)
						}, longWaitTimeout, longPollingInterval).Should(BeTrue())
					},
					func() {
						clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
						Eventually(func() bool {
							m := make(map[string]string)
							m[labelNS] = appNamespace
							m[clusterNameMetricsLabel] = clusterName
							return pkg.MetricsExistInCluster(memUsageBytes, m, adminKubeconfig)
						}, longWaitTimeout, longPollingInterval).Should(BeTrue())
					},
					func() {
						clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
						Eventually(func() bool {
							m := make(map[string]string)
							m[labelCluster] = cohClusterName
							m[clusterNameMetricsLabel] = clusterName
							return pkg.MetricsExistInCluster(clusterSize, m, adminKubeconfig)
						}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find coherence metric")
					},
					func() {
						clusterNameMetricsLabel, _ := pkg.GetClusterNameMetricLabel(adminKubeconfig)
						Eventually(func() bool {
							m := make(map[string]string)
							m[labelCluster] = cohClusterName
							m[clusterNameMetricsLabel] = clusterName
							return pkg.MetricsExistInCluster(serviceMessagesLocal, m, adminKubeconfig)
						}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find coherence metric")
					},
				)
			})
		}
	})

	t.Context("Delete resources", func() {
		t.It("on admin cluster", func() {
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
				return multicluster.VerifyDeleteOnAdminCluster(adminKubeconfig, false, appNamespace, projectName, appConfigName, appPod)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Verify automatic deletion on managed cluster", func() {
			if skipUndeploy {
				Skip(skipDeletions)
			}
			Eventually(func() bool {
				return multicluster.VerifyDeleteOnManagedCluster(managedKubeconfig, appNamespace, projectName, appConfigName, appPod)
			}, waitTimeout, pollingInterval).Should(BeTrue())
		})

		t.It("Delete test namespace on managed cluster", func() {
			if skipUndeploy {
				Skip(skipDeletions)
			}
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(appNamespace, managedKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})

		t.It("Delete test namespace on admin cluster", func() {
			if skipUndeploy {
				Skip(skipDeletions)
			}
			Eventually(func() error {
				return pkg.DeleteNamespaceInCluster(appNamespace, adminKubeconfig)
			}, waitTimeout, pollingInterval).ShouldNot(HaveOccurred())
		})
	})
})

func cleanUp(kubeconfigPath string) error {
	start := time.Now()
	if err := pkg.DeleteResourceFromFileInClusterInGeneratedNamespace(appConfiguration, kubeconfigPath, appNamespace); err != nil {
		return fmt.Errorf("failed to delete application resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInClusterInGeneratedNamespace(compConfiguration, kubeconfigPath, appNamespace); err != nil {
		return fmt.Errorf("failed to delete component resource: %v", err)
	}

	if err := pkg.DeleteResourceFromFileInCluster(projectConfiguration, kubeconfigPath); err != nil {
		return fmt.Errorf("failed to delete project resource: %v", err)
	}
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
	return nil
}
