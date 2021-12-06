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
	"os"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"

	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// ISO8601Layout defines the timestamp format
	ISO8601Layout = "2006-01-02T15:04:05.999999999-07:00"
)

func UseExternalElasticsearch() bool {
	return os.Getenv("EXTERNAL_ELASTICSEARCH") == "true"
}

// GetExternalElasticSearchURL gets the external Elasticsearch URL
func GetExternalElasticSearchURL(kubeconfigPath string) string {
	// the equivalent of kubectl get svc quickstart-es-http -o=jsonpath='{.status.loadBalancer.ingress[0].ip}'
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	svc, err := clientset.CoreV1().Services("default").Get(context.TODO(), "quickstart-es-http", metav1.GetOptions{})
	if err != nil {
		Log(Info, fmt.Sprintf("Could not get services quickstart-es-http in sockshop: %v\n", err.Error()))
		return ""
	}
	if svc.Status.LoadBalancer.Ingress != nil && len(svc.Status.LoadBalancer.Ingress) > 0 {
		return fmt.Sprintf("https://%s:9200", svc.Status.LoadBalancer.Ingress[0].IP)
	}
	return ""
}

// GetSystemElasticSearchIngressURL gets the system Elasticsearch Ingress host in the given cluster
func GetSystemElasticSearchIngressURL(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingressList, _ := clientset.NetworkingV1().Ingresses("verrazzano-system").List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "vmi-system-es-ingest" {
			Log(Info, fmt.Sprintf("Found Elasticsearch Ingress %v, host %s", ingress.Name, ingress.Spec.Rules[0].Host))
			return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
		}
	}
	return ""
}

// getElasticSearchURL returns VMI or external ES URL depending on env var EXTERNAL_ELASTICSEARCH
func getElasticSearchURL(kubeconfigPath string) string {
	if UseExternalElasticsearch() {
		return GetExternalElasticSearchURL(kubeconfigPath)
	}
	return GetSystemElasticSearchIngressURL(kubeconfigPath)
}

func getElasticSearchUsernamePassword(kubeconfigPath string) (username, password string, err error) {
	if UseExternalElasticsearch() {
		esSecret, err := GetSecretInCluster("verrazzano-system", "external-es-secret", kubeconfigPath)
		if err != nil {
			Log(Error, fmt.Sprintf("Failed to get external-es-secret secret: %v", err))
			return "", "", err
		}
		return string(esSecret.Data["username"]), string(esSecret.Data["password"]), err
	}
	password, err = GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return "", "", err
	}
	return "verrazzano", password, err
}

// getElasticSearchWithBasicAuth access ES with GET using basic auth, using a given kubeconfig
func getElasticSearchWithBasicAuth(url string, hostHeader string, username string, password string, kubeconfigPath string) (*HTTPResponse, error) {
	retryableClient, err := getElasticSearchClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "GET", "", hostHeader, username, password, nil, retryableClient)
}

// postElasticSearchWithBasicAuth retries POST using basic auth
func postElasticSearchWithBasicAuth(url, body, username, password, kubeconfigPath string) (*HTTPResponse, error) {
	retryableClient, err := getElasticSearchClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "POST", "application/json", "", username, password, strings.NewReader(body), retryableClient)
}

func getElasticSearchClient(kubeconfigPath string) (*retryablehttp.Client, error) {
	var retryableClient *retryablehttp.Client
	var err error
	if UseExternalElasticsearch() {
		caCert, err := getExternalESCACert(kubeconfigPath)
		if err != nil {
			return nil, err
		}
		client, err := getHTTPClientWithCABundle(caCert, kubeconfigPath)
		if err != nil {
			return nil, err
		}
		retryableClient = newRetryableHTTPClient(client)
		if err != nil {
			return nil, err
		}
	} else {
		retryableClient, err = GetVerrazzanoHTTPClient(kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}
	return retryableClient, nil
}

// getExternalESCACert returns the CA cert from external-es-secret in the specified cluster
func getExternalESCACert(kubeconfigPath string) ([]byte, error) {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	certSecret, err := clientset.CoreV1().Secrets("verrazzano-system").Get(context.TODO(), "external-es-secret", metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return certSecret.Data["ca-bundle"], nil
}

// listSystemElasticSearchIndices lists the system Elasticsearch indices in the given cluster
func listSystemElasticSearchIndices(kubeconfigPath string) []string {
	list := []string{}
	url := fmt.Sprintf("%s/_all", getElasticSearchURL(kubeconfigPath))
	username, password, err := getElasticSearchUsernamePassword(kubeconfigPath)
	if err != nil {
		return list
	}
	resp, err := getElasticSearchWithBasicAuth(url, "", username, password, kubeconfigPath)
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
	url := fmt.Sprintf("%s/%s/_doc/_search?q=%s", getElasticSearchURL(kubeconfigPath), index, query)
	username, password, err := getElasticSearchUsernamePassword(kubeconfigPath)
	if err != nil {
		return result
	}
	resp, err := getElasticSearchWithBasicAuth(url, "", username, password, kubeconfigPath)
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

const numberOfErrorsToLog = 5

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
		if i < numberOfErrorsToLog {
			Log(Error, fmt.Sprintf("Found unexpected log record: %v", hit))
		} else {
			break
		}
	}
	return false
}

var elasticQueryTemplate *template.Template

// SearchLog search recent log records for the index with matching filters.
func SearchLog(index string, query ElasticQuery) map[string]interface{} {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return nil
	}
	if elasticQueryTemplate == nil {
		temp, err := template.New("esQueryTemplate").Parse(queryTemplate)
		if err != nil {
			Log(Error, fmt.Sprintf("Error: %v", err))
		}
		elasticQueryTemplate = temp
	}
	var buffer bytes.Buffer
	err = elasticQueryTemplate.Execute(&buffer, query)
	if err != nil {
		Log(Error, fmt.Sprintf("Error: %v", err))
	}
	configPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error retrieving kubeconfig, error=%v", err))
		return nil
	}
	var result map[string]interface{}
	url := fmt.Sprintf("%s/%s/_search", getElasticSearchURL(kubeconfigPath), index)
	username, password, err := getElasticSearchUsernamePassword(configPath)
	if err != nil {
		return result
	}
	Log(Debug, fmt.Sprintf("Search: %v \nQuery: \n%v", url, buffer.String()))
	resp, err := postElasticSearchWithBasicAuth(url, buffer.String(), username, password, configPath)
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

// POST the request entity body to Elasticsearch API path
func PostElasticsearch(path string, body string) (string, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return "", err
	}
	url := fmt.Sprintf("%s/%s", getElasticSearchURL(kubeconfigPath), path)
	configPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error retrieving kubeconfig, error=%v", err))
		return "", err
	}
	username, password, err := getElasticSearchUsernamePassword(configPath)
	if err != nil {
		return "", err
	}
	Log(Debug, fmt.Sprintf("REST API path: %v \nQuery: \n%v", url, body))
	resp, err := postElasticSearchWithBasicAuth(url, body, username, password, configPath)
	return string(resp.Body), err
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
