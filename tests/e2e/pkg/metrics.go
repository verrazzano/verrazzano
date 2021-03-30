// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// QueryMetric queries a metric from the system Prometheus
func QueryMetric(metricsName string) (string, error) {
	metricsURL := fmt.Sprintf("https://%s/api/v1/query?query=%s", getPrometheusIngressHost(), metricsName)
	status, content := GetWebPageWithBasicAuth(metricsURL, "", "verrazzano", GetVerrazzanoPassword())
	if status != 200 {
		Log(Error, fmt.Sprintf("Error retrieving metric %s, status %d", metricsName, status))
		// do not call ginkgo.Fail() here - this method is called in Eventually funcs, so if you call Fail()
		// you essentially limit all of those to only one attempt, which defeats the purpose
		return "", errors.New("no content")
	}
	Log(Info, fmt.Sprintf("metric: %s", content))
	return content, nil
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
