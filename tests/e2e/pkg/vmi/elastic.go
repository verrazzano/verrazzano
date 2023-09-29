// Copyright (c) 2020, 2023, Oracle and/or its affiliates.
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

// Opensearch contains information about the Opensearch instance
type Opensearch struct {
	ClusterName       string            `json:"cluster_name"`
	EsVersion         OpensearchVersion `json:"version"`
	binding           string
	vmiHTTPClient     *retryablehttp.Client
	OperatorManaged   bool
	opensearchIngress string
	osdIngress        string
}

// OpensearchVersion contains information about the version of Opensearch instance
type OpensearchVersion struct {
	Number       string `json:"number"`
	Distribution string `json:"distribution"`
}

// GetOpensearch gets Opensearch representing the opensearch cluster with the binding name
func GetOpensearch(binding string, operatorManaged bool) *Opensearch {
	opensearchIngress := fmt.Sprintf("vmi-%v-os-ingest", binding)
	osdIngress := fmt.Sprintf("vmi-%v-osd", binding)
	if operatorManaged {
		opensearchIngress = "opensearch"
		osdIngress = "opensearch-dashboards"
	}
	return &Opensearch{
		binding:           binding,
		OperatorManaged:   operatorManaged,
		opensearchIngress: opensearchIngress,
		osdIngress:        osdIngress,
	}
}

// PodsRunning checks if all opensearch required pods are running
func (e *Opensearch) PodsRunning() bool {
	expectedOpensearchPods := []string{
		fmt.Sprintf("vmi-%s-es-master", e.binding),
		fmt.Sprintf("vmi-%s-kibana", e.binding),
		fmt.Sprintf("vmi-%s-grafana", e.binding),
		fmt.Sprintf("vmi-%s-prometheus", e.binding),
		fmt.Sprintf("vmi-%s-api", e.binding)}
	running, _ := pkg.PodsRunning("verrazzano-system", expectedOpensearchPods)

	if running {
		expectedOpensearchPods = []string{
			fmt.Sprintf("vmi-%s-es-ingest", e.binding),
			fmt.Sprintf("vmi-%s-es-data", e.binding)}
		running, _ = pkg.PodsRunning("verrazzano-system", expectedOpensearchPods)
	}

	return running
}

// getResponseBody gets the response body for the specified path from opensearch cluster
func (e *Opensearch) getResponseBody(path string) ([]byte, error) {
	kubeConfigPath, err := k8sutil.GetKubeConfigLocation()
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Error getting kubeconfig: %v", err))
		return nil, err
	}

	api := pkg.EventuallyGetAPIEndpoint(kubeConfigPath)
	esURL, err := api.GetOpensearchURL()
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
func (e *Opensearch) Connect() bool {
	body, err := e.getResponseBody("/")
	if err != nil {
		return false
	}
	err = json.Unmarshal(body, e)
	return err == nil
}

func (e *Opensearch) retryGet(url, username, password string, kubeconfigPath string) ([]byte, error) {
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

func (e *Opensearch) GetOSIngressName() string {
	return e.opensearchIngress
}

func (e *Opensearch) GetOSDIngressName() string {
	return e.osdIngress
}

func (e *Opensearch) GetVmiHTTPClient(kubeconfigPath string) (*retryablehttp.Client, error) {
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
func (e *Opensearch) ListIndices() []string {
	idx := []string{}
	for i := range e.getIndices() {
		idx = append(idx, i)
	}
	return idx
}

// getIndices gets index metadata (aliases, mappings, and settings) of all Opensearch indices in the given cluster
func (e *Opensearch) getIndices() map[string]interface{} {
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
func (e *Opensearch) CheckTLSSecret() bool {
	secretName := fmt.Sprintf("%v-tls", e.binding)
	return pkg.SecretsCreated("verrazzano-system", secretName)
}

// CheckHealth checks the health status of Opensearch cluster
// Returns true if the health status is green otherwise false
func (e *Opensearch) CheckHealth(kubeconfigPath string) bool {
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
	} else if status == "yellow" { // Temporary WA for security audit log index as it gets created with 1 replica
		pkg.Log(pkg.Info, "Opensearch cluster health status is yellow")
		return true
	}
	pkg.Log(pkg.Error, fmt.Sprintf("Opensearch cluster health status is %v instead of green", status))
	return false
}

// CheckIndicesHealth checks the health status of indices in a cluster
// Returns true if the health status of all the indices is green otherwise false
func (e *Opensearch) CheckIndicesHealth(kubeconfigPath string) bool {
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
		// Temporary WA for security audit log index as it gets created with 1 replica
		if val.(string) == "red" {
			pkg.Log(pkg.Error, fmt.Sprintf("Current index health status %v is not green or yellow", val))
			return false
		}
	}
	pkg.Log(pkg.Info, "The health status of all the indices is green or yellow")
	return true
}

// //Check the Opensearch certificate
// func (e *Opensearch) CheckCertificate() bool {
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
func (e *Opensearch) CheckIngress() bool {
	ingressList, err := pkg.ListIngresses("verrazzano-system")
	if err != nil {
		pkg.Log(pkg.Error, fmt.Sprintf("Could not get list of ingresses: %v", err))
		return false
	}
	for _, ingress := range ingressList.Items {
		if ingress.Name == e.opensearchIngress {
			pkg.Log(pkg.Info, fmt.Sprintf("Found Ingress %v for binding %v", ingress.Name, e.binding))
			return true
		}
	}
	return false
}
