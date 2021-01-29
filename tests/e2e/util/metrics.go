// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"context"
	"fmt"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QueryMetric queries a metric from the system Prometheus
func QueryMetric(metricsName string) string {
	metricsURL := fmt.Sprintf("https://%s/api/v1/query?query=%s", getPrometheusIngressHost(), metricsName)
	status, content := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", getVerrazzanoPassword())
	if status != 200 {
		ginkgo.Fail(fmt.Sprintf("Error retrieving metric %s, status %d", metricsName, status))
	}
	Log(Info,fmt.Sprintf("metric: %s", content))
	return content
}

// getPrometheusIngressHost gest the host used for ingress to the system Prometheus
func getPrometheusIngressHost() string {
	ingressList, _ := GetKubernetesClientset().ExtensionsV1beta1().Ingresses("verrazzano-system").List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "vmi-system-prometheus" {
			Log(Info, fmt.Sprintf("Found Ingress %v", ingress.Name))
			return ingress.Spec.Rules[0].Host
		}
	}
	return ""
}
