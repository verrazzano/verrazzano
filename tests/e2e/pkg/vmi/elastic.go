// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

// Open contains information about the Opensearch instance
type Open struct {
	ClusterName   string      `json:"cluster_name"`
	EsVersion     OpenVersion `json:"version"`
	binding       string
	vmiHTTPClient *retryablehttp.Client
}

// OpenVersion contains information about the version of Opensearch instance
type OpenVersion struct {
	Number       string `json:"number"`
	Distribution string `json:"distribution"`
}

// GetOpen gets Open representing the opensearch cluster with the binding name
func GetOpen(binding string) *Open {
	return &Open{
		binding: binding,
	}
}

// PodsRunning checks if all opensearch required pods are running
func (e *Open) PodsRunning() bool {
	expectedOpenPods := []string{
		fmt.Sprintf("vmi-%s-es-master", e.binding),
		fmt.Sprintf("vmi-%s-kibana", e.binding),
		fmt.Sprintf("vmi-%s-grafana", e.binding),
		fmt.Sprintf("vmi-%s-prometheus", e.binding),
		fmt.Sprintf("vmi-%s-api", e.binding)}
	running, _ := pkg.PodsRunning("verrazzano-system", expectedOpenPods)

	if running {
		expectedOpenPods = []string{
			fmt.Sprintf("vmi-%s-es-ingest", e.binding),
			fmt.Sprintf("vmi-%s-es-data", e.binding)}
		running, _ = pkg.PodsRunning("verrazzano-system", expectedOpenPods)
	}

	return running
}

// getResponseBody gets the response body for the specified path from opensearch cluster
func (e *Open) getResponseBody(path string) ([]byte, error) {
	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return nil, err
	}

	api := pkg.EventuallyGetAPIEndpoint(kubeConfigPath)
	esURL, err := api.GetOpenURL()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting Opensearch URL: %v", err))
		return nil, err
	}

	esURL = esURL + path

	password, err := pkg.GetVerrazzanoPasswordInCluster(kubeConfigPath)
	if err != nil {
		return nil, err
	}

	return e.retryGet(esURL, pkg.Username, password, kubeConfigPath)
}

// Connect checks if the Opensearch cluster can be connected
func (e *Open) Connect() bool {
	body, err := e.getResponseBody("/")
	if err != nil {
		return false
	}
	err = json.Unmarshal(body, e)
	return err == nil
}

func (e *Open) retryGet(url, username, password string, kubeconfigPath string) ([]byte, error) {
	req, _ := retryablehttp.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	client, err := e.GetVmiHTTPClient(kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Error getting HTTP client: %v", err))
		return nil, err
	}
	client.CheckRetry = pkg.GetRetryPolicy()
	resp, err := client.Do(req)
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Error GET %v error: %v", url, err))
		return nil, err
	}
	if resp.StatusCode != 200 {
		pkg.Log(pkg.Info, fmt.Sprintf("Response status code: %d", resp.StatusCode))
	}
	httpResp, err := pkg.ProcessHTTPResponse(resp)
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Error reading response from GET %v error: %v", url, err))
		return nil, err
	}
	if httpResp.StatusCode == http.StatusNotFound {
		err = fmt.Errorf("url %s returned not found", url)
		pkg.Log(pkg.Info, fmt.Sprintf("NotFound %v error: %v", url, err))
		return nil, err
	}
	return httpResp.Body, nil
}

func (e *Open) GetVmiHTTPClient(kubeconfigPath string) (*retryablehttp.Client, error) {
	if e.vmiHTTPClient == nil {
		var err error
		e.vmiHTTPClient, err = pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}
	return e.vmiHTTPClient, nil
}

// ListIndices lists Opensearch indices
func (e *Open) ListIndices() []string {
	idx := []string{}
	for i := range e.getIndices() {
		idx = append(idx, i)
	}
	return idx
}

// getIndices gets index metadata (aliases, mappings, and settings) of all Opensearch indices in the given cluster
func (e *Open) getIndices() map[string]interface{} {
	body, err := e.getResponseBody("/_all")
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Error ListIndices error: %v", err))
		return nil
	}
	var indices map[string]interface{}
	json.Unmarshal(body, &indices)
	return indices
}

