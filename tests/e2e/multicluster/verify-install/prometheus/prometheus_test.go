// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mc_prometheus_test

import (
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"os"
	"strconv"
	"strings"
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
var kubeConfigDir = os.Getenv("KUBECONFIG_DIR")

var envoyStatsNamespaces = []string{
	"ingress-nginx",
	"istio-system",
	"verrazzano-system",
}

var excludePodsVS = []string{
	"coherence-operator",
	"oam-kubernetes-runtime",
	"verrazzano-application-operator",
	"verrazzano-monitoring-operator",
	"verrazzano-operator",
	"istiocoredns",
	"istiod",
}

var excludePodsIstio = []string{
	"istiocoredns",
	"istiod",
}

var _ = ginkgo.Describe("Prometheus", func() {
	// Query Prometheus for the sample metrics from the default scraping jobs
	var _ = ginkgo.Describe("Verify default component metrics", func() {
		ginkgo.Context("Verify metrics from NGINX ingress controller", func() {
			ginkgo.It("Verify sample NGINX metrics can be queried from Prometheus", func() {
				gomega.Eventually(func() bool {
					return metricsContainLabels("nginx_ingress_controller_success", "controller_namespace",
						"ingress-nginx", "app_kubernetes_io_instance", "ingress-controller")
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
		})

		ginkgo.Context("Verify metrics from Container Advisor", func() {
			ginkgo.It("Verify sample Container Advisor metrics can be queried from Prometheus", func() {
				gomega.Eventually(func() bool {
					return metricsExistInCluster("container_start_time_seconds")
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
					return metricsExistInCluster("istio_tcp_connections_opened_total")
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
			ginkgo.It("Verify sample istiod metrics can be queried from Prometheus", func() {
				gomega.Eventually(func() bool {
					return metricsContainLabels("sidecar_injection_requests_total", "app", "istiod", "job", "pilot")
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
	})

	// Query Prometheus for the envoys server statistics
	var _ = ginkgo.Describe("Verify Envoy stats", func() {
		metricName := "envoy_server_stats_recent_lookups"
		ginkgo.It("Verify sample metrics from job envoy-stats", func() {
			gomega.Eventually(func() bool {
				return metricsExistInCluster(metricName)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
		ginkgo.It("Verify count of metrics for job envoy-stats", func() {
			gomega.Eventually(func() bool {
				return verifyEnvoyStats("envoy_server_stats_recent_lookups")
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
	})
})

// Validate the given metric exists in all the managed clusters
func metricsExistInCluster(metricName string) bool {
	clusterCount, _ := strconv.Atoi(totalClusters)
	for i := 1; i < clusterCount; i++ {
		managedCluster := managedPrefix + strconv.Itoa(i)
		gomega.Expect(pkg.MetricsExistInCluster(metricName, "managed_cluster", managedCluster, adminKubeconfig)).To(gomega.BeTrue())
	}
	return true
}

// Validate the metrics contain labels
func metricsContainLabels(metricName string, key1, value1, key2, value2 string) bool {
	compMetrics, err := pkg.QueryMetric(metricName, adminKubeconfig)
	if err != nil {
		return false
	}
	metricsCount := 0
	metricsCountMC := 0
	clusterCount, _ := strconv.Atoi(totalClusters)
	metrics := pkg.JTq(compMetrics, "data", "result").([]interface{})
	for _, metric := range metrics {
		if pkg.Jq(metric, "metric", key1) == value1 && pkg.Jq(metric, "metric", key2) == value2 {
			for i := 1; i < clusterCount; i++ {
				managedCluster := managedPrefix + strconv.Itoa(i)
				if pkg.Jq(metric, "metric", "managed_cluster") == managedCluster {
					metricsCountMC++
				}
			}
			metricsCount++
		}
	}
	if metricsCount != clusterCount {
		return false
	}
	if metricsCountMC != clusterCount-1 {
		return false
	}
	return true
}

// Validate the Istio envoy stats
func verifyEnvoyStats(metricName string) bool {
	envoyStatsMetric, err := pkg.QueryMetric(metricName, adminKubeconfig)
	if err != nil {
		return false
	}

	clusterCount, _ := strconv.Atoi(totalClusters)
	for i := 2; i <= clusterCount; i++ {
		managedCluster := managedPrefix + strconv.Itoa(i-1)
		managedKubeConfig := kubeConfigDir + "/" + strconv.Itoa(i) + "/kube_config"
		clientset := pkg.GetKubernetesClientsetForCluster(managedKubeConfig)
		for _, ns := range envoyStatsNamespaces {
			pods := pkg.ListPodsInCluster(ns, clientset)
			for _, pod := range pods.Items {
				var retValue bool
				switch ns {
				case "istio-system":
					if excludePods(pod.Name, excludePodsIstio) {
						retValue = true
						break
					} else {
						retValue = verifyLabelsEnvoyStats(envoyStatsMetric, ns, pod.Name, managedCluster)
					}
				case "verrazzano-system":
					if excludePods(pod.Name, excludePodsVS) {
						retValue = true
						break
					} else {
						retValue = verifyLabelsEnvoyStats(envoyStatsMetric, ns, pod.Name, managedCluster)
					}
				case "ingress-nginx":
					retValue = verifyLabelsEnvoyStats(envoyStatsMetric, ns, pod.Name, managedCluster)
				}
				if !retValue {
					return false
				}
			}
		}
	}
	return true
}

// Assert the existence of labels for namespace and pod in the envoyStatsMetric
func verifyLabelsEnvoyStats(envoyStatsMetric string, namespace string, podName string, managedCluster string) bool {
	metrics := pkg.JTq(envoyStatsMetric, "data", "result").([]interface{})
	for _, metric := range metrics {
		if pkg.Jq(metric, "metric", "namespace") == namespace && pkg.Jq(metric, "metric", "pod_name") == podName &&
			pkg.Jq(metric, "metric", "managed_cluster") == managedCluster {
			return true
		}
	}
	return false
}

// Exclude the pods where envoy stats are not available
func excludePods(podName string, excludeList []string) bool {
	for _, excludes := range excludeList {
		if strings.HasPrefix(podName, excludes) {
			return true
		}
	}
	return false
}
