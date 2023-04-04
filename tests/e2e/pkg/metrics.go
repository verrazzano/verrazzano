// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	vaoClient "github.com/verrazzano/verrazzano/application-operator/clientset/versioned"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/pkg/vzcr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MetricsTest struct {
	Source        MetricSource
	DefaultLabels map[string]string
}

// NewMetricsTest returns a metric test object with which to query metrics
// Parameters:
// kubeconfigs 		a list of kubeconfigs from all clusters
// kubeconfigPath 	this is the kubeconfigPath for the cluster we want to search metrics from
// defaultLabels    the default labels will be added to the test metric when the query begins
func NewMetricsTest(kubeconfigs []string, kubeconfigPath string, defaultLabels map[string]string) (MetricsTest, error) {
	mt := MetricsTest{
		DefaultLabels: defaultLabels,
	}

	for _, kc := range kubeconfigs {
		vz, err := GetVerrazzanoInstallResourceInCluster(kc)
		if err != nil {
			return MetricsTest{}, err
		}
		if !vzcr.IsThanosEnabled(vz) {
			source, err := NewPrometheusSource(kubeconfigPath)
			if err != nil {
				return MetricsTest{}, err
			}
			mt.Source = source
			return mt, nil
		}
	}

	source, err := NewThanosSource(kubeconfigPath)
	if err != nil {
		return MetricsTest{}, err
	}
	mt.Source = source
	return mt, nil
}

func (m MetricsTest) QueryMetric(metricName string, labels map[string]string) (string, error) {
	metricsURL := fmt.Sprintf("https://%s/api/v1/query?query=%s", m.Source.GetHost(), metricName)
	metricsURL = m.appendLabels(metricsURL, labels)
	password, err := GetVerrazzanoPasswordInCluster(m.Source.getKubeConfigPath())
	if err != nil {
		return "", err
	}
	resp, err := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", password, m.Source.getKubeConfigPath())
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error retrieving metric %s, status %d", metricName, resp.StatusCode)
	}
	Log(Info, fmt.Sprintf("metric: %s", resp.Body))
	return string(resp.Body), nil
}

func (m MetricsTest) MetricsExist(metricName string, labels map[string]string) bool {
	metric, err := m.QueryMetric(metricName, labels)
	if err != nil {
		return false
	}
	return metric != ""
}

func (m MetricsTest) appendLabels(query string, labels map[string]string) string {
	if len(labels) == 0 && len(m.DefaultLabels) == 0 {
		return query
	}

	var labelStrings []string
	for k, v := range m.DefaultLabels {
		labelStrings = append(labelStrings, fmt.Sprintf(`%s="%s"`, k, v))
	}
	for k, v := range labels {
		labelStrings = append(labelStrings, fmt.Sprintf(`%s="%s"`, k, v))
	}
	return fmt.Sprintf("%s{%s}", query, strings.Join(labelStrings, ","))
}

// QueryMetricWithLabelByHost queries a metric using a label from the given query function, derived from the kubeconfig
func QueryMetricWithLabelByHost(metricsName string, kubeconfigPath string, label string, labelValue string, queryFunc func(string, string) (string, error), host string) (string, error) {
	if len(labelValue) == 0 {
		return queryFunc(metricsName, kubeconfigPath)
	}
	metricsURL := fmt.Sprintf("https://%s/api/v1/query?query=%s{%s=\"%s\"}", host, metricsName,
		label, labelValue)

	password, err := GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return "", err
	}
	resp, err := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", password, kubeconfigPath)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error retrieving metric %s, status %d", metricsName, resp.StatusCode)
	}
	Log(Info, fmt.Sprintf("metric: %s", resp.Body))
	return string(resp.Body), nil
}

