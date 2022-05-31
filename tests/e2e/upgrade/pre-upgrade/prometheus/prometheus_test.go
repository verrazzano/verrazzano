// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	v1 "k8s.io/api/core/v1"
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
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

	// Constants for test metric
	testNamespace        = "deploymetrics"
	testMetricName       = "tomcat_sessions_created_sessions_total"
	testMetricLabelKey   = "app_oam_dev_component"
	testMetricLabelValue = "deploymetrics-deployment"
)

var expectedPodsDeploymetricsApp = []string{"deploymetrics-workload"}
var adminKubeConfig string
var isManagedClusterProfile bool

var t = framework.NewTestFramework("prometheus")

var _ = t.BeforeSuite(func() {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Fail(fmt.Sprintf("Failed to get default kubeconfig path: %s", err.Error()))
	}
	supported := pkg.IsPrometheusEnabled(kubeconfigPath)
	// Only run tests if Prometheus component is enabled in Verrazzano CR
	if !supported {
		Skip("Prometheus component is not enabled")
	}
	var present bool
	adminKubeConfig, present = os.LookupEnv("ADMIN_KUBECONFIG")
	isManagedClusterProfile = pkg.IsManagedClusterProfile()
	if isManagedClusterProfile {
		if !present {
			Fail(fmt.Sprintln("Environment variable ADMIN_KUBECONFIG is required to run the test"))
		}
	} else {
		adminKubeConfig = kubeconfigPath
	}
	deployMetricsApplication()
})

var _ = t.Describe("Pre upgrade Prometheus", Label("f:observability.logging.es"), func() {

	// GIVEN a running Prometheus instance,
	// WHEN a scrape config is created,
	// THEN verify that the scrape config is created correctly
	It("Scrape targets can be listed and there is at least 1 scrape target", func() {
		Eventually(func() bool {
			scrapeTargets, err := pkg.ScrapeTargetsInCluster(adminKubeConfig)
			if err != nil {
				pkg.Log(pkg.Error, err.Error())
				return false
			}
			return len(scrapeTargets) > 0
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(),
			"Expected to find at least 1 scraping target")
	})

	// GIVEN a running Prometheus instance,
	// WHEN a sample NGINX metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample NGINX metrics can be queried from Prometheus", func() {
		Eventually(func() bool {
			return pkg.MetricsExistInCluster(ingressControllerSuccess,
				map[string]string{controllerNamespace: ingressNginxNamespace}, adminKubeConfig)
		}).WithPolling(pollingInterval).WithTimeout(longTimeout).Should(BeTrue())
	})

	// GIVEN a running Prometheus instance,
	// WHEN a sample Container advisor metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample Container Advisor metrics can be queried from Prometheus", func() {
		Eventually(func() bool {
			return pkg.MetricsExist(containerStartTimeSeconds, job, cadvisor)
		}).WithPolling(pollingInterval).WithTimeout(longTimeout).Should(BeTrue())
	})

	// GIVEN a running Prometheus instance,
	// WHEN a sample node exporter metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample Node Exporter metrics can be queried from Prometheus", func() {
		Eventually(func() bool {
			return pkg.MetricsExistInCluster(cpuSecondsTotal,
				map[string]string{job: nodeExporter}, adminKubeConfig)
		}).WithPolling(pollingInterval).WithTimeout(longTimeout).Should(BeTrue())
	})

	// GIVEN a running Prometheus instance,
	// WHEN a metric is created,
	// THEN verify that the metric is persisted in the prometheus time series DB.
	It("Validate if the test metric created by the test OAM deployment exists", func() {
		Eventually(func() bool {
			return pkg.MetricsExistInCluster(testMetricName,
				map[string]string{testMetricLabelKey: testMetricLabelValue}, adminKubeConfig)
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(),
			"Expected to find test metrics created by application deploy with metrics trait")
	})

})

func deployMetricsApplication() {
	t.Logs.Info("Deploy DeployMetrics Application")
	Eventually(func() *v1.Namespace {
		nsLabels := map[string]string{
			"verrazzano-managed": "true",
			"istio-injection":    "true"}
		ns, err := pkg.CreateNamespace(testNamespace, nsLabels)
		if err != nil && strings.Contains(err.Error(), "already exists") {
			ns, _ = pkg.GetNamespace(testNamespace)
			return ns
		}
		return ns
	}, threeMinutes, pollingInterval).ShouldNot(BeNil())

	t.Logs.Info("Create component resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/deploymetrics/deploymetrics-comp.yaml")
	}, threeMinutes, pollingInterval).ShouldNot(HaveOccurred())

	t.Logs.Info("Create application resource")
	Eventually(func() error {
		return pkg.CreateOrUpdateResourceFromFile("testdata/deploymetrics/deploymetrics-app.yaml")
	}, threeMinutes, pollingInterval).ShouldNot(HaveOccurred(), "Failed to create DeployMetrics application resource")

	Eventually(func() bool {
		return pkg.ContainerImagePullWait(testNamespace, expectedPodsDeploymetricsApp)
	}, threeMinutes, pollingInterval).Should(BeTrue())

	t.Logs.Info("Verify deploymetrics-workload pod is running")
	Eventually(func() bool {
		result, err := pkg.PodsRunning(testNamespace, expectedPodsDeploymetricsApp)
		if err != nil {
			pkg.Log(pkg.Error, fmt.Sprintf("One or more pods are not running in the namespace: %v, error: %v", testNamespace, err))
			return false
		}
		return result
	}, threeMinutes, pollingInterval).Should(BeTrue())
}
