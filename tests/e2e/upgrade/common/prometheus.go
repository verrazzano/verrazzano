// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"fmt"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"os"
	"time"
)

const (
	threeMinutes    = 3 * time.Minute
	pollingInterval = 10 * time.Second
	longTimeout     = 10 * time.Minute

	// Constants for sample metrics of system components validated by the test
	ingressControllerSuccess  = "nginx_ingress_controller_success"
	containerStartTimeSeconds = "container_start_time_seconds"
	cpuSecondsTotal           = "node_cpu_seconds_total"

	// Namespaces used for validating envoy stats
	ingressNginxNamespace = "ingress-nginx"

	// Constants for various metric labels, used in the validation
	nodeExporter        = "node-exporter"
	controllerNamespace = "controller_namespace"
	job                 = "job"
	cadvisor            = "cadvisor"
)

// Constants for test deployment with Metric trait
const (
	TestNamespace        = "deploymetrics"
	TestMetricName       = "tomcat_sessions_created_sessions_total"
	TestMetricLabelKey   = "app_oam_dev_component"
	TestMetricLabelValue = "deploymetrics-deployment"
)

var ExpectedPodsDeploymetricsApp = []string{"deploymetrics-workload"}
var clusterName = os.Getenv("CLUSTER_NAME")

func SkipIfPrometheusDisabled() {
	kubeConfigPath := getDefaultKubeConfigPath()
	supported := pkg.IsPrometheusEnabled(kubeConfigPath)
	// Only run tests if Prometheus component is enabled in Verrazzano CR
	if !supported {
		ginkgo.Skip("Prometheus component is not enabled")
	}
}

func InitKubeConfigPath() string {
	adminKubeConfig, present := os.LookupEnv("ADMIN_KUBECONFIG")
	if pkg.IsManagedClusterProfile() {
		if !present {
			ginkgo.Fail(fmt.Sprintln("Environment variable ADMIN_KUBECONFIG is required to run the test"))
		}
	} else {
		adminKubeConfig = getDefaultKubeConfigPath()
	}
	return adminKubeConfig
}

func VerifyScrapeTargets(kubeConfigPath string) func() {
	return func() {
		gomega.Eventually(func() bool {
			scrapeTargets, err := pkg.ScrapeTargetsInCluster(kubeConfigPath)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			return len(scrapeTargets) > 0
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(gomega.BeTrue(),
			"Expected to find at least 1 scraping target")
	}
}

func VerifyNginxMetric(kubeConfigPath string) func() {
	return func() {
		gomega.Eventually(func() bool {
			return pkg.MetricsExistInCluster(ingressControllerSuccess,
				map[string]string{controllerNamespace: ingressNginxNamespace}, kubeConfigPath)
		}).WithPolling(pollingInterval).WithTimeout(longTimeout).Should(gomega.BeTrue())
	}
}

func VerifyContainerAdvisorMetric(kubeConfigPath string) func() {
	return func() {
		gomega.Eventually(func() bool {
			return pkg.MetricsExistInCluster(containerStartTimeSeconds,
				map[string]string{job: cadvisor}, kubeConfigPath)
		}).WithPolling(pollingInterval).WithTimeout(longTimeout).Should(gomega.BeTrue())
	}
}

func VerifyNodeExporterMetric(kubeConfigPath string) func() {
	return func() {
		gomega.Eventually(func() bool {
			return pkg.MetricsExistInCluster(cpuSecondsTotal,
				map[string]string{job: nodeExporter}, kubeConfigPath)
		}).WithPolling(pollingInterval).WithTimeout(longTimeout).Should(gomega.BeTrue())
	}
}

func VerifyDeploymentMetric(kubeConfigPath string) func() {
	return func() {
		gomega.Eventually(func() bool {
			defaultKubeConfigPath, err := k8sutil.GetKubeConfigLocation()
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			label, err := pkg.GetClusterNameMetricLabel(defaultKubeConfigPath)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			pkg.Log(pkg.Info, "Found cluster name metric label - "+label)
			metricLabels := map[string]string{
				TestMetricLabelKey: TestMetricLabelValue,
				label:              getClusterNameForPromQuery(kubeConfigPath),
			}
			return pkg.MetricsExistInCluster(TestMetricName, metricLabels, kubeConfigPath)
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(gomega.BeTrue(),
			"Expected to find test metrics created by application deploy with metrics trait")
	}
}

// Return the cluster name used for the Prometheus query
func getClusterNameForPromQuery(kubeConfigPath string) string {
	if pkg.IsManagedClusterProfile() {
		return clusterName
	}
	isMinVersion110, err := pkg.IsVerrazzanoMinVersion("1.1.0", kubeConfigPath)
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
		ginkgo.Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	return kubeConfigPath
}