// QueryMetricRange queries a metric from the Prometheus host over a specified range, derived from the kubeconfig
//
// Parameters:
// - metricsName The name of the metric being queried
// - start The start time for the range
// - end The end time for the range
// - stepDuration Query resolution step width
func QueryMetricRange(metricsName string, start *time.Time, end *time.Time, stepSeconds int64, kubeconfigPath string) (string, error) {
	start.Unix()
	metricsURL := fmt.Sprintf("https://%s/api/v1/query_range?query=%s&start=%v&end=%v&step=%v", GetPrometheusIngressHost(kubeconfigPath), metricsName,
		start.Unix(), end.Unix(), stepSeconds)
	password, err := GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return "", err
	}
	resp, err := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", password, kubeconfigPath)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error retrieving metric %s, status %d", metricsName, resp.StatusCode)
	}
	Log(Info, fmt.Sprintf("metric: %s", resp.Body))
	return string(resp.Body), nil
}

// QueryMetric queries a metric from the Prometheus host, derived from the kubeconfig
func QueryMetric(metricsName string, kubeconfigPath string) (string, error) {
	return QueryMetricByHost(metricsName, kubeconfigPath, GetPrometheusIngressHost(kubeconfigPath))
}

// QueryThanosMetric queries a metric from the Thanos host, derived from the kubeconfig
func QueryThanosMetric(metricsName string, kubeconfigPath string) (string, error) {
	return QueryMetricByHost(metricsName, kubeconfigPath, GetThanosQueryIngressHost(kubeconfigPath))
}

func QueryMetricByHost(metricsName, kubeconfigPath, host string) (string, error) {
	metricsURL := fmt.Sprintf("https://%s/api/v1/query?query=%s", host, metricsName)
	password, err := GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return "", err
	}
	resp, err := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", password, kubeconfigPath)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error retrieving metric %s, status %d", metricsName, resp.StatusCode)
	}
	Log(Info, fmt.Sprintf("metric: %s", resp.Body))
	return string(resp.Body), nil
}

// GetPrometheusIngressHost gets the host used for ingress to the system Prometheus in the given cluster
func GetPrometheusIngressHost(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingressList, _ := clientset.NetworkingV1().Ingresses("verrazzano-system").List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "vmi-system-prometheus" {
			Log(Info, fmt.Sprintf("Found Ingress %v", ingress.Name))
			return ingress.Spec.Rules[0].Host
		}
	}
	return ""
}

// GetThanosQueryIngressHost gets the host used for ingress to Thanos Query in the given cluster
func GetThanosQueryIngressHost(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingressList, _ := clientset.NetworkingV1().Ingresses("verrazzano-system").List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "thanos-query-frontend" {
			Log(Info, fmt.Sprintf("Found Ingress %v", ingress.Name))
			return ingress.Spec.Rules[0].Host
		}
	}
	return ""
}

// GetQueryStoreIngressHost gets the host used for ingress to Thanos Query Store in the given cluster
func GetQueryStoreIngressHost(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingress, err := clientset.NetworkingV1().Ingresses("verrazzano-system").Get(context.TODO(), "thanos-grpc", metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Ingress thanos-grpc from the cluster: %v", err))
	}
	return ingress.Spec.Rules[0].Host
}

// MetricsExistInCluster validates the availability of a given metric in the given cluster
func MetricsExistInCluster(metricsName string, keyMap map[string]string, kubeconfigPath string) bool {
	metric, err := QueryMetric(metricsName, kubeconfigPath)
	if err != nil {
		return false
	}
	metrics := JTq(metric, "data", "result").([]interface{})
	if metrics != nil {
		return FindMetric(metrics, keyMap)
	}
	return false
}

// ThanosMetricsExistInCluster validates the availability of a given metric in the given cluster in Thanos
func ThanosMetricsExistInCluster(metricsName string, keyMap map[string]string, kubeconfigPath string) bool {
	metric, err := QueryThanosMetric(metricsName, kubeconfigPath)
	if err != nil {
		return false
	}
	metrics := JTq(metric, "data", "result").([]interface{})
	if metrics != nil {
		return FindMetric(metrics, keyMap)
	}
	return false
}

