// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/onsi/ginkgo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ISO8601Layout = "2006-01-02T15:04:05.999999999-07:00"
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
	status, body := GetWebPageWithBasicAuth(url, "", "verrazzano", GetVerrazzanoPassword())
	if status != 200 {
		ginkgo.Fail(fmt.Sprintf("Error retrieving Elasticsearch indices: url=%s, status=%d", url, status))
	}
	Log(Debug, fmt.Sprintf("indices: %s", body))
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
	status, body := GetWebPageWithBasicAuth(url, "", "verrazzano", GetVerrazzanoPassword())
	if status != 200 {
		ginkgo.Fail(fmt.Sprintf("Error retrieving Elasticsearch query results: url=%s, status=%d", url, status))
	}
	Log(Debug, fmt.Sprintf("records: %s", body))
	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)
	return result
}

// LogIndexFound confirms a named index can be found.
func LogIndexFound(indexName string) bool {
	for _, name := range ListSystemElasticSearchIndices() {
		if name == indexName {
			return true
		}
	}
	Log(Error, fmt.Sprintf("Expected to find log index %s", indexName))
	return false
}

// LogRecordFound confirms a recent log record for the index with matching fields can be found.
func LogRecordFound(indexName string, after time.Time, fields map[string]string) bool {
	searchResult := QuerySystemElasticSearch(indexName, fields)
	hits := Jq(searchResult, "hits", "hits")
	if hits == nil {
		Log(Info, "Expected to find hits in log record query results")
		return false
	}
	Log(Info, fmt.Sprintf("Found %d records", len(hits.([]interface{}))))
	if len(hits.([]interface{})) == 0 {
		Log(Info, "Expected log record query results to contain at least one hit")
		return false
	}
	for _, hit := range hits.([]interface{}) {
		timestamp := Jq(hit, "_source", "@timestamp")
		t, err := time.Parse(ISO8601Layout, timestamp.(string))
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, timestamp.(string))
			if err != nil {
				Log(Error, fmt.Sprintf("Failed to parse timestamp: %s", timestamp))
				return false
			}
		}
		if t.After(after) {
			Log(Info, fmt.Sprintf("Found recent record: %s", timestamp))
			return true
		}
		Log(Info, fmt.Sprintf("Found old record: %s", timestamp))
	}
	Log(Error, fmt.Sprintf("Failed to find recent log record for index %s", indexName))
	return false
}
