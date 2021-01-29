// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package util

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetSystemElasticSearchIngressHost gets the system Elasticsearch Ingress host
func GetSystemElasticSearchIngressHost() string {
	ingressList, _ := GetKubernetesClientset().ExtensionsV1beta1().Ingresses("verrazzano-system").List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "vmi-system-es-ingest" {
			Log(Info, fmt.Sprintf("Found Elasticsearch Ingress %v, host %s", ingress.Name, ingress.Spec.Rules[0].Host))
			return ingress.Spec.Rules[0].Host
		}
	}
	return ""
}

// ListSystemElasticSearchIndices lists the system Elasticsearch indices
func ListSystemElasticSearchIndices() []string {
	url := fmt.Sprintf("https://%s/_all", GetSystemElasticSearchIngressHost())
	status, body := GetWebPageWithBasicAuth(url, "", "verrazzano", getVerrazzanoPassword())
	if status != 200 {
		ginkgo.Fail(fmt.Sprintf("Error retrieving Elasticsearch indices: url=%s, status=%d", url, status))
	}
	Log(Debug,fmt.Sprintf("indices: %s", body))
	var indexMap map[string]interface{}
	json.Unmarshal([]byte(body), &indexMap)
	list := []string{}
	for name, _ := range indexMap {
		list = append(list, name)
	}
	return list
}

// QuerySystemElasticSearch searches the Elasticsearch index with the fields
func QuerySystemElasticSearch(index string, fields map[string]string) map[string]interface{} {
	query := ""
	for name, value := range fields {
		fieldQuery := fmt.Sprintf("%s:%s", name, value)
		if query == "" {
			query = fieldQuery
		} else {
			query = fmt.Sprintf("%s+AND+%s", query, fieldQuery)
		}
	}
	url := fmt.Sprintf("https://%s/%s/_doc/_search?q=%s", GetSystemElasticSearchIngressHost(), index, query)
	status, body := GetWebPageWithBasicAuth(url, "", "verrazzano", getVerrazzanoPassword())
	if status != 200 {
		ginkgo.Fail(fmt.Sprintf("Error retrieving Elasticsearch query results: url=%s, status=%d", url, status))
	}
	Log(Debug,fmt.Sprintf("records: %s", body))
	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)
	return result
}