// GetClusterNameMetricLabel returns the label name used for labeling metrics with the Verrazzano cluster
// This is different in pre-1.1 VZ releases versus later releases
func GetClusterNameMetricLabel(kubeconfigPath string) (string, error) {
	isVz11OrGreater, err := IsVerrazzanoMinVersion("1.1.0", kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error checking Verrazzano min version == 1.1: %t", err))
		return "verrazzano_cluster", err // callers can choose to ignore the error
	} else if !isVz11OrGreater {
		Log(Info, "GetClusterNameMetricsLabel: version is less than 1.1.0")
		// versions < 1.1 use the managed_cluster label not the verrazzano_cluster label
		return "managed_cluster", nil
	}
	Log(Info, "GetClusterNameMetricsLabel: version is greater than or equal to 1.1.0")
	return "verrazzano_cluster", nil
}

// JTq queries JSON text with a JSON path
func JTq(jtext string, path ...string) interface{} {
	var j map[string]interface{}
	json.Unmarshal([]byte(jtext), &j)
	return Jq(j, path...)
}

// FindMetric parses a Prometheus response to find a specified set of metric values
func FindMetric(metrics []interface{}, keyMap map[string]string) bool {
	for _, metric := range metrics {

		// allExist only remains true if all metrics are found in a given JSON response
		allExist := true

		for key, value := range keyMap {
			exists := false
			if Jq(metric, "metric", key) == value {
				// exists is true if the specific key-value pair is found in a given JSON response
				exists = true
			}
			allExist = exists && allExist
		}
		if allExist {
			return true
		}
	}
	// if metric is not found, list unhealthy scrape targets if any for debugging
	ListUnhealthyScrapeTargets()
	return false
}

// MetricsExist is identical to MetricsExistInCluster, except that it uses the cluster specified in the environment
func MetricsExist(metricsName, key, value string) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return false
	}

	// map with single key-value pair
	m := make(map[string]string)
	m[key] = value

	return MetricsExistInCluster(metricsName, m, kubeconfigPath)
}

// ThanosMetricsExist is identical to ThanosMetricsExistInCluster, except that it uses the cluster specified in the environment
func ThanosMetricsExist(metricsName, key, value string) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return false
	}

	// map with single key-value pair
	m := make(map[string]string)
	m[key] = value

	return ThanosMetricsExistInCluster(metricsName, m, kubeconfigPath)
}

// ListUnhealthyScrapeTargets lists all the scrape targets that are unhealthy
func ListUnhealthyScrapeTargets() {
	targets, err := ScrapeTargets()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting scrape targets: %v", err))
		return
	}
	for _, target := range targets {
		if Jq(target, "health") != "up" {
			Log(Info, fmt.Sprintf("target: %s is not ready", Jq(target, "scrapeUrl")))
		}
	}
}

// ScrapeTargets queries Prometheus API /api/v1/targets to list scrape targets
func ScrapeTargets() ([]interface{}, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return nil, err
	}

	metricsURL := fmt.Sprintf("https://%s/api/v1/targets", GetPrometheusIngressHost(kubeconfigPath))
	password, err := GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	resp, err := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", password, kubeconfigPath)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error retrieving targets %d", resp.StatusCode)
	}
	// Log(Info, fmt.Sprintf("targets: %s", string(resp.Body)))
	var result map[string]interface{}
	json.Unmarshal(resp.Body, &result)
	activeTargets := Jq(result, "data", "activeTargets").([]interface{})
	return activeTargets, nil
}

// ThanosQueryStores queries Thanos API /api/v1/stores to list query stores
func ThanosQueryStores() ([]interface{}, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return nil, err
	}

	metricsURL := fmt.Sprintf("https://%s/api/v1/stores", GetThanosQueryIngressHost(kubeconfigPath))
	password, err := GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	resp, err := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", password, kubeconfigPath)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error retrieving targets %d", resp.StatusCode)
	}

	var result map[string]interface{}
	err = json.Unmarshal(resp.Body, &result)
	if err != nil {
		return nil, err
	}
	queryStores, ok := Jq(result, "data", "query").([]interface{})
	if !ok {
		return nil, fmt.Errorf("error finding query store in the Thanos store list")
	}
	return queryStores, nil
}

