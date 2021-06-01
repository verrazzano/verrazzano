// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus_test

import (
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"os"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
)

const (
	longPollingInterval = 20 * time.Second
	longWaitTimeout     = 10 * time.Minute
)

var adminKubeconfig = strings.Trim(os.Getenv("ADMIN_KUBECONFIG"), " ")

var _ = ginkgo.Describe("Prometheus", func() {
	// Query Prometheus for the sample metrics from the default scraping jobs
	var _ = ginkgo.Describe("Verify default component metrics", func() {
		if adminKubeconfig == "" {
			ginkgo.Context("Verify metrics from NGINX ingress controller", func() {
				ginkgo.It("Verify sample NGINX metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("nginx_ingress_controller_success", "controller_namespace", "ingress-nginx")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})
			ginkgo.Context("Verify metrics from Container Advisor", func() {
				ginkgo.It("Verify sample Container Advisor metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("container_start_time_seconds", "namespace", "verrazzano-system")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Node Exporter", func() {
				ginkgo.It("Verify sample Node Exporter metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("go_gc_duration_seconds", "job", "node-exporter")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Istio mesh and istiod", func() {
				ginkgo.It("Verify sample mesh metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("istio_tcp_connections_opened_total", "namespace", "verrazzano-system")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
				ginkgo.It("Verify sample istiod metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("sidecar_injection_requests_total", "app", "istiod")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Prometheus", func() {
				ginkgo.It("Verify sample metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist("prometheus_target_interval_length_seconds", "job", "prometheus")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})
		}
	})

	// Query Prometheus for the envoys server statistics
	var _ = ginkgo.Describe("Verify Envoy stats", func() {
		metricName := "envoy_server_stats_recent_lookups"
		if adminKubeconfig == "" {
			ginkgo.It("Verify sample metrics from job envoy-stats", func() {
				gomega.Eventually(func() bool {
					return pkg.MetricsExist(metricName, "job", "envoy-stats")
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
			ginkgo.It("Verify count of metrics for job envoy-stats", func() {
				gomega.Eventually(func() bool {
					return verifyEnvoyStats(metricName)
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
		}
	})
})

func verifyEnvoyStats(metricName string) bool {
	envoyStatsMetric, err := pkg.QueryMetric(metricName, pkg.GetKubeConfigPathFromEnv())
	if err != nil {
		return false
	}

	var countVzSystem int = 0
	var countKyycloak int = 0
	var countIstioSystem int = 0
	var countIngressNginx int = 0
	metrics := pkg.JTq(envoyStatsMetric, "data", "result").([]interface{})
	if metrics != nil {
		for _, metric := range metrics {
			switch ns := pkg.Jq(metric, "metric", "namespace"); ns {
			case "verrazzano-system":
				countVzSystem++
			case "keycloak":
				countKyycloak++
			case "istio-system":
				countIstioSystem++
			case "ingress-nginx":
				countIngressNginx++
			}
		}
	}

	// keycloak and mysql pods
	gomega.Expect(countKyycloak == 2).To(gomega.BeTrue())

	// istio-ingressgateway and istio-egressgateway
	gomega.Expect(countIstioSystem == 2).To(gomega.BeTrue())

	// ingress-controller-ingress-nginx-controller and ingress-controller-ingress-nginx-defaultbackend
	gomega.Expect(countIngressNginx == 2).To(gomega.BeTrue())

	// Pods in verrazzano-system with envoy proxy - verrazzano-console, weblogic-operator, vmi-system-kibana,
	// vmi-system-prometheus, verrazzano-api, vmi-system-grafana, vmi-system-es-master-0, fluentd
	//
	// With the default replica count, prod profile contains additional ES pods vmi-system-es-data-0, vmi-system-es-data-1,
	//vmi-system-es-master-1, and vmi-system-es-master-2 and vmi-system-es-ingest, when compared to dev profile.
	if pkg.IsProdProfile() {
		gomega.Expect(countVzSystem == 13).To(gomega.BeTrue())
	} else if pkg.IsDevProfile() {
		// The dev profile contains 2 additional fluentd pods, when compared to prod profile.
		gomega.Expect(countVzSystem == 10).To(gomega.BeTrue())
	}
	return true
}