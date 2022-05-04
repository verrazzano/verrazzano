// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package syscomponents

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/verrazzano/verrazzano/pkg/test/framework"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

const (
	longPollingInterval = 8 * time.Second
	longWaitTimeout     = 10 * time.Minute

	// Constants for sample metrics of system components validated by the test
	ingressControllerSuccess       = "nginx_ingress_controller_success"
	containerStartTimeSeconds      = "container_start_time_seconds"
	cpuSecondsTotal                = "node_cpu_seconds_total"
	istioRequestsTotal             = "istio_requests_total"
	sidecarInjectionRequests       = "sidecar_injection_requests_total"
	prometheusTargetIntervalLength = "prometheus_target_interval_length_seconds"
	envoyStatsRecentLookups        = "envoy_server_stats_recent_lookups"

	// Namespaces used for validating envoy stats
	verrazzanoSystemNamespace = "verrazzano-system"
	istioSystemNamespace      = "istio-system"
	ingressNginxNamespace     = "ingress-nginx"
	keycloakNamespace         = "keycloak"

	// Constants for various metric labels, used in the validation
	nodeExporter        = "node-exporter"
	istiod              = "istiod"
	prometheus          = "prometheus"
	controllerNamespace = "controller_namespace"
	ingressController   = "ingress-controller"
	appK8SIOInstance    = "app_kubernetes_io_instance"
	job                 = "job"
	app                 = "app"
	namespace           = "namespace"
	pilot               = "pilot"
	podName             = "pod_name"
)

var clusterName = os.Getenv("CLUSTER_NAME")
var kubeConfig = os.Getenv("KUBECONFIG")

// will be initialized in BeforeSuite so that any log messages during init are available
var clusterNameMetricsLabel = ""
var isMinVersion110 bool

var adminKubeConfig string
var isManagedClusterProfile bool

// List of namespaces considered for validating the envoy-stats
var envoyStatsNamespaces = []string{
	ingressNginxNamespace,
	istioSystemNamespace,
	verrazzanoSystemNamespace,
}

// List of pods to be excluded from verrazzano-system namespace for envoy-stats as they do not have envoy
var excludePodsVS = []string{
	"coherence-operator",
	"oam-kubernetes-runtime",
	"verrazzano-application-operator",
	"verrazzano-monitoring-operator",
	"verrazzano-operator",
}

// List of pods to be excluded from istio-system namespace for envoy-stats as they do not have envoy
var excludePodsIstio = []string{
	"istiocoredns",
	"istiod",
}

var t = framework.NewTestFramework("syscomponents")

var _ = t.BeforeSuite(func() {
	present := false
	var err error
	adminKubeConfig, present = os.LookupEnv("ADMIN_KUBECONFIG")
	isManagedClusterProfile = pkg.IsManagedClusterProfile()
	if isManagedClusterProfile {
		if !present {
			Fail(fmt.Sprintln("Environment variable ADMIN_KUBECONFIG is required to run the test"))
		}
	} else {
		// Include the namespace keycloak for the validation for admin cluster and single cluster installation
		envoyStatsNamespaces = append(envoyStatsNamespaces, keycloakNamespace)
		adminKubeConfig, err = k8sutil.GetKubeConfigLocation()
		if err != nil {
			Fail(err.Error())
		}
	}

	isMinVersion110, err = pkg.IsVerrazzanoMinVersion("1.1.0", adminKubeConfig)
	if err != nil {
		Fail(err.Error())
	}
})

var _ = t.AfterSuite(func() {})

var _ = t.AfterEach(func() {})

