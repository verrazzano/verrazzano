// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ISO8601Layout defines the timestamp format
	ISO8601Layout = "2006-01-02T15:04:05.999999999-07:00"
)

// getSystemElasticSearchIngressHost gets the system Elasticsearch Ingress host in the given cluster
func getSystemElasticSearchIngressHost(kubeconfigPath string) string {
	clientset := GetKubernetesClientsetForCluster(kubeconfigPath)
	ingressList, _ := clientset.ExtensionsV1beta1().Ingresses("verrazzano-system").List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "vmi-system-es-ingest" {
			Log(Info, fmt.Sprintf("Found Elasticsearch Ingress %v, host %s", ingress.Name, ingress.Spec.Rules[0].Host))
			return ingress.Spec.Rules[0].Host
		}
	}
	return ""
}

// listSystemElasticSearchIndices lists the system Elasticsearch indices in the given cluster
func listSystemElasticSearchIndices(kubeconfigPath string) []string {
	url := fmt.Sprintf("https://%s/_all", getSystemElasticSearchIngressHost(kubeconfigPath))
	status, body := RetryGetWithBasicAuth(url, "", "verrazzano", GetVerrazzanoPasswordInCluster(kubeconfigPath), kubeconfigPath)
	list := []string{}
	if status != 200 {
		Log(Debug, fmt.Sprintf("Error retrieving Elasticsearch indices: url=%s, status=%d", url, status))
		return list
	}
	Log(Debug, fmt.Sprintf("indices: %s", body))
	var indexMap map[string]interface{}
	json.Unmarshal([]byte(body), &indexMap)
	for name := range indexMap {
		list = append(list, name)
	}
	return list
}

// querySystemElasticSearch searches the Elasticsearch index with the fields in the given cluster
func querySystemElasticSearch(index string, fields map[string]string, kubeconfigPath string) map[string]interface{} {
	query := ""
	for name, value := range fields {
		fieldQuery := fmt.Sprintf("%s:%s", name, value)
		if query == "" {
			query = fieldQuery
		} else {
			query = fmt.Sprintf("%s+AND+%s", query, fieldQuery)
		}
	}
	url := fmt.Sprintf("https://%s/%s/_doc/_search?q=%s", getSystemElasticSearchIngressHost(kubeconfigPath), index, query)
	status, body := RetryGetWithBasicAuth(url, "", "verrazzano", GetVerrazzanoPasswordInCluster(kubeconfigPath), kubeconfigPath)
	var result map[string]interface{}
	if status != 200 {
		Log(Debug, fmt.Sprintf("Error retrieving Elasticsearch query results: url=%s, status=%d", url, status))
		return result
	}
	Log(Debug, fmt.Sprintf("records: %s", body))
	json.Unmarshal([]byte(body), &result)
	return result
}

// LogIndexFound confirms a named index can be found in Elasticsearch in the cluster specified in the environment
func LogIndexFound(indexName string) bool {
	return LogIndexFoundInCluster(indexName, GetKubeConfigPathFromEnv())
}

// LogIndexFoundInCluster confirms a named index can be found in Elasticsearch on the given cluster
func LogIndexFoundInCluster(indexName, kubeconfigPath string) bool {
	Log(Info, fmt.Sprintf("Looking for log index %s in cluster with kubeconfig %s", indexName, kubeconfigPath))
	for _, name := range listSystemElasticSearchIndices(kubeconfigPath) {
		if name == indexName {
			return true
		}
	}
	Log(Error, fmt.Sprintf("Expected to find log index %s", indexName))
	return false
}

// LogRecordFound confirms a recent log record for the index with matching fields can be found
// in the cluster specified in the environment
func LogRecordFound(indexName string, after time.Time, fields map[string]string) bool {
	return LogRecordFoundInCluster(indexName, after, fields, GetKubeConfigPathFromEnv())
}

// LogRecordFoundInCluster confirms a recent log record for the index with matching fields can be found
// in the given cluster
func LogRecordFoundInCluster(indexName string, after time.Time, fields map[string]string, kubeconfigPath string) bool {
	searchResult := querySystemElasticSearch(indexName, fields, kubeconfigPath)
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
