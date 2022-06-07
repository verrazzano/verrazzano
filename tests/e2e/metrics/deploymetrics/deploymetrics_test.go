// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package deploymetrics

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"os"
	"time"

	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

const (
	deploymetricsCompYaml = "testdata/deploymetrics/deploymetrics-comp.yaml"
	deploymetricsCompName = "deploymetrics-deployment"
	deploymetricsAppYaml  = "testdata/deploymetrics/deploymetrics-app.yaml"
	deploymetricsAppName  = "deploymetrics-appconf"
	skipVerifications     = "Skip Verifications"
)

var (
	expectedPodsDeploymetricsApp = []string{"deploymetrics-workload"}
	generatedNamespace           = pkg.GenerateNamespace("deploymetrics")

	waitTimeout              = 10 * time.Minute
	pollingInterval          = 30 * time.Second
	shortPollingInterval     = 10 * time.Second
	shortWaitTimeout         = 5 * time.Minute
	longWaitTimeout          = 15 * time.Minute
	longPollingInterval      = 30 * time.Second
	imagePullWaitTimeout     = 40 * time.Minute
	imagePullPollingInterval = 30 * time.Second

	adminKubeConfig   string
	promConfigJobName string
	clusterNameLabel  string

	t = framework.NewTestFramework("deploymetrics")
)

var clusterDump = pkg.NewClusterDumpWrapper()
var _ = clusterDump.BeforeSuite(func() {
	if !skipDeploy {
		deployMetricsApplication()
	}
	var err error
	clusterNameLabel, err = pkg.GetClusterNameMetricLabel(getDefaultKubeConfigPath())
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		Fail(err.Error())
	}
	initKubeConfigPath()
	initPromConfigJobName()
})
var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	if !skipUndeploy {
		undeployMetricsApplication()
	}
})