var _ = t.Describe("Prometheus Metrics", Label("f:observability.monitoring.prom"), func() {
	// Query Prometheus for the sample metrics from the default scraping jobs
	var _ = t.Describe("for the system components", func() {
		t.It("Verify sample NGINX metrics can be queried from Prometheus", func() {
			Eventually(func() bool {
				kv := map[string]string{
					controllerNamespace: ingressNginxNamespace,
					appK8SIOInstance:    ingressController,
				}
				return metricsContainLabels(ingressControllerSuccess, kv)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})

		t.It("Verify sample Container Advisor metrics can be queried from Prometheus", func() {
			Eventually(func() bool {
				return metricsContainLabels(containerStartTimeSeconds, map[string]string{})
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})

		t.It("Verify sample Node Exporter metrics can be queried from Prometheus", func() {
			Eventually(func() bool {
				kv := map[string]string{
					job: nodeExporter,
				}
				return metricsContainLabels(cpuSecondsTotal, kv)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})

		if istioInjection == "enabled" {
			t.It("Verify sample mesh metrics can be queried from Prometheus", func() {
				Eventually(func() bool {
					kv := map[string]string{
						namespace: verrazzanoSystemNamespace,
					}
					return metricsContainLabels(istioRequestsTotal, kv)
				}, longWaitTimeout, longPollingInterval).Should(BeTrue())
			})

			t.It("Verify sample istiod metrics can be queried from Prometheus", func() {
				Eventually(func() bool {
					kv := map[string]string{
						app: istiod,
						job: pilot,
					}
					return metricsContainLabels(sidecarInjectionRequests, kv)
				}, longWaitTimeout, longPollingInterval).Should(BeTrue())
			})
		}

		t.It("Verify sample metrics can be queried from Prometheus", func() {
			Eventually(func() bool {
				kv := map[string]string{
					job: prometheus,
				}
				return metricsContainLabels(prometheusTargetIntervalLength, kv)
			}, longWaitTimeout, longPollingInterval).Should(BeTrue())
		})

		if istioInjection == "enabled" {
			t.It("Verify envoy stats", func() {
				Eventually(func() bool {
					return verifyEnvoyStats(envoyStatsRecentLookups)
				}, longWaitTimeout, longPollingInterval).Should(BeTrue())
			})
		}
	})
})

// Validate the Istio envoy stats for the pods in the namespaces defined in envoyStatsNamespaces
func verifyEnvoyStats(metricName string) bool {
	envoyStatsMetric, err := pkg.QueryMetricWithLabel(metricName, adminKubeConfig, getClusterNameMetricLabel(), getClusterNameForPromQuery())
	if err != nil {
		return false
	}
	clientset, err := pkg.GetKubernetesClientsetForCluster(kubeConfig)
	if err != nil {
		t.Logs.Errorf("Error getting clienset for %s, error: %v", kubeConfig, err)
		return false
	}
	for _, ns := range envoyStatsNamespaces {
		pods, err := pkg.ListPodsInCluster(ns, clientset)
		if err != nil {
			t.Logs.Errorf("Error listing pods in cluster for namespace: %s, error: %v", namespace, err)
			return false
		}
		for _, pod := range pods.Items {
			var retValue bool
			switch ns {
			case istioSystemNamespace:
				if excludePods(pod.Name, excludePodsIstio) {
					retValue = true
				} else {
					retValue = verifyLabels(envoyStatsMetric, ns, pod.Name)
				}
			case verrazzanoSystemNamespace:
				if excludePods(pod.Name, excludePodsVS) {
					retValue = true
				} else {
					retValue = verifyLabels(envoyStatsMetric, ns, pod.Name)
				}
			default:
				retValue = verifyLabels(envoyStatsMetric, ns, pod.Name)
			}
			if !retValue {
				return false
			}
		}
	}
	return true
}

func getClusterNameMetricLabel() string {
	if clusterNameMetricsLabel == "" {
		// ignore error getting the metric label - we'll just use the default value returned
		lbl, err := pkg.GetClusterNameMetricLabel(adminKubeConfig)
		if err != nil {
			t.Logs.Errorf("Error getting cluster name metric label: %s", err.Error())
		}
		clusterNameMetricsLabel = lbl
	}
	return clusterNameMetricsLabel
}

// Assert the existence of labels for namespace and pod in the envoyStatsMetric
func verifyLabels(envoyStatsMetric string, ns string, pod string) bool {
	metrics := pkg.JTq(envoyStatsMetric, "data", "result").([]interface{})
	for _, metric := range metrics {
		if pkg.Jq(metric, "metric", namespace) == ns && pkg.Jq(metric, "metric", podName) == pod {
			if isManagedClusterProfile {
				// when the admin cluster scrapes the metrics from a managed cluster, as label verrazzano_cluster with value
				// name of the managed cluster is added to the metrics
				if pkg.Jq(metric, "metric", getClusterNameMetricLabel()) == clusterName {
					return true
				}
			} else {
				// the metrics for the admin cluster or in the single cluster installation should contain the label
				// verrazzano_cluster with the value "local" when version 1.1 or higher.
				if isMinVersion110 {
					if pkg.Jq(metric, "metric", getClusterNameMetricLabel()) == "local" {
						return true
					}
				} else {
					if pkg.Jq(metric, "metric", getClusterNameMetricLabel()) == nil {
						return true
					}
				}
			}
		}
	}
	return false
}

// Validate the metrics contain the labels with values specified as key-value pairs of the map
func metricsContainLabels(metricName string, kv map[string]string) bool {
	clusterNameValue := getClusterNameForPromQuery()
	t.Logs.Debugf("Looking for metric name %s with label %s = %s", metricName, getClusterNameMetricLabel(), clusterNameValue)
	compMetrics, err := pkg.QueryMetricWithLabel(metricName, adminKubeConfig, getClusterNameMetricLabel(), clusterNameValue)
	if err != nil {
		return false
	}
	metrics := pkg.JTq(compMetrics, "data", "result").([]interface{})
	for _, metric := range metrics {
		metricFound := true
		for key, value := range kv {
			if pkg.Jq(metric, "metric", key) != value {
				metricFound = false
				break
			}
		}

		if metricFound {
			if isManagedClusterProfile {
				// when the admin cluster scrapes the metrics from a managed cluster, as label verrazzano_cluster with value
				// name of the managed cluster is added to the metrics
				if pkg.Jq(metric, "metric", getClusterNameMetricLabel()) == clusterName {
					return true
				}
			} else {
				// the metrics for the admin cluster or in the single cluster installation should contain the label
				// verrazzano_cluster with the local cluster as its value when version 1.1 or higher
				if isMinVersion110 {
					if pkg.Jq(metric, "metric", getClusterNameMetricLabel()) == "local" {
						return true
					}
				} else {
					if pkg.Jq(metric, "metric", getClusterNameMetricLabel()) == nil {
						return true
					}
				}
			}
		}
	}
	return false
}

// Exclude the pods where envoy stats are not available
func excludePods(pod string, excludeList []string) bool {
	for _, excludes := range excludeList {
		if strings.HasPrefix(pod, excludes) {
			return true
		}
	}
	return false
}

// Return the cluster name used for the Prometheus query
func getClusterNameForPromQuery() string {
	if isManagedClusterProfile {
		return clusterName
	}
	if isMinVersion110 {
		return "local"
	}
	return ""
}
