// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
)

const (
	jaegerServiceIndexPrefix = "verrazzano-jaeger-jaeger-service"
	jaegerSpanIndexPrefix    = "verrazzano-jaeger-jaeger-span"
	jaegerClusterNameLabel   = "verrazzano_local"
	jaegerClusterName        = "local"
)

type JaegerTraceData struct {
	TraceID string `json:"traceID"`
	Spans   []struct {
		TraceID       string `json:"traceID"`
		SpanID        string `json:"spanID"`
		Flags         int    `json:"flags"`
		OperationName string `json:"operationName"`
		References    []struct {
			RefType string `json:"refType"`
			TraceID string `json:"traceID"`
			SpanID  string `json:"spanID"`
		} `json:"references"`
		StartTime int64 `json:"startTime"`
		Duration  int   `json:"duration"`
		Tags      []struct {
			Key   string      `json:"key"`
			Type  string      `json:"type"`
			Value interface{} `json:"value"`
		} `json:"tags"`
		Logs []struct {
			Timestamp int64 `json:"timestamp"`
			Fields    []struct {
				Key   string `json:"key"`
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"fields"`
		} `json:"logs"`
		ProcessID string      `json:"processID"`
		Warnings  interface{} `json:"warnings"`
	} `json:"spans"`
	Processes struct {
		P1 struct {
			ServiceName string `json:"serviceName"`
			Tags        []struct {
				Key   string `json:"key"`
				Type  string `json:"type"`
				Value string `json:"value"`
			} `json:"tags"`
		} `json:"p1"`
	} `json:"processes"`
	Warnings interface{} `json:"warnings"`
}

type JaegerTraceDataWrapper struct {
	Data   []JaegerTraceData `json:"data"`
	Total  int               `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Errors interface{}       `json:"errors"`
}

func VerifyJaegerSpans(service string) bool {
	return false
}

func IsJaegerInstanceCreated() (bool, error) {
	deployments, err := ListDeployments(constants.VerrazzanoMonitoringNamespace)
	if err != nil {
		return false, err
	}
	if len(deployments.Items) > 0 {
		return true, nil
	}
	return false, nil
}

func GetJaegerSpanIndexInElasticSearch(kubeconfigPath string) []string {
	var jaegerIndices []string
	for _, indexName := range listSystemElasticSearchIndices(kubeconfigPath) {
		if strings.HasPrefix(indexName, jaegerSpanIndexPrefix) {
			jaegerIndices = append(jaegerIndices, indexName)
		}
	}
	return jaegerIndices
}

func GetJaegerServiceIndexInElasticSearch(kubeconfigPath string) []string {
	var jaegerIndices []string
	for _, indexName := range listSystemElasticSearchIndices(kubeconfigPath) {
		if strings.HasPrefix(indexName, jaegerServiceIndexPrefix) {
			jaegerIndices = append(jaegerIndices, indexName)
		}
	}
	return jaegerIndices
}

// IsJaegerMetricFound validates if the given jaeger metrics contain the labels with values specified as key-value pairs of the map
func IsJaegerMetricFound(kubeconfigPath, metricName string, kv map[string]string) bool {
	compMetrics, err := QueryMetricWithLabel(metricName, kubeconfigPath, jaegerClusterNameLabel, jaegerClusterName)
	if err != nil {
		return false
	}
	metrics := JTq(compMetrics, "data", "result").([]interface{})
	for _, metric := range metrics {
		metricFound := true
		for key, value := range kv {
			if Jq(metric, "metric", key) != value {
				metricFound = false
				break
			}
		}
		return metricFound
	}
	return false
}

func ListJaegerTraces(kubeconfigPath string, serviceName string) []string {
	var traces []string
	url := fmt.Sprintf("%s/api/traces?service=%s", getJaegerURL(kubeconfigPath), serviceName)
	username, password, err := getJaegerUsernamePassword(kubeconfigPath)
	if err != nil {
		return traces
	}
	resp, err := getJaegerWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting Jaeger traces: url=%s, error=%v", url, err))
		return traces
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Error retrieving Jaeger traces: url=%s, status=%d", url, resp.StatusCode))
		return traces
	}
	Log(Debug, fmt.Sprintf("traces: %s", resp.Body))
	var jaegerTraceDataWrapper JaegerTraceDataWrapper
	json.Unmarshal(resp.Body, &jaegerTraceDataWrapper)
	for _, traceObj := range jaegerTraceDataWrapper.Data {
		traces = append(traces, traceObj.TraceID)
	}
	return traces
}

func ListServicesInJaeger(kubeconfigPath string) []string {
	var services []string
	url := fmt.Sprintf("%s/api/services", getJaegerURL(kubeconfigPath))
	username, password, err := getJaegerUsernamePassword(kubeconfigPath)
	if err != nil {
		return services
	}
	resp, err := getJaegerWithBasicAuth(url, "", username, password, kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Error getting Jaeger traces: url=%s, error=%v", url, err))
		return services
	}
	if resp.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Error retrieving Jaeger traces: url=%s, status=%d", url, resp.StatusCode))
		return services
	}
	Log(Debug, fmt.Sprintf("traces: %s", resp.Body))
	var serviceMap map[string][]string
	json.Unmarshal(resp.Body, &serviceMap)
	for _, name := range serviceMap["data"] {
		services = append(services, name)
	}
	return services
}

// getJaegerWithBasicAuth access Jaeger with GET using basic auth, using a given kubeconfig
func getJaegerWithBasicAuth(url string, hostHeader string, username string, password string, kubeconfigPath string) (*HTTPResponse, error) {
	retryableClient, err := getJaegerClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "GET", "", hostHeader, username, password, nil, retryableClient)
}

// getJaegerClient returns the Jaeger client which can be used for GET/POST operations using a given kubeconfig
func getJaegerClient(kubeconfigPath string) (*retryablehttp.Client, error) {
	client, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return client, err
}

// getJaegerURL returns Jaeger URL from the corresponding ingress resource using the given kubeconfig
func getJaegerURL(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingressList, _ := clientset.NetworkingV1().Ingresses(VerrazzanoNamespace).List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "verrazzano-jaeger" {
			Log(Info, fmt.Sprintf("Found Jaeger Ingress %v, host %s", ingress.Name, ingress.Spec.Rules[0].Host))
			return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
		}
	}
	return ""
}

// getJaegerUsernamePassword returns the username/password for connecting to Jaeger
func getJaegerUsernamePassword(kubeconfigPath string) (username, password string, err error) {
	password, err = GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return "", "", err
	}
	return "verrazzano", password, err
}
