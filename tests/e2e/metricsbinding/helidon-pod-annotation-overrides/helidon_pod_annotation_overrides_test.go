// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package helidonpodannotation

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/test/framework"
	"github.com/verrazzano/verrazzano/pkg/test/framework/metrics"
	"github.com/verrazzano/verrazzano/tests/e2e/metricsbinding"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longWaitTimeout      = 15 * time.Minute
	longPollingInterval  = 20 * time.Second
	namespace            = "hello-helidon-namespace"
	applicationPodPrefix = "hello-helidon-deployment-"
	yamlPath             = "tests/e2e/metricsbinding/testdata/hello-helidon-deployment-pod-annotated.yaml"
	promConfigJobName    = "hello-helidon-namespace_hello-helidon-deployment_apps_v1_Deployment"

	PrometheusPortAnnotation   = "prometheus.io/port"
	PrometheusPathAnnotation   = "prometheus.io/path"
	PrometheusScrapeAnnotation = "prometheus.io/scrape"

	PrometheusPortDefault    = "8080"
	PrometheusPathDefault    = "/metrics"
	PrometheusScrapeOverride = "false"
)

var t = framework.NewTestFramework("helidonpodannotation")

var _ = t.BeforeSuite(func() {
	start := time.Now()
	metricsbinding.DeployApplication(namespace, yamlPath)
	metrics.Emit(t.Metrics.With("deployment_elapsed_time", time.Since(start).Milliseconds()))
})

var clusterDump = pkg.NewClusterDumpWrapper()
var _ = clusterDump.AfterEach(func() {}) // Dump cluster if spec fails
var _ = clusterDump.AfterSuite(func() {  // Dump cluster if aftersuite fails
	metricsbinding.UndeployApplication(namespace, yamlPath, promConfigJobName)
})

var _ = t.AfterEach(func() {})

var _ = t.Describe("Verify", Label("f:app-lcm.poko"), func() {
	t.Context("app deployment", func() {
		// GIVEN the app is deployed
		// WHEN the running pods are checked
		// THEN the Helidon pod should exist
		t.It("Verify 'hello-helidon-deployment' pod is running", func() {
			Eventually(func() bool {
				return pkg.PodsRunning(namespace, []string{applicationPodPrefix})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})
	})

	// GIVEN the Helidon app is deployed and the pods are running
	// WHEN the Prometheus metrics in the app namespace are scraped with the scrape endpoint set to false
	// THEN the Helidon application metrics should not exist
	t.Context("Verify Prometheus scraped metrics.", Label("f:observability.monitoring.prom"), func() {
		t.It("Retrieve Prometheus scraped metrics for 'hello-helidon-deployment' Pod", func() {
			Eventually(func() bool {
				kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
					return false
				}
				clientset, err := pkg.GetKubernetesClientsetForCluster(kubeconfigPath)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error creating clientset from kubeconfig, error: %v", err))
					return false
				}
				pods, err := pkg.ListPodsInCluster("hello-helidon-namespace", clientset)
				if err != nil {
					pkg.Log(pkg.Error, fmt.Sprintf("Error listing pods in the namespace hello-helidon-namespace, error: %v", err))
					return false
				}
				podItems := pods.Items
				if len(podItems) != 1 {
					return false
				}
				podAnnotations := podItems[0].GetAnnotations()
				return podAnnotations[PrometheusPortAnnotation] == PrometheusPortDefault &&
					podAnnotations[PrometheusPathAnnotation] == PrometheusPathDefault &&
					podAnnotations[PrometheusScrapeAnnotation] == PrometheusScrapeOverride
			}, longWaitTimeout, longPollingInterval).Should(BeTrue(), "Expected to find Prometheus scraped metrics for Helidon application.")
			Eventually(func() bool {
				return pkg.MetricsExist("base_jvm_uptime_seconds", "app_verrazzano_io_workload", "hello-helidon-deployment-apps-v1-deployment")
			}, longWaitTimeout, longPollingInterval).Should(BeFalse(), "Expected to find Prometheus scraped metrics for Helidon application.")
			Eventually(func() bool {
				return pkg.MetricsExist("base_jvm_uptime_seconds", "job", promConfigJobName)
			}, longWaitTimeout, longPollingInterval).Should(BeFalse(), "Expected to find Prometheus scraped metrics for Helidon application.")
			Eventually(func() bool {
				return pkg.MetricsExist("base_jvm_uptime_seconds", "test_namespace", "hello-helidon-namespace-test")
			}, longWaitTimeout, longPollingInterval).Should(BeFalse(), "Expected not to find Prometheus scraped metrics for Helidon application.")
		})
	})
})