// CheckTLSSecret checks the Opensearch secret
func (e *Open) CheckTLSSecret() bool {
	secretName := fmt.Sprintf("%v-tls", e.binding)
	return pkg.SecretsCreated("verrazzano-system", secretName)
}

// CheckHealth checks the health status of Opensearch cluster
// Returns true if the health status is green otherwise false
func (e *Open) CheckHealth(kubeconfigPath string) bool {
	supported, err := pkg.IsVerrazzanoMinVersion("1.1.0", kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting Verrazzano version: %v", err))
		return false
	}
	if !supported {
		pkg.Log(pkg.Info, "Skipping Elasticsearch cluster health check since version < 1.1.0")
		return true
	}
	body, err := e.getResponseBody("/_cluster/health")
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting cluster health: %v", err))
		return false
	}
	pkg.Log(pkg.Info, fmt.Sprintf("Response body %v", string(body)))
	status, err := httputil.ExtractFieldFromResponseBodyOrReturnError(string(body), "status", "unable to find status in Opensearch health response")
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error extracting health status from response body: %v", err))
		return false
	}
	indexTemplate, _ := e.getResponseBody("/_index_template")
	pkg.Log(pkg.Info, fmt.Sprintf("IndexTemplate: %v", string(indexTemplate)))
	catIndices, _ := e.getResponseBody("/_cat/indices")
	pkg.Log(pkg.Info, fmt.Sprintf("Indices: \n%v", string(catIndices)))
	if status == "green" {
		pkg.Log(pkg.Info, "Opensearch cluster health status is green")
		return true
	}
	pkg.Log(pkg.Error, fmt.Sprintf("Opensearch cluster health status is %v instead of green", status))
	return false
}

// CheckIndicesHealth checks the health status of indices in a cluster
// Returns true if the health status of all the indices is green otherwise false
func (e *Open) CheckIndicesHealth(kubeconfigPath string) bool {
	supported, err := pkg.IsVerrazzanoMinVersion("1.1.0", kubeconfigPath)
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting Verrazzano version: %v", err))
		return false
	}
	if !supported {
		pkg.Log(pkg.Info, "Skipping Elasticsearch indices health check since version < 1.1.0")
		return true
	}
	body, err := e.getResponseBody("/_cat/indices?format=json")
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting cluster indices: %v", err))
		return false
	}
	pkg.Log(pkg.Info, fmt.Sprintf("Response body %v", string(body)))
	var indices []map[string]interface{}
	if err := json.Unmarshal(body, &indices); err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error unmarshalling indices response body: %v", err))
		return false
	}

	for _, index := range indices {
		pkg.Log(pkg.Debug, fmt.Sprintf("Index details: %v", index))
		val, found := index["health"]
		if !found {
			pkg.Log(pkg.Error, fmt.Sprintf("Not able to find the health of the index: %v", index))
			return false
		}
		if val.(string) != "green" {
			pkg.Log(pkg.Error, fmt.Sprintf("Current index health status %v is not green", val))
			return false
		}
	}
	pkg.Log(pkg.Info, "The health status of all the indices is green")
	return true
}

// //Check the Opensearch certificate
// func (e *Open) CheckCertificate() bool {
//	certList, _ := pkg.ListCertificates("verrazzano-system")
//	for _, cert := range certList.Items {
//		if cert.Name == fmt.Sprintf("%v-tls", e.binding) {
//			pkg.Log(pkg.Info, fmt.Sprintf("Found Certificate %v for binding %v", cert.Name, e.binding))
//			for _, condition := range cert.Status.Conditions {
//				if condition.Type == "Ready" {
//					pkg.Log(pkg.Info, fmt.Sprintf("Certificate %v status: Ready = %v", cert.Name, condition.Status))
//					return condition.Status == "True"
//				}
//			}
//		}
//	}
//	return false
// }

// CheckIngress checks the Opensearch Ingress
func (e *Open) CheckIngress() bool {
	ingressList, err := pkg.ListIngresses("verrazzano-system")
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Could not get list of ingresses: %v", err))
		return false
	}
	for _, ingress := range ingressList.Items {
		if ingress.Name == fmt.Sprintf("vmi-%v-es-ingest", e.binding) {
			pkg.Log(pkg.Info, fmt.Sprintf("Found Ingress %v for binding %v", ingress.Name, e.binding))
			return true
		}
	}
	return false
}