// Jq queries JSON nodes with a JSON path
func Jq(node interface{}, path ...string) interface{} {
	for _, p := range path {
		if node == nil {
			return nil
		}
		var nodeMap, ok = node.(map[string]interface{})
		if ok {
			node = nodeMap[p]
		} else {
			return nil
		}
	}
	return node
}

// DoesMetricsTemplateExist takes a metrics template name and checks if it exists
func DoesMetricsTemplateExist(namespacedName types.NamespacedName) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return false
	}
	config, err := k8sutil.GetKubeConfigGivenPath(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting config from kubeconfig, error: %v", err))
		return false
	}
	client, err := vaoClient.NewForConfig(config)
	if err != nil {
		Log(Error, fmt.Sprintf("Error creating client from config, error: %v", err))
		return false
	}

	metricsTemplateClient := client.AppV1alpha1().MetricsTemplates(namespacedName.Namespace)
	metricsTemplates, err := metricsTemplateClient.List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Could not list metrics templates, error: %v", err))
		return false
	}

	for _, template := range metricsTemplates.Items {
		if template.Name == namespacedName.Name {
			return true
		}
	}
	return false
}

// getPromOperatorClient returns a client for fetching ServiceMonitor resources
func getPromOperatorClient() (client.Client, error) {
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		return nil, err
	}

	scheme := runtime.NewScheme()
	_ = promoperapi.AddToScheme(scheme)

	cli, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return cli, nil
}

// GetAppServiceMonitorName returns the service monitor name used in VZ 1.4+ for the given
// namespace and app name
func GetAppServiceMonitorName(namespace string, appName string, component string) string {
	// For VZ versions starting from 1.4.0, the service monitor name for prometheus is of the format
	// <app_name>_<app_namespace>
	var smName string
	if component == "" {
		smName = fmt.Sprintf("%s-%s", appName, namespace)
	} else {
		smName = fmt.Sprintf("%s-%s-%s", appName, namespace, component)
		if len(smName) > 63 {
			smName = fmt.Sprintf("%s-%s", appName, namespace)
		}
	}
	if len(smName) > 63 {
		smName = smName[:63]
	}
	return smName
}

// GetServiceMonitor returns the ServiceMonitor identified by namespace and name
func GetServiceMonitor(namespace, name string) (*promoperapi.ServiceMonitor, error) {
	cli, err := getPromOperatorClient()
	if err != nil {
		return nil, err
	}

	serviceMonitor := &promoperapi.ServiceMonitor{}
	err = cli.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: name}, serviceMonitor)
	if err != nil {
		return nil, err
	}
	return serviceMonitor, nil
}

// ScrapeTargetsHealthy validates the health of the scrape targets for the given scrapePools
func ScrapeTargetsHealthy(scrapePools []string) (bool, error) {
	targets, err := ScrapeTargets()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting scrape targets: %v", err))
		return false, err
	}
	for _, scrapePool := range scrapePools {
		found := false
		for _, target := range targets {
			targetScrapePool := Jq(target, "scrapePool").(string)
			if strings.Contains(targetScrapePool, scrapePool) {
				found = true
				// If any of the target health is not "up" return false
				health := Jq(target, "health")
				if health != "up" {
					scrapeURL := Jq(target, "scrapeUrl").(string)
					Log(Error, fmt.Sprintf("target with scrapePool %s and scrapeURL %s is not ready with health %s", targetScrapePool, scrapeURL, health))
					return false, fmt.Errorf("target with scrapePool %s and scrapeURL %s is not healthy", targetScrapePool, scrapeURL)
				}
			}
		}
		// If target with scrapePool not found, then return false and error
		if !found {
			Log(Error, fmt.Sprintf("target with scrapePool %s is not found", scrapePool))
			return false, fmt.Errorf("target with scrapePool %s not found", scrapePool)
		}
	}
	return true, nil
}
