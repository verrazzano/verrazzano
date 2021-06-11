// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package mc_prometheus_test

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

	labelManagedCluster = "managed_cluster"

	// Constants for sample metrics from system components
	ingressControllerSuccess       = "nginx_ingress_controller_success"
	containerStartTimeSeconds      = "container_start_time_seconds"
	gcDurationSeconds              = "go_gc_duration_seconds"
	totolTCPConnectionsOpened      = "istio_tcp_connections_opened_total"
	sidecarInjectionRequests       = "sidecar_injection_requests_total"
	prometheusTargetIntervalLength = "prometheus_target_interval_length_seconds"
	envoyStatsRecentLookups        = "envoy_server_stats_recent_lookups"

	// Namespaces used for validating envoy stats
	verrazzanoSystemNamespace = "verrazzano-system"
	istioSystemNamespace      = "istio-system"
	ingressNginxNamespace     = "ingress-nginx"

	// Constants for various metric labels
	ingressController     = "ingress-controller"
	nodeExporter          = "node-exporter"
	istiod                = "istiod"
	prometheus            = "prometheus"
	controllerNamespace   = "controller_namespace"
	appK8SIOInstance      = "app_kubernetes_io_instance"
	job                   = "job"
	app                   = "app"
	quantile              = "quantile"
	pilot                 = "pilot"
	quantileZero          = "0"
	quantilePointNineNine = "0.99"
)

var managedCluster = os.Getenv("MANAGED_CLUSTER_NAME")
var adminKubeconfig = os.Getenv("ADMIN_KUBECONFIG")

// Namespaces used for validating envoy stats
var envoyStatsNamespaces = []string{
	ingressNginxNamespace,
	istioSystemNamespace,
	verrazzanoSystemNamespace,
}

// Pods excluded from verrazzano-system namespace for validation envoy proxy stats
var excludePodsVS = []string{
	"coherence-operator",
	"oam-kubernetes-runtime",
	"verrazzano-application-operator",
	"verrazzano-monitoring-operator",
	"verrazzano-operator",
}

// Pods excluded from istio-system namespace for validation envoy proxy stats
var excludePodsIstio = []string{
	"istiocoredns",
	"istiod",
}

// Validation of Prometheus metrics for system components and envoy proxy for managed clusters
// The validation for the admin cluster in a multi-cluster setup is done by test suite verify-install/prometheus
var _ = ginkgo.Describe("Prometheus", func() {
	// Query Prometheus for the sample metrics from the default scraping jobs
	var _ = ginkgo.Describe("Verify default component metrics", func() {
		ginkgo.Context("Verify metrics from NGINX ingress controller", func() {
			ginkgo.It("Verify sample NGINX metrics can be queried from Prometheus", func() {
				gomega.Eventually(func() bool {
					return metricsContainLabels(ingressControllerSuccess, controllerNamespace,
						ingressNginxNamespace, appK8SIOInstance, ingressController)
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
		})

		ginkgo.Context("Verify metrics from Container Advisor", func() {
			ginkgo.It("Verify sample Container Advisor metrics can be queried from Prometheus", func() {
				gomega.Eventually(func() bool {
					return metricsExistInCluster(containerStartTimeSeconds)
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
		})

		ginkgo.Context("Verify metrics from Node Exporter", func() {
			ginkgo.It("Verify sample Node Exporter metrics can be queried from Prometheus", func() {
				gomega.Eventually(func() bool {
					return metricsContainLabels(gcDurationSeconds, job, nodeExporter, quantile, quantileZero)
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
		})

		ginkgo.Context("Verify metrics from Istio mesh and istiod", func() {
			ginkgo.It("Verify sample mesh metrics can be queried from Prometheus", func() {
				gomega.Eventually(func() bool {
					return metricsExistInCluster(totolTCPConnectionsOpened)
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
			ginkgo.It("Verify sample istiod metrics can be queried from Prometheus", func() {
				gomega.Eventually(func() bool {
					return metricsContainLabels(sidecarInjectionRequests, app, istiod, job, pilot)
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
		})

		ginkgo.Context("Verify metrics from Prometheus", func() {
			ginkgo.It("Verify sample metrics can be queried from Prometheus", func() {
				gomega.Eventually(func() bool {
					return metricsContainLabels(prometheusTargetIntervalLength, job, prometheus, quantile, quantilePointNineNine)
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
		})
	})

	// Query Prometheus for the envoys server statistics
	var _ = ginkgo.Describe("Verify Envoy stats", func() {
		ginkgo.It("Verify sample metrics from job envoy-stats", func() {
			gomega.Eventually(func() bool {
				return metricsExistInCluster(envoyStatsRecentLookups)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
		ginkgo.It("Verify count of metrics for job envoy-stats", func() {
			gomega.Eventually(func() bool {
				return verifyEnvoyStats(envoyStatsRecentLookups)
			}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
		})
	})
})

// Validate the given metric exists in the managed cluster
func metricsExistInCluster(metricName string) bool {
	return gomega.Expect(pkg.MetricsExistInCluster(metricName, labelManagedCluster, managedCluster, adminKubeconfig)).To(gomega.BeTrue())
}

// Validate the metrics contain the given labels in the managed cluster
func metricsContainLabels(metricName string, key1, value1, key2, value2 string) bool {
	compMetrics, err := pkg.QueryMetric(metricName, adminKubeconfig)
	if err != nil {
		return false
	}
	metrics := pkg.JTq(compMetrics, "data", "result").([]interface{})
	for _, metric := range metrics {
		// The metrics scraped from the managed cluster contains a special label managed_cluster
		if pkg.Jq(metric, "metric", key1) == value1 && pkg.Jq(metric, "metric", key2) == value2 &&
			pkg.Jq(metric, "metric", labelManagedCluster) == managedCluster {
			return true
		}
	}
	return false
}

// Validate the Istio envoy stats
func verifyEnvoyStats(metricName string) bool {
	envoyStatsMetric, err := pkg.QueryMetric(metricName, adminKubeconfig)
	if err != nil {
		return false
	}
	clientset := pkg.GetKubernetesClientset()
	for _, ns := range envoyStatsNamespaces {
		pods := pkg.ListPodsInCluster(ns, clientset)
		for _, pod := range pods.Items {
			var retValue bool
			switch ns {
				case istioSystemNamespace:
					if excludePods(pod.Name, excludePodsIstio) {
						retValue = true
						break
					} else {
						retValue = verifyLabelsEnvoyStats(envoyStatsMetric, ns, pod.Name)
					}
				case verrazzanoSystemNamespace:
					if excludePods(pod.Name, excludePodsVS) {
						retValue = true
						break
					} else {
						retValue = verifyLabelsEnvoyStats(envoyStatsMetric, ns, pod.Name)
					}
				case ingressNginxNamespace:
					retValue = verifyLabelsEnvoyStats(envoyStatsMetric, ns, pod.Name)
			}
			if !retValue {
				return false
			}
		}
	}
	return true
}

// Assert the existence of labels for namespace and pod in the envoyStatsMetric
func verifyLabelsEnvoyStats(envoyStatsMetric string, namespace string, podName string) bool {
	metrics := pkg.JTq(envoyStatsMetric, "data", "result").([]interface{})
	for _, metric := range metrics {
		if pkg.Jq(metric, "metric", "namespace") == namespace && pkg.Jq(metric, "metric", "pod_name") == podName &&
			pkg.Jq(metric, "metric", labelManagedCluster) == managedCluster {
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
