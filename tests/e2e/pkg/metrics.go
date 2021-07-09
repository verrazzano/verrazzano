// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QueryMetric queries a metric from the Prometheus host, derived from the kubeconfig
func QueryMetric(metricsName string, kubeconfigPath string) (string, error) {
	metricsURL := fmt.Sprintf("https://%s/api/v1/query?query=%s", getPrometheusIngressHost(kubeconfigPath), metricsName)
	resp, err := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", GetVerrazzanoPasswordInCluster(kubeconfigPath), kubeconfigPath)
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
	clientset := GetKubernetesClientsetForCluster(kubeconfigPath)
	ingressList, _ := clientset.ExtensionsV1beta1().Ingresses("verrazzano-system").List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "vmi-system-prometheus" {
			Log(Info, fmt.Sprintf("Found Ingress %v", ingress.Name))
			return ingress.Spec.Rules[0].Host
		}
	}
	return ""
}
