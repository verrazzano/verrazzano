// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mc_prometheus_test

import (
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"os"
	"strconv"
	"time"

	"github.com/onsi/ginkgo"
)

const (
	longPollingInterval = 20 * time.Second
	longWaitTimeout     = 10 * time.Minute
)

var managedPrefix = os.Getenv("MANAGED_CLUSTER_PREFIX")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")
var totalClusters = os.Getenv("TOTAL_CLUSTERS")

var _ = ginkgo.Describe("Prometheus", func() {
	isManagedClusterProfile := pkg.IsManagedClusterProfile()
	// Query Prometheus for the sample metrics from the default scraping jobs
	var _ = ginkgo.Describe("Verify default component metrics", func() {
		if !isManagedClusterProfile {
			ginkgo.Context("Verify metrics from NGINX ingress controller", func() {
				ginkgo.It("Verify sample NGINX metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return metricsContainLabels("nginx_ingress_controller_success", "controller_namespace", "ingress-nginx", "app_kubernetes_io_instance", "ingress-controller")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Container Advisor", func() {
				ginkgo.It("Verify sample Container Advisor metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return compMetricsExistInCluster("container_start_time_seconds")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Node Exporter", func() {
				ginkgo.It("Verify sample Node Exporter metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return metricsContainLabels("go_gc_duration_seconds", "job", "node-exporter", "quantile", "0")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Istio mesh and istiod", func() {
				ginkgo.It("Verify sample mesh metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return compMetricsExistInCluster("istio_tcp_connections_opened_total")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
				ginkgo.It("Verify sample istiod metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return metricsContainLabels("sidecar_injection_requests_total", "app", "istiod", "app", "istiod")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Prometheus", func() {
				ginkgo.It("Verify sample metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return metricsContainLabels("prometheus_target_interval_length_seconds", "job", "prometheus", "quantile", "0.99")
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})
		}
	})

	// Query Prometheus for the envoys server statistics
	var _ = ginkgo.Describe("Verify Envoy stats", func() {
		metricName := "envoy_server_stats_recent_lookups"
		if !isManagedClusterProfile {
			ginkgo.It("Verify sample metrics from job envoy-stats", func() {
				gomega.Eventually(func() bool {
					return compMetricsExistInCluster(metricName)
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
			ginkgo.It("Verify count of metrics for job envoy-stats", func() {
				gomega.Eventually(func() bool {
					return verifyEnvoyStats("envoy_server_stats_recent_lookups")
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
		}
	})
})

func compMetricsExistInCluster(metricName string) bool {
	clusterCount, err := strconv.Atoi(totalClusters)
	if err != nil {
		return false
	}
	for i := 1; i < clusterCount; i++ {
		managedCluster := managedPrefix + strconv.Itoa(i)
		gomega.Expect(pkg.MetricsExistInCluster(metricName, "managed_cluster", managedCluster, adminKubeconfig)).To(gomega.BeTrue())
	}
	return true
}

func metricsContainLabels(metricName string, key1, value1, key2, value2 string) bool {
	compMetrics, err := pkg.QueryMetric(metricName, adminKubeconfig)
	if err != nil {
		return false
	}
	metricsCount := 0
	metricsCountMC := 0
	clusterCount, err := strconv.Atoi(totalClusters)
	metrics := pkg.JTq(compMetrics, "data", "result").([]interface{})
	if metrics != nil {
		for _, metric := range metrics {
			if pkg.Jq(metric, "metric", key1) == value1 {
				if pkg.Jq(metric, "metric", key2) == value2 {
					for i := 1; i < clusterCount; i++ {
						managedCluster := managedPrefix + strconv.Itoa(i)
						if pkg.Jq(metric, "metric", "managed_cluster") == managedCluster {
							metricsCountMC++
						}
					}
					metricsCount++
				}
			}
		}
	}
	gomega.Expect(metricsCount == clusterCount).To(gomega.BeTrue())
	gomega.Expect(metricsCountMC == clusterCount - 1).To(gomega.BeTrue())
	return true
}

func verifyEnvoyStats(metricName string) bool {
	envoyStatsMetric, err := pkg.QueryMetric(metricName, adminKubeconfig)
	if err != nil {
		return false
	}

	clusterCount, err := strconv.Atoi(totalClusters)
	if err != nil {
		return false
	}

	metrics := pkg.JTq(envoyStatsMetric, "data", "result").([]interface{})
	if metrics != nil {
		for i := 1; i < clusterCount; i++ {
			managedCluster := managedPrefix + strconv.Itoa(i)
			var countVzSystem int = 0
			var countIstioSystem int = 0
			var countIngressNginx int = 0
			for _, metric := range metrics {
				if pkg.Jq(metric, "metric", "managed_cluster") == managedCluster {
					switch ns := pkg.Jq(metric, "metric", "namespace"); ns {
					case "verrazzano-system":
						countVzSystem++
					case "istio-system":
						countIstioSystem++
					case "ingress-nginx":
						countIngressNginx++
					}
				}
			}
			// metric for pod istio-ingressgateway and istio-egressgateway
			gomega.Expect(countIstioSystem == 2).To(gomega.BeTrue())

			// metric for pd ingress-controller-ingress-nginx-controller and ingress-controller-ingress-nginx-defaultbackend
			gomega.Expect(countIngressNginx == 2).To(gomega.BeTrue())

			// metric for pod verrazzano-api, weblogic-operator, vmi-system-prometheus, fluentd
			gomega.Expect(countVzSystem == 4).To(gomega.BeTrue())
		}
		return true
	}
	return false
}