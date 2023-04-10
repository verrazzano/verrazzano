// Copyright (c) 2021, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	promoperapi "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
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
	result, err := m.QueryMetric(metricName, labels)
	if err != nil {
		return false
	}

	metricList, ok := JTq(result, "data", "result").([]interface{})
	if !ok {
		Log(Error, "error extracting metric result, format is not a list type")
	}
	return ok && len(metricList) > 0
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

// GetPrometheusIngressHost gets the host used for ingress to the system Prometheus in the given cluster
func GetPrometheusIngressHost(kubeconfigPath string) string {
	source, err := NewPrometheusSource(kubeconfigPath)
	if err != nil {
		return ""
	}
	return source.GetHost()
}

// GetThanosQueryIngressHost gets the host used for ingress to Thanos Query in the given cluster
func GetThanosQueryIngressHost(kubeconfigPath string) string {
	source, err := NewThanosSource(kubeconfigPath)
	if err != nil {
		return ""
	}
	return source.GetHost()
}

// GetQueryStoreIngressHost gets the host used for ingress to Thanos Query Store in the given cluster
func GetQueryStoreIngressHost(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingress, err := clientset.NetworkingV1().Ingresses("verrazzano-system").Get(context.TODO(), "thanos-query-store", metav1.GetOptions{})
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get Ingress thanos-query-store from the cluster: %v", err))
	}
	return ingress.Spec.Rules[0].Host
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
	var result map[string]interface{}
	if err = json.Unmarshal(resp.Body, &result); err != nil {
		return nil, err
	}
	activeTargets := Jq(result, "data", "activeTargets").([]interface{})
	return activeTargets, nil
}

func ScrapeTargetsFromExec() ([]interface{}, error) {
	metricsURL := "http://localhost:9090/api/v1/targets"
	cmd := exec.Command("kubectl", "exec", "prometheus-prometheus-operator-kube-p-prometheus-0", "-n", "verrazzano-monitoring", "--", "curl", metricsURL)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if len(string(out)) == 0 {
		return nil, fmt.Errorf("prometheus scrape targets request returned no data")
	}
	var data map[string]interface{}
	if err = json.Unmarshal(out, &data); err != nil {
		return nil, err
	}
	activeTargets := Jq(data, "data", "activeTargets").([]interface{})
	return activeTargets, nil
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
	return verifyScrapePoolsHealthy(targets, scrapePools)
}

// ScrapeTargetsHealthyFromExec validates the health of the scrape targets for the given scrapePools by execing into the prometheus pod
func ScrapeTargetsHealthyFromExec(scrapePools []string) (bool, error) {
	targets, err := ScrapeTargetsFromExec()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting scrape targets: %v", err))
		return false, err
	}
	return verifyScrapePoolsHealthy(targets, scrapePools)
}

// verifyScrapePoolsHealthy iterates through the scrape pools and makes sure that it is present in the scrape targets
func verifyScrapePoolsHealthy(scrapeTargets []interface{}, scrapePools []string) (bool, error) {
	for _, scrapePool := range scrapePools {
		found := false
		for _, target := range scrapeTargets {
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