func deployMetricsApplication() {
	t.Logs.Info("Deploy DeployMetrics Application")

	t.Logs.Info("Create namespace")
	start := time.Now()
	Eventually(func() (*v1.Namespace, error) {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    istioInjection}
		return pkg.CreateNamespace(namespace, nsLabels)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create component resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(deploymetricsCompYaml, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFileInGeneratedNamespace(deploymetricsAppYaml, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred(), "Failed to create DeployMetrics application resource")

	Eventually(func() bool {
		return pkg.ContainerImagePullWait(namespace, expectedPodsDeploymetricsApp)
	}, imagePullWaitTimeout, imagePullPollingInterval).Should(BeTrue())

	t.Logs.Info("Verify deploymetrics-workload pod is running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(namespace, expectedPodsDeploymetricsApp)
		if err != nil {
			AbortSuite(fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", namespace, err))
		}
		return result
	}, waitTimeout, pollingInterval).Should(BeTrue())
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
}

func undeployMetricsApplication() {
	t.Logs.Info("Undeploy DeployMetrics Application")

	t.Logs.Info("Delete application")
	start := time.Now()
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(deploymetricsCompYaml, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Delete components")
	Eventually(func() error {
		return pkg.DeleteResourceFromFileInGeneratedNamespace(deploymetricsAppYaml, namespace)
	}, shortWaitTimeout, shortPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for pods to terminate")
	Eventually(func() bool {
		podsNotRunning, _ := pkg.PodsNotRunning(namespace, expectedPodsDeploymetricsApp)
		return podsNotRunning
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	Eventually(func() bool {
		return pkg.IsAppInPromConfig(promConfigJobName)
	}, waitTimeout, pollingInterval).Should(BeFalse(), "Expected App to be removed from Prometheus Config")

	t.Logs.Info("Delete namespace")
	Eventually(func() error {
		return pkg.DeleteNamespace(namespace)
	}, longWaitTimeout, longPollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Wait for Finalizer to be removed")
	Eventually(func() bool {
		return pkg.CheckNamespaceFinalizerRemoved(namespace)
	}, shortWaitTimeout, shortPollingInterval).Should(BeTrue())

	t.Logs.Info("Waiting for namespace deletion")
	Eventually(func() bool {
		_, err := pkg.GetNamespace(namespace)
		return err != nil && errors.IsNotFound(err)
	}, longWaitTimeout, longPollingInterval).Should(BeTrue())
	metrics.Emit(t.Metrics.With("undeployment_elapsed_time", time.Since(start).Milliseconds()))
}

var _ = t.Describe("DeployMetrics Application test", Label("f:app-lcm.oam"), func() {

	t.Context("for Prometheus Config.", Label("f:observability.monitoring.prom"), func() {
		t.It(fmt.Sprintf("Verify that Prometheus Config Data contains %s", promConfigJobName), func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			Eventually(func() bool {
				return pkg.IsAppInPromConfig(promConfigJobName)
			}, waitTimeout, pollingInterval).Should(BeTrue(), "Expected to find App in Prometheus Config")
		})
	})

	t.Context("Retrieve Prometheus scraped metrics for", Label("f:observability.monitoring.prom"), func() {
		t.It("App Component", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			metricLabels := map[string]string{
				"app_oam_dev_name": "deploymetrics-appconf",
				clusterNameLabel:   getClusterNameForPromQuery(),
			}
			Eventually(func() bool {
				return pkg.MetricsExistInCluster("http_server_requests_seconds_count", metricLabels, adminKubeConfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Component.")
		})
		t.It("App Config", func() {
			if skipVerify {
				Skip(skipVerifications)
			}
			metricLabels := map[string]string{
				"app_oam_dev_component": "deploymetrics-deployment",
				clusterNameLabel:        getClusterNameForPromQuery(),
			}
			Eventually(func() bool {
				return pkg.MetricsExistInCluster("tomcat_sessions_created_sessions_total", metricLabels, adminKubeConfig)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for App Config.")
		})
	})

})

// Return the cluster name used for the Prometheus query
func getClusterNameForPromQuery() string {
	if pkg.IsManagedClusterProfile() {
		var clusterName = os.Getenv("CLUSTER_NAME")
		pkg.Log(pkg.Info, "This is a managed cluster, returning cluster name - "+clusterName)
		return clusterName
	}
	isMinVersion110, err := pkg.IsVerrazzanoMinVersion("1.1.0", adminKubeConfig)
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		return ""
	}
	if isMinVersion110 {
		return "local"
	}
	return ""
}

func getDefaultKubeConfigPath() string {
	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	pkg.Log(pkg.Info, "Default kube config path -"+kubeConfigPath)
	return kubeConfigPath
}

func initKubeConfigPath() {
	var present bool
	adminKubeConfig, present = os.LookupEnv("ADMIN_KUBECONFIG")
	if pkg.IsManagedClusterProfile() {
		if !present {
			Fail(fmt.Sprintln("Environment variable ADMIN_KUBECONFIG is required to run the test"))
		}
	} else {
		adminKubeConfig = getDefaultKubeConfigPath()
	}
	pkg.Log(pkg.Info, "Initialized kube config path - "+adminKubeConfig)
}

func initPromConfigJobName() {
	isPromJobNameInNewFmt, err := pkg.IsVerrazzanoMinVersion("1.4.0", adminKubeConfig)
	if err != nil {
		pkg.Log(pkg.Error, err.Error())
		Fail(err.Error())
	}
	if isPromJobNameInNewFmt {
		// For VZ versions starting from 1.4.0, the job name in prometheus scrape config is of the format
		// <app_name>_<app_namespace>
		promConfigJobName = fmt.Sprintf("%s_%s", deploymetricsAppName, namespace)
	} else {
		// For VZ versions prior to 1.4.0, the job name in prometheus scrape config was of the old format
		// <app_name>_default_<app_namespace>_<app_component_name>
		promConfigJobName =
			fmt.Sprintf("%s_default_%s_%s", deploymetricsAppName, generatedNamespace, deploymetricsCompName)
	}
}
