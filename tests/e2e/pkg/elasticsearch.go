// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/verrazzano/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ISO8601Layout defines the timestamp format
	ISO8601Layout = "2006-01-02T15:04:05.999999999-07:00"
)

// getSystemElasticSearchIngressHost gets the system Elasticsearch Ingress host in the given cluster
func getSystemElasticSearchIngressHost(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
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
	list := []string{}
	url := fmt.Sprintf("https://%s/_all", getSystemElasticSearchIngressHost(kubeconfigPath))
	resp, err := GetWebPageWithBasicAuth(url, "", "verrazzano", GetVerrazzanoPasswordInCluster(kubeconfigPath), kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting Elasticsearch indices: url=%s, error=%v", url, err))
		return list
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Error retrieving Elasticsearch indices: url=%s, status=%d", url, resp.StatusCode))
		return list
	}
	Log(Debug, fmt.Sprintf("indices: %s", resp.Body))
	var indexMap map[string]interface{}
	json.Unmarshal(resp.Body, &indexMap)
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
	var result map[string]interface{}
	url := fmt.Sprintf("https://%s/%s/_doc/_search?q=%s", getSystemElasticSearchIngressHost(kubeconfigPath), index, query)
	resp, err := GetWebPageWithBasicAuth(url, "", "verrazzano", GetVerrazzanoPasswordInCluster(kubeconfigPath), kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error retrieving Elasticsearch query results: url=%s, error=%v", url, err))
		return result
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Error retrieving Elasticsearch query results: url=%s, status=%d", url, resp.StatusCode))
		return result
	}
	Log(Debug, fmt.Sprintf("records: %s", resp.Body))
	json.Unmarshal(resp.Body, &result)
	return result
}

// LogIndexFound confirms a named index can be found in Elasticsearch in the cluster specified in the environment
func LogIndexFound(indexName string) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return false
	}

	return LogIndexFoundInCluster(indexName, kubeconfigPath)
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
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return false
	}

	return LogRecordFoundInCluster(indexName, after, fields, kubeconfigPath)
}

// LogRecordFoundInCluster confirms a recent log record for the index with matching fields can be found
// in the given cluster
func LogRecordFoundInCluster(indexName string, after time.Time, fields map[string]string, kubeconfigPath string) bool {
	searchResult := querySystemElasticSearch(indexName, fields, kubeconfigPath)
	if len(searchResult) == 0 {
		Log(Info, fmt.Sprintf("Expected to find log record matching fields %v", fields))
		return false
	}
	found := findHits(searchResult, &after)
	if !found {
		Log(Error, fmt.Sprintf("Failed to find recent log record for index %s", indexName))
	}
	return found
}

func findHits(searchResult map[string]interface{}, after *time.Time) bool {
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
	if after == nil {
		return true
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
		if t.After(*after) {
			Log(Info, fmt.Sprintf("Found recent record: %s", timestamp))
			return true
		}
		Log(Info, fmt.Sprintf("Found old record: %s", timestamp))
	}
	return false
}

// FindLog returns true if a recent log record can be found in the index with matching filters.
func FindLog(index string, match []Match, mustNot []Match) bool {
	after := time.Now().Add(-24 * time.Hour)
	query := ElasticQuery{
		Filters: match,
		MustNot: mustNot,
	}
	result := SearchLog(index, query)
	found := findHits(result, &after)
	if !found {
		Log(Error, fmt.Sprintf("Failed to find recent log record for index %s", index))
	}
	return found
}

// FindAnyLog returns true if a log record of any time can be found in the index with matching filters.
func FindAnyLog(index string, match []Match, mustNot []Match) bool {
	query := ElasticQuery{
		Filters: match,
		MustNot: mustNot,
	}
	result := SearchLog(index, query)
	found := findHits(result, nil)
	if !found {
		Log(Error, fmt.Sprintf("Failed to find recent log record for index %s", index))
	}
	return found
}

// NoLog returns true if no matched log record can be found in the index.
func NoLog(index string, match []Match, mustNot []Match) bool {
	query := ElasticQuery{
		Filters: match,
		MustNot: mustNot,
	}
	result := SearchLog(index, query)
	if len(result) == 0 {
		return true
	}
	hits := Jq(result, "hits", "hits")
	if hits == nil || len(hits.([]interface{})) == 0 {
		return true
	}
	Log(Error, fmt.Sprintf("Found unexpected %d records", len(hits.([]interface{}))))
	for i, hit := range hits.([]interface{}) {
		if i < 10 {
			Log(Error, fmt.Sprintf("Found unexpected log record: %v", hit))
		}
	}
	return false
}

var systemElasticHost string
var elasticQueryTemplate *template.Template

func systemES() string {
	if systemElasticHost == "" {
		kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
		if err != nil {
			Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
			return ""
		}
		systemElasticHost = getSystemElasticSearchIngressHost(kubeconfigPath)
	}
	return systemElasticHost
}

// SearchLog search recent log records for the index with matching filters.
func SearchLog(index string, query ElasticQuery) map[string]interface{} {
	url := fmt.Sprintf("https://%s/%s/_search", systemES(), index)
	if elasticQueryTemplate == nil {
		temp, err := template.New("esQueryTemplate").Parse(queryTemplate)
		if err != nil {
			Log(Error, fmt.Sprintf("Error: %v", err))
		}
		elasticQueryTemplate = temp
	}
	var buffer bytes.Buffer
	err := elasticQueryTemplate.Execute(&buffer, query)
	if err != nil {
		Log(Error, fmt.Sprintf("Error: %v", err))
	}
	Log(Debug, fmt.Sprintf("Search: %v \nQuery: \n%v", url, buffer.String()))
	configPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error retrieving kubeconfig, error=%v", err))
		return nil
	}

	var result map[string]interface{}
	resp, err := PostWithBasicAuth(url, buffer.String(), "verrazzano", GetVerrazzanoPasswordInCluster(configPath), configPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error retrieving Elasticsearch query results: url=%s, error=%s", url, err))
		return result
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Error retrieving Elasticsearch query results: url=%s, status=%d", url, resp.StatusCode))
		return result
	}
	json.Unmarshal(resp.Body, &result)
	return result
}

// ElasticQuery describes an Elasticsearch Query
type ElasticQuery struct {
	Filters []Match
	MustNot []Match
}

// Match describes a match_phrase in Elasticsearch Query
type Match struct {
	Key   string
	Value string
}

const queryTemplate = `{
  "size": 100,
  "sort": [
    {
      "@timestamp": {
        "order": "desc",
        "unmapped_type": "boolean"
      }
    }
  ],
  "query": {
    "bool": {
      "filter": [
        {
          "match_all": {}
        }
{{range $filter := .Filters}} 
		,
        {
          "match_phrase": {
            "{{$filter.Key}}": "{{$filter.Value}}"
          }
        }
{{end}}
      ],
      "must_not": [
{{range $index, $mustNot := .MustNot}} 
        {{if $index}},{{end}}
        {
          "match_phrase": {
            "{{$mustNot.Key}}": "{{$mustNot.Value}}"
          }
        }
{{end}}
      ]
    }
  }
}
`
