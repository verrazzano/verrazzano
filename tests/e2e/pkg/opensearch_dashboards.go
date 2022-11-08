// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"text/template"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IndexPattern struct {
	Name string
}

// ListIndexPatterns gets the configured index patterns in OpenSearch Dashboards
func ListIndexPatterns(kubeconfigPath string) []string {
	list := []string{}
	url := fmt.Sprintf("%s/api/saved_objects/_find?type=index-pattern&fields=title", getOpenSearchDashboardsURL(kubeconfigPath))
	username, password, err := getOpenSearchDashboardsUsernamePassword(kubeconfigPath)
	if err != nil {
		return list
	}
	resp, err := getOpenSearchDashboardsWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting Opensearch indices: url=%s, error=%v", url, err))
		return list
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Error retrieving Opensearch indices: url=%s, status=%d", url, resp.StatusCode))
		return list
	}
	Log(Debug, fmt.Sprintf("indices: %s", resp.Body))
	var responseMap map[string]interface{}
	if err := json.Unmarshal(resp.Body, &responseMap); err != nil {
		Log(Error, fmt.Sprintf("OpenSearch Dashboards: Error unmarshalling index patterns response body: %v", err))
	}
	if responseMap["saved_objects"] != nil {
		savedObjects := reflect.ValueOf(responseMap["saved_objects"])
		for i := 0; i < savedObjects.Len(); i++ {
			Log(Debug, fmt.Sprintf("OpenSearch Dashboards: Index pattern details: %v", savedObjects.Index(i)))
			savedObject := savedObjects.Index(i).Interface().(map[string]interface{})
			attributes := savedObject["attributes"].(map[string]interface{})
			if attributes["title"].(string) != "" {
				list = append(list, attributes["title"].(string))
			}
		}
	}
	return list
}

// LogIndexPatternFound confirms a named index pattern can be found in OpenSearch Dashboards in the cluster specified in the environment
func LogIndexPatternFound(indexName string) bool {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig, error: %v", err))
		return false
	}

	return LogIndexFoundInCluster(indexName, kubeconfigPath)
}

// LogIndexFoundInCluster confirms a named index pattern can be found in OpenSearch Dashboards on the given cluster
func LogIndexPatternFoundInCluster(indexName, kubeconfigPath string) bool {
	Log(Info, fmt.Sprintf("Looking for log index %s in cluster with kubeconfig %s", indexName, kubeconfigPath))
	for _, name := range ListIndexPatterns(kubeconfigPath) {
		if name == indexName {
			return true
		}
	}
	Log(Error, fmt.Sprintf("Expected to find log index %s", indexName))
	return false
}

// CreateIndexPattern creates the specified index pattern in OpenSearch Dashboards
func CreateIndexPattern(pattern string) map[string]interface{} {
	template, err := template.New("indexPatternTemplate").Parse(indexPatternTemplate)
	if err != nil {
		Log(Error, fmt.Sprintf("Error: %v", err))
	}
	var buffer bytes.Buffer
	err = template.Execute(&buffer, IndexPattern{Name: pattern})
	if err != nil {
		Log(Error, fmt.Sprintf("Error: %v", err))
	}
	var result map[string]interface{}
	resp, err := PostOpensearchDashboards("api/saved_objects/index-pattern", buffer.String(), "osd-xsrf:true", "kbn-xsrf: true")
	if err != nil {
		Log(Error, fmt.Sprintf("Error creating index patterns in OpenSearchDashboards: error=%s", err))
		return result
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Error creating index patterns in OpenSearchDashboards: status=%d", resp.StatusCode))
		return result
	}
	json.Unmarshal(resp.Body, &result)
	return result
}

// PostOpensearchDashboards POST the request entity body to Opensearch API path
// The provided path is appended to the OpenSearchDashboards base URL
func PostOpensearchDashboards(path string, body string, additionalHeaders ...string) (*HTTPResponse, error) {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return nil, err
	}
	url := fmt.Sprintf("%s/%s", getOpenSearchDashboardsURL(kubeconfigPath), path)
	configPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf("Error retrieving kubeconfig, error=%v", err))
		return nil, err
	}
	username, password, err := getOpenSearchDashboardsUsernamePassword(configPath)
	if err != nil {
		return nil, err
	}
	Log(Debug, fmt.Sprintf("REST API path: %v \nQuery: \n%v", url, body))
	resp, err := postOpenSearchDashboardsWithBasicAuth(url, body, username, password, configPath, additionalHeaders...)
	return resp, err
}

// getOpenSearchDashboardsURL gets the OpenSearch Dashboards Ingress host in the given cluster
func getOpenSearchDashboardsURL(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	isMinversion150, _ := IsVerrazzanoMinVersionEventually("1.5.0", kubeconfigPath)

	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingressList, _ := clientset.NetworkingV1().Ingresses("verrazzano-system").List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if (isMinversion150 && ingress.Name == "vmi-system-opensearchdashboards") || (!isMinversion150 && ingress.Name == "vmi-system-kibana") {
			Log(Info, fmt.Sprintf("Found Kibana/OpenSearch Dashboards Ingress %v, host %s", ingress.Name, ingress.Spec.Rules[0].Host))
			return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
		}
	}
	return ""
}

// getOpenSearchDashboardsUsernamePassword gets the Verrazzano user name and password
func getOpenSearchDashboardsUsernamePassword(kubeconfigPath string) (username, password string, err error) {
	password, err = GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return "", "", err
	}
	return "verrazzano", password, err
}

// getOpenSearchDashboardsWithBasicAuth access OpenSearch Dashboards with GET using basic auth, using a given kubeconfig
func getOpenSearchDashboardsWithBasicAuth(url string, hostHeader string, username string, password string, kubeconfigPath string) (*HTTPResponse, error) {
	retryableClient, err := getOpenSearchDashboardsClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "GET", "", hostHeader, username, password, nil, retryableClient)
}

// postOpenSearchDashboardsWithBasicAuth retries POST to OpenSearch Dashboards using basic auth
func postOpenSearchDashboardsWithBasicAuth(url, body, username, password, kubeconfigPath string, additionalHeaders ...string) (*HTTPResponse, error) {
	retryableClient, err := getOpenSearchDashboardsClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "POST", "application/json", "", username, password, strings.NewReader(body), retryableClient, additionalHeaders...)
}

func getOpenSearchDashboardsClient(kubeconfigPath string) (*retryablehttp.Client, error) {
	var retryableClient *retryablehttp.Client
	var err error
	retryableClient, err = GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return retryableClient, nil
}

const indexPatternTemplate = `{
      "attributes": {
        "title": "{{.Name}}"
      }
    }
`
