// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus

import (
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
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

	// Constants for test metric
	testMetricName       = "tomcat_sessions_created_sessions_total"
	testMetricLabelKey   = "app_oam_dev_component"
	testMetricLabelValue = "deploymetrics-deployment"
)

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
})

var _ = t.Describe("Post upgrade Prometheus", Label("f:observability.logging.es"), func() {

	// GIVEN a running Prometheus instance,
	// WHEN a sample NGINX metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample NGINX metrics can be queried from Prometheus", func() {
		Eventually(func() bool {
			return pkg.MetricsExist(ingressControllerSuccess, controllerNamespace, ingressNginxNamespace)
		}).WithPolling(pollingInterval).WithTimeout(longTimeout).Should(BeTrue())
	})

	// GIVEN a running Prometheus instance,
	// WHEN a sample Container advisor metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample Container Advisor metrics can be queried from Prometheus", func() {
		Eventually(func() bool {
			return pkg.MetricsExist(containerStartTimeSeconds, "", "")
		}).WithPolling(pollingInterval).WithTimeout(longTimeout).Should(BeTrue())
	})

	// GIVEN a running Prometheus instance,
	// WHEN a sample node exporter metric is queried,
	// THEN verify that the metric could be retrieved.
	t.It("Verify sample Node Exporter metrics can be queried from Prometheus", func() {
		Eventually(func() bool {
			return pkg.MetricsExist(cpuSecondsTotal, job, nodeExporter)
		}).WithPolling(pollingInterval).WithTimeout(longTimeout).Should(BeTrue())
	})

	// GIVEN a running Prometheus instance,
	// WHEN checking for the test metric created during pre-upgrade,
	// THEN verify that the metric is present.
	It("Check if the created test metrics is present", func() {
		Eventually(func() bool {
			return pkg.MetricsExist(testMetricName, testMetricLabelKey, testMetricLabelValue)
		}).WithPolling(pollingInterval).WithTimeout(threeMinutes).Should(BeTrue(),
			"Expected not to find any old indices")
	})

})
