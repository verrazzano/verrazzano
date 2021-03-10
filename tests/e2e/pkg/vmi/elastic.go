// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
)

// Elastic contains information about the Elasticsearch instance
type Elastic struct {
	ClusterName   string `json:"cluster_name"`
	binding       string
	vmiHTTPClient *retryablehttp.Client
}

// GetElastic gets Elastic representing the elasticsearch cluster with the binding name
func GetElastic(binding string) *Elastic {
	return &Elastic{
		binding: binding,
	}
}

// PodsRunning checks if all elasticsearch required pods are running
func (e *Elastic) PodsRunning() bool {
	expectedElasticPods := []string{
		fmt.Sprintf("vmi-%s-es-master", e.binding),
		fmt.Sprintf("vmi-%s-kibana", e.binding),
		fmt.Sprintf("vmi-%s-grafana", e.binding),
		fmt.Sprintf("vmi-%s-prometheus", e.binding),
		fmt.Sprintf("vmi-%s-api", e.binding)}
	running := pkg.PodsRunning("verrazzano-system", expectedElasticPods)

	if running {
		expectedElasticPods = []string{
			fmt.Sprintf("vmi-%s-es-ingest", e.binding),
			fmt.Sprintf("vmi-%s-es-data", e.binding)}
		running = pkg.PodsRunning("verrazzano-system", expectedElasticPods)
	}

	return running
}

// Connect checks if the elasticsearch cluster can be connected
func (e *Elastic) Connect() bool {
	esURL := pkg.GetAPIEndpoint().GetElasticURL()
	body, err := e.retryGet(esURL, pkg.Username, pkg.GetVerrazzanoPassword())
	if err != nil {
		return false
	}
	err = json.Unmarshal(body, e)
	if err != nil {
		return false
	}
	return true
}

func (e *Elastic) retryGet(url, username, password string) ([]byte, error) {
	req, _ := retryablehttp.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	client := e.getVmiHTTPClient()
	client.CheckRetry = pkg.GetRetryPolicy(url)
	resp, err := client.Do(req)
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Error GET %v error: %v", url, err))
		return nil, err
	}
	if resp.StatusCode != 200 {
		pkg.Log(pkg.Info, fmt.Sprintf("Response status code: %d", resp.StatusCode))
	}
	httpResp := pkg.ProcHTTPResponse(resp, err)
	if httpResp.StatusCode == http.StatusNotFound {
		err = fmt.Errorf("url %s returned not found", url)
		pkg.Log(pkg.Info, fmt.Sprintf("NotFound %v error: %v", url, err))
		return nil, err
	}
	return httpResp.Body, nil
}

func (e *Elastic) getVmiHTTPClient() *retryablehttp.Client {
	if e.vmiHTTPClient == nil {
		e.vmiHTTPClient = pkg.GetBindingVmiHTTPClient(e.binding)
	}
	return e.vmiHTTPClient
}

// ListIndices lists elasticsearch indices
func (e *Elastic) ListIndices() []string {
	idx := []string{}
	for i := range e.GetIndices() {
		idx = append(idx, i)
	}
	return idx
}

// GetIndices gets index metadata (aliases, mappings, and settings) of all elasticsearch indices
func (e *Elastic) GetIndices() map[string]interface{} {
	esURL := pkg.GetAPIEndpoint().GetElasticURL() + "/_all"
	body, err := e.retryGet(esURL, pkg.Username, pkg.GetVerrazzanoPassword())
	if err != nil {
		pkg.Log(pkg.Info, fmt.Sprintf("Error ListIndices %v error: %v", esURL, err))
		return nil
	}
	var indices map[string]interface{}
	json.Unmarshal(body, &indices)
	return indices
}

// CheckTLSSecret checks the Elasticsearch secret
func (e *Elastic) CheckTLSSecret() bool {
	secretName := fmt.Sprintf("%v-tls", e.binding)
	return pkg.SecretsCreated("verrazzano-system", secretName)
}

////Check the Elasticsearch certificate
//func (e *Elastic) CheckCertificate() bool {
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
//}

// CheckIngress checks the Elasticsearch Ingress
func (e *Elastic) CheckIngress() bool {
	ingressList, _ := pkg.ListIngresses("verrazzano-system")
	for _, ingress := range ingressList.Items {
		if ingress.Name == fmt.Sprintf("vmi-%v-es-ingest", e.binding) {
			pkg.Log(pkg.Info, fmt.Sprintf("Found Ingress %v for binding %v", ingress.Name, e.binding))
			return true
		}
	}
	return false
}
