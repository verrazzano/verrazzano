// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hashicorp/go-retryablehttp"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	NUM_RETRIES    = 7
	RETRY_WAIT_MIN = 1 * time.Second
	RETRY_WAIT_MAX = 30 * time.Second
)

type Elastic struct {
	ClusterName   string `json:"cluster_name"`
	binding       string
	vmiHttpClient *retryablehttp.Client
}

func GetSystemVmiHttpClient() *retryablehttp.Client {
	vmiRawClient := getHttpClientWIthCABundle(getSystemVMICACert())
	return newRetryableHttpClient(vmiRawClient)
}

// GetSystemVMICredentials - Obtain VMI system credentials
func GetSystemVMICredentials() (*UsernamePassword, error) {
	vmi, err := GetVerrazzanoMonitoringInstance("verrazzano-system", "system")
	if err != nil {
		return nil, fmt.Errorf("error getting system VMI: %w", err)
	}

	secret, err := GetSecret("verrazzano-system", vmi.Spec.SecretsName)
	if err != nil {
		return nil, err
	}

	username := secret.Data["username"]
	password := secret.Data["password"]
	if username == nil || password == nil {
		return nil, fmt.Errorf("username and password fields required in secret %v", secret)
	}

	return &UsernamePassword{
		Username: string(username),
		Password: string(password),
	}, nil
}

//Gets Elastic representing the elasticsearch cluster with the binding name
func GetElastic(binding string) *Elastic {
	return &Elastic{
		binding: binding,
	}
}

//Checks if all elasticsearch required pods are running
func (e *Elastic) PodsRunning() bool {
	expectedElasticPods := []string{
		fmt.Sprintf("vmi-%s-es-master", e.binding),
		fmt.Sprintf("vmi-%s-kibana", e.binding),
		fmt.Sprintf("vmi-%s-grafana", e.binding),
		fmt.Sprintf("vmi-%s-prometheus", e.binding),
		fmt.Sprintf("vmi-%s-api", e.binding)}
	running := PodsRunning("verrazzano-system", expectedElasticPods)

	if running {
		expectedElasticPods = []string{
			fmt.Sprintf("vmi-%s-es-ingest", e.binding),
			fmt.Sprintf("vmi-%s-es-data", e.binding)}
		running = PodsRunning("verrazzano-system", expectedElasticPods)
	}

	return running
}

//Checks if the elasticsearch cluster can be connected
func (e *Elastic) Connect() bool {
	esURL := GetVerrazzanoInstance().ElasticURL
	body, err := e.retryGet(esURL, username, GetVerrazzanoPassword())
	if err != nil {
		return false
	}
	err = json.Unmarshal(body, e)
	if err != nil {
		return false
	}
	return true
}

func newRetryableHttpClient(client *http.Client) *retryablehttp.Client {
	retryableClient := retryablehttp.NewClient() //default of 4 retries is sufficient for us
	retryableClient.RetryMax = NUM_RETRIES
	retryableClient.RetryWaitMin = RETRY_WAIT_MIN
	retryableClient.RetryWaitMax = RETRY_WAIT_MAX
	retryableClient.HTTPClient = client
	return retryableClient
}

func (e *Elastic) retryGet(url, username, password string) ([]byte, error) {
	req, _ := retryablehttp.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	client := e.getVmiHttpClient()
	client.CheckRetry = GetRetryPolicy(url)
	resp, err := client.Do(req)
	if err != nil {
		Log(Info, fmt.Sprintf("Error GET %v error: %v", url, err))
		return nil, err
	}
	if resp.StatusCode != 200 {
		Log(Info, fmt.Sprintf("Response status code: %d", resp.StatusCode))
	}
	httpResp := ProcHttpResponse(resp, err)
	if httpResp.StatusCode == http.StatusNotFound {
		err = errors.New(fmt.Sprintf("url %s returned not found", url))
		Log(Info, fmt.Sprintf("NotFound %v error: %v", url, err))
		return nil, err
	}
	return httpResp.Body, nil
}

func (e *Elastic) getVmiHttpClient() *retryablehttp.Client {
	if e.vmiHttpClient == nil {
		e.vmiHttpClient = GetBindingVmiHttpClient(e.binding)
	}
	return e.vmiHttpClient
}

