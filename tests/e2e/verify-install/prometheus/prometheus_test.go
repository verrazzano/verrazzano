// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package prometheus_test

import (
	"github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
)

const (
	longPollingInterval = 20 * time.Second
	longWaitTimeout     = 10 * time.Minute

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
	keycloakNamespace         = "keycloak"

	// Constants for various metric labels
	nodeExporter        = "node-exporter"
	istiod              = "istiod"
	prometheus          = "prometheus"
	controllerNamespace = "controller_namespace"
	job                 = "job"
	envoyStats          = "envoy-stats"
)

// List of namespaces considered for validating the envoy-stats
var envoyStatsNamespaces = []string{
	ingressNginxNamespace,
	istioSystemNamespace,
	keycloakNamespace,
	verrazzanoSystemNamespace,
}

// List of pods to be excluded for envoy-stats in verrazzano-system namespace
var excludePodsVS = []string{
	"coherence-operator",
	"oam-kubernetes-runtime",
	"verrazzano-application-operator",
	"verrazzano-monitoring-operator",
	"verrazzano-operator",
}

// List of pods to be excluded for envoy-stats in istio-system namespace
var excludePodsIstio = []string{
	"istiocoredns",
	"istiod",
}

var _ = ginkgo.Describe("Prometheus", func() {
	isManagedClusterProfile := pkg.IsManagedClusterProfile()
	// Query Prometheus for the sample metrics from the default scraping jobs
	var _ = ginkgo.Describe("Verify default component metrics", func() {
		if !isManagedClusterProfile {
			ginkgo.Context("Verify metrics from NGINX ingress controller", func() {
				ginkgo.It("Verify sample NGINX metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist(ingressControllerSuccess, controllerNamespace, ingressNginxNamespace)
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Container Advisor", func() {
				ginkgo.It("Verify sample Container Advisor metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist(containerStartTimeSeconds, "namespace", verrazzanoSystemNamespace)
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Node Exporter", func() {
				ginkgo.It("Verify sample Node Exporter metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist(gcDurationSeconds, job, nodeExporter)
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Istio mesh and istiod", func() {
				ginkgo.It("Verify sample mesh metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist(totolTCPConnectionsOpened, "namespace", verrazzanoSystemNamespace)
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
				ginkgo.It("Verify sample istiod metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist(sidecarInjectionRequests, "app", istiod)
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})

			ginkgo.Context("Verify metrics from Prometheus", func() {
				ginkgo.It("Verify sample metrics can be queried from Prometheus", func() {
					gomega.Eventually(func() bool {
						return pkg.MetricsExist(prometheusTargetIntervalLength, job, prometheus)
					}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
				})
			})
		}
	})

	// Query Prometheus for the envoy server statistics
	var _ = ginkgo.Describe("Verify Envoy stats", func() {
		if !isManagedClusterProfile {
			ginkgo.It("Verify sample metrics from job envoy-stats", func() {
				gomega.Eventually(func() bool {
					return pkg.MetricsExist(envoyStatsRecentLookups, job, envoyStats)
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
			ginkgo.It("Verify count of metrics for job envoy-stats", func() {
				gomega.Eventually(func() bool {
					return verifyEnvoyStats(envoyStatsRecentLookups)
				}, longWaitTimeout, longPollingInterval).Should(gomega.BeTrue())
			})
		}
	})
})

// Validate the Istio envoy stats
func verifyEnvoyStats(metricName string) bool {
	envoyStatsMetric, err := pkg.QueryMetric(metricName, pkg.GetKubeConfigPathFromEnv())
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
			default:
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
		if pkg.Jq(metric, "metric", "namespace") == namespace && pkg.Jq(metric, "metric", "pod_name") == podName {
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
