// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"time"

	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"strings"
)

const (
	grafanaErrMsgFmt         = "Failed to GET Grafana testDashboard: status=%d: body=%s"
	testDashboardTitle       = "E2ETestDashboard"
	systemDashboardTitle     = "Host Metrics"
	openSearchMetricsDbTitle = "OpenSearch Summary Dashboard"
)

type DashboardMetadata struct {
	ID      int    `json:"id"`
	Slug    string `json:"slug"`
	Status  string `json:"status"`
	UID     string `json:"uid"`
	URL     string `json:"url"`
	Version int    `json:"version"`
}

// CreateGrafanaDashboard creates a grafana dashboard using the JSON string provided
func CreateGrafanaDashboard(body string) (*HTTPResponse, error) {
	path := "api/dashboards/db"
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return nil, err
	}
	url := fmt.Sprintf("%s/%s", GetSystemGrafanaIngressURL(kubeconfigPath), path)
	configPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return nil, err
	}
	password, err := GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	Log(Debug, fmt.Sprintf("REST API path: %v \nQuery: \n%v", url, body))
	resp, err := postGrafanaWithBasicAuth(url, body, "verrazzano", password, configPath)
	return resp, err
}

// GetGrafanaDashboard returns the dashboard metadata for the given uid.
func GetGrafanaDashboard(uid string) (*HTTPResponse, error) {
	path := fmt.Sprintf("/api/dashboards/uid/%s", uid)
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return nil, err
	}
	url := fmt.Sprintf("%s/%s", GetSystemGrafanaIngressURL(kubeconfigPath), path)
	configPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return nil, err
	}
	password, err := GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	Log(Debug, fmt.Sprintf("REST API path: %s", url))
	resp, err := getGrafanaWithBasicAuth(url, "", "verrazzano", password, configPath)
	return resp, err
}

// SearchGrafanaDashboard returns the dashboard metadata for the given uid.
func SearchGrafanaDashboard(searchParams map[string]string) (*HTTPResponse, error) {
	queryParams := ""
	for key, value := range searchParams {
		queryParams += fmt.Sprintf("%s=%s", key, value)
		queryParams += "&"
	}
	queryParams = strings.TrimSuffix(queryParams, "&")
	path := fmt.Sprintf("/api/search?%s", queryParams)
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return nil, err
	}
	url := fmt.Sprintf("%s/%s", GetSystemGrafanaIngressURL(kubeconfigPath), path)
	configPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		Log(Error, fmt.Sprintf(kubeconfigErrorFormat, err))
		return nil, err
	}
	password, err := GetVerrazzanoPasswordInCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	Log(Debug, fmt.Sprintf("REST API path: %s", url))
	resp, err := getGrafanaWithBasicAuth(url, "", "verrazzano", password, configPath)
	return resp, err
}

// GetSystemGrafanaIngressURL gets the system Grafana Ingress host in the given cluster
func GetSystemGrafanaIngressURL(kubeconfigPath string) string {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		Log(Error, fmt.Sprintf("Failed to get clientset for cluster %v", err))
		return ""
	}
	ingressList, _ := clientset.NetworkingV1().Ingresses(VerrazzanoNamespace).List(context.TODO(), metav1.ListOptions{})
	for _, ingress := range ingressList.Items {
		if ingress.Name == "vmi-system-grafana" {
			Log(Info, fmt.Sprintf("Found Opensearch Ingress %v, host %s", ingress.Name, ingress.Spec.Rules[0].Host))
			return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host)
		}
	}
	return ""
}

// postGrafanaWithBasicAuth retries POST using basic auth
func postGrafanaWithBasicAuth(url, body, username, password, kubeconfigPath string) (*HTTPResponse, error) {
	retryableClient, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "POST", "application/json", "", username, password, strings.NewReader(body), retryableClient)
}

// getGrafanaWithBasicAuth access Grafana with GET using basic auth, using a given kubeconfig
func getGrafanaWithBasicAuth(url string, hostHeader string, username string, password string, kubeconfigPath string) (*HTTPResponse, error) {
	retryableClient, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	return doReq(url, "GET", "", hostHeader, username, password, nil, retryableClient)
}

func TestOpenSearchGrafanaDashBoard(pollingInterval time.Duration, timeout time.Duration) {
	uid := "pIZicTl7z"
	gomega.Eventually(func() bool {
		resp, err := GetGrafanaDashboard(uid)
		if err != nil {
			Log(Error, err.Error())
			return false
		}
		if resp.StatusCode != http.StatusOK {
			Log(Error, fmt.Sprintf(grafanaErrMsgFmt, resp.StatusCode, string(resp.Body)))
			return false
		}
		body := make(map[string]map[string]string)
		json.Unmarshal(resp.Body, &body)
		return strings.Contains(body["dashboard"]["title"], openSearchMetricsDbTitle)
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}

func TestSystemGrafanaDashboard(pollingInterval time.Duration, timeout time.Duration) {
	// UID of system testDashboard, which is created by the VMO on startup.
	uid := "H0xWYyyik"
	gomega.Eventually(func() bool {
		resp, err := GetGrafanaDashboard(uid)
		if err != nil {
			Log(Error, err.Error())
			return false
		}
		if resp.StatusCode != http.StatusOK {
			Log(Error, fmt.Sprintf(grafanaErrMsgFmt, resp.StatusCode, string(resp.Body)))
			return false
		}
		body := make(map[string]map[string]string)
		json.Unmarshal(resp.Body, &body)
		return strings.Contains(body["dashboard"]["title"], systemDashboardTitle)
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}

func TestGrafanaTestDashboard(testDashboard DashboardMetadata, pollingInterval time.Duration, timeout time.Duration) {
	gomega.Eventually(func() bool {
		// UID of testDashboard, which is created by the previous test.
		uid := testDashboard.UID
		if uid == "" {
			return false
		}
		resp, err := GetGrafanaDashboard(uid)
		if err != nil {
			Log(Error, err.Error())
			return false
		}
		if resp.StatusCode != http.StatusOK {
			Log(Error, fmt.Sprintf(grafanaErrMsgFmt, resp.StatusCode, string(resp.Body)))
			return false
		}
		body := make(map[string]map[string]string)
		json.Unmarshal(resp.Body, &body)
		return strings.Contains(body["dashboard"]["title"], testDashboardTitle)
	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}

func TestSearchGrafanaDashboard(pollingInterval time.Duration, timeout time.Duration) {
	gomega.Eventually(func() bool {
		resp, err := SearchGrafanaDashboard(map[string]string{"query": testDashboardTitle})
		if err != nil {
			Log(Error, err.Error())
			return false
		}
		if resp.StatusCode != http.StatusOK {
			Log(Error, fmt.Sprintf(grafanaErrMsgFmt, resp.StatusCode, string(resp.Body)))
			return false
		}
		var body []map[string]string
		json.Unmarshal(resp.Body, &body)
		for _, dashboard := range body {
			if dashboard["title"] == "E2ETestDashboard" {
				return true
			}
		}
		return false

	}).WithPolling(pollingInterval).WithTimeout(timeout).Should(gomega.BeTrue())
}
