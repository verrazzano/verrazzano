// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	vaoClient "github.com/verrazzano/verrazzano/application-operator/clients/app/clientset/versioned"
	"k8s.io/apimachinery/pkg/types"
	"net/http"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QueryMetricWithLabel queries a metric using a label from the Prometheus host, derived from the kubeconfig
func QueryMetricWithLabel(metricsName string, kubeconfigPath string, label string, labelValue string) (string, error) {
	if len(labelValue) == 0 {
		return QueryMetric(metricsName, kubeconfigPath)
	}
	metricsURL := fmt.Sprintf("https://%s/api/v1/query?query=%s{%s=\"%s\"}", getPrometheusIngressHost(kubeconfigPath), metricsName,
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

// QueryMetric queries a metric from the Prometheus host, derived from the kubeconfig
func QueryMetric(metricsName string, kubeconfigPath string) (string, error) {
	metricsURL := fmt.Sprintf("https://%s/api/v1/query?query=%s", getPrometheusIngressHost(kubeconfigPath), metricsName)
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

// getPrometheusIngressHost gest the host used for ingress to the system Prometheus in the given cluster
func getPrometheusIngressHost(kubeconfigPath string) string {
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

// MetricsExist validates the availability of a given metric in the given cluster
func MetricsExistInCluster(metricsName string, keyMap map[string]string, kubeconfigPath string) bool {
	metric, err := QueryMetric(metricsName, kubeconfigPath)
	if err != nil {
		return false
	}
	metrics := JTq(metric, "data", "result").([]interface{})
	if metrics != nil {
		return findMetric(metrics, keyMap)
	}
	return false
}

// GetClusterNameMetricLabel returns the label name used for labeling metrics with the Verrazzano cluster
// This is different in pre-1.1 VZ releases versus later releases
func GetClusterNameMetricLabel() (string, error) {
	isVz11OrGreater, err := IsVerrazzanoMinVersion("1.1.0")
	if err != nil {
		Log(Error, fmt.Sprintf("Error checking Verrazzano min version == 1.1: %t", err))
		return "verrazzano_cluster", err //callers can choose to ignore the error
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

// findMetric parses a Prometheus response to find a specified set of metric values
func findMetric(metrics []interface{}, keyMap map[string]string) bool {
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

func DoesMetricsTemplateExist(namespacedName types.NamespacedName) (bool) {
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