//Lists elasticsearch indices
func (e *Elastic) ListIndices() []string {
	idx := []string{}
	for i, _ := range e.GetIndices() {
		idx = append(idx, i)
	}
	return idx
}

//GetIndices gets index metadata (aliases, mappings, and settings) of all elasticsearch indices
func (e *Elastic) GetIndices() map[string]interface{} {
	esURL := GetVerrazzanoInstance().ElasticURL
	body, err := e.retryGet(esURL, username, GetVerrazzanoPassword())
	if err != nil {
		Log(Info, fmt.Sprintf("Error ListIndices %v error: %v", esURL, err))
		return nil
	}
	var indices map[string]interface{}
	json.Unmarshal(body, &indices)
	return indices
}

func GetBindingVmiHttpClient(bindingName string) *retryablehttp.Client {
	bindingVmiCaCert := getBindingVMICACert(bindingName)
	vmiRawClient := getHttpClientWIthCABundle(bindingVmiCaCert)
	return newRetryableHttpClient(vmiRawClient)
}

//Lookup the Elasticsearch host
func (e *Elastic) LookupHost() bool {
	esURL := GetVerrazzanoInstance().ElasticURL
	return Lookup(esURL)
}

//Check the Elasticsearch secret
func (e *Elastic) CheckTlsSecret() bool {
	secretName := fmt.Sprintf("%v-tls", e.binding)
	return SecretsCreated("verrazzano-system", secretName)
}

//Check the Elasticsearch certificate
func (e *Elastic) CheckCertificate() bool {
	certList, _ := ListCertificates("verrazzano-system")
	for _, cert := range certList.Items {
		if cert.Name == fmt.Sprintf("%v-tls", e.binding) {
			Log(Info, fmt.Sprintf("Found Certificate %v for binding %v", cert.Name, e.binding))
			for _, condition := range cert.Status.Conditions {
				if condition.Type == "Ready" {
					Log(Info, fmt.Sprintf("Certificate %v status: Ready = %v", cert.Name, condition.Status))
					return condition.Status == "True"
				}
			}
		}
	}
	return false
}

//Check the Elasticsearch Ingress
func (e *Elastic) CheckIngress() bool {
	ingressList, _ := ListIngresses("verrazzano-system")
	for _, ingress := range ingressList.Items {
		if ingress.Name == fmt.Sprintf("vmi-%v-es-ingest", e.binding) {
			Log(Info, fmt.Sprintf("Found Ingress %v for binding %v", ingress.Name, e.binding))
			return true
		}
	}
	return false
}

func getBindingVMICACert(bindingName string) []byte {
	return doGetCACertFromSecret(fmt.Sprintf("%v-tls", bindingName), "verrazzano-system")
}

func getHttpClientWIthCABundle(caData []byte) *http.Client {
	tr := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: rootCertPool(caData)}}

	proxyURL := getProxyURL()
	if proxyURL != "" {
		tURL := url.URL{}
		tURLProxy, _ := tURL.Parse(proxyURL)
		tr.Proxy = http.ProxyURL(tURLProxy)
	}

	ipResolve := getManagementClusterNodeIP()
	if ipResolve != "" {
		dialer := &net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}
		tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			Log(Debug, fmt.Sprintf("original address %s", addr))
			if strings.Contains(addr, "127.0.0.1") && strings.Contains(addr, ":443") {
				addr = ipResolve + ":443"
				Log(Debug, fmt.Sprintf("modified address %s", addr))
			}
			return dialer.DialContext(ctx, network, addr)
		}
	}

	return &http.Client{Transport: tr}
}

// If testing against KIND, returns the control-plane node ip ; "" otherwise
func getManagementClusterNodeIP() string {
	pods := ListPods("ingress-nginx")
	for i := range pods.Items {
		pod := pods.Items[i]
		if strings.HasPrefix(pod.Name, "ingress-controller-ingress-nginx-controller-") {
			return pod.Status.HostIP
		}
	}

	return ""
}

func getSystemVMICACert() []byte {
	return doGetCACertFromSecret("system-tls", "verrazzano-system")
}