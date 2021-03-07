// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
)

// APIEndpoint contains information needed to access an API
type APIEndpoint struct {
	AccessToken string `json:"access_token"`
	APIURL      string
	HTTPClient  *retryablehttp.Client
}

// GetAPIEndpoint returns the APIEndpoint stub with AccessToken
func GetAPIEndpoint() *APIEndpoint {
	ingress, _ := GetKubernetesClientset().ExtensionsV1beta1().Ingresses("keycloak").Get(context.TODO(), "keycloak", v1.GetOptions{})
	var ingressRules []extensionsv1beta1.IngressRule = ingress.Spec.Rules
	keycloakURL := fmt.Sprintf("https://%s/auth/realms/%s/protocol/openid-connect/token", ingressRules[0].Host, realm)
	body := fmt.Sprintf("username=%s&password=%s&grant_type=password&client_id=%s", Username, GetVerrazzanoPassword(), clientID)
	status, resp := postWithClient(keycloakURL, "application/x-www-form-urlencoded", strings.NewReader(body), GetKeycloakHTTPClient())
	var api APIEndpoint
	if status == http.StatusOK {
		json.Unmarshal([]byte(resp), &api)
	} else {
		if status != http.StatusNotFound { //old installder may still using realm=sauron
			ginkgo.Fail(fmt.Sprintf("%v error getting API access token from %v", status, keycloakURL))
		}
	}
	api.APIURL = getAPIURL()
	api.HTTPClient = GetVerrazzanoHTTPClient()
	return &api
}

// getAPIURL returns the Verrazzano REST API URL
func getAPIURL() string {
	ingress, _ := GetKubernetesClientset().ExtensionsV1beta1().Ingresses("verrazzano-system").Get(context.TODO(), "verrazzano-console-ingress", v1.GetOptions{})
	var ingressRules []extensionsv1beta1.IngressRule = ingress.Spec.Rules
	return fmt.Sprintf("https://%s/%s", ingressRules[0].Host, verrazzanoAPIURLPrefix)
}

// Get Invoke GET API Request
func (api *APIEndpoint) Get(path string) (*HTTPResponse, error) {
	return api.Request(http.MethodGet, path, nil)
}

// Post Invoke POST API Request
func (api *APIEndpoint) Post(path string, body io.Reader) (*HTTPResponse, error) {
	return api.Request(http.MethodPost, path, body)
}

// Patch Invoke POST API Request
func (api *APIEndpoint) Patch(path string, body io.Reader) (*HTTPResponse, error) {
	return api.Request(http.MethodPut, path, body)
}

// Delete Invoke DELETE API Request
func (api *APIEndpoint) Delete(path string) (*HTTPResponse, error) {
	return api.Request(http.MethodDelete, path, nil)
}

// Request Invoke API
func (api *APIEndpoint) Request(method, path string, body io.Reader) (*HTTPResponse, error) {
	url := fmt.Sprintf("%s/%s", api.APIURL, path)
	req, _ := retryablehttp.NewRequest(method, url, body)
	if api.AccessToken != "" {
		value := fmt.Sprintf("Bearer %v", api.AccessToken)
		req.Header.Set("Authorization", value)
	}
	resp, err := api.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	return ProcHTTPResponse(resp, err), nil
}

// ProcHTTPResponse processes the HTTP response by reading and closing the body, then returning
// the HTTPResponse object.  This function is used to prevent file descriptor leaks
// and other problems.
// See https://github.com/golang/go/blob/master/src/net/http/response.go
//
// Params
//   resp: Http response returned by http call
//   httpErr: Http error returned by the http call
// Returns
//   HttpReponse which has the body and status code.
//
func ProcHTTPResponse(resp *http.Response, httpErr error) *HTTPResponse {
	if httpErr != nil {
		return &HTTPResponse{}
	}

	// Must read entire body and close it.  See http.Response.Body doc
	defer resp.Body.Close()
	body, bodyErr := ioutil.ReadAll(resp.Body)
	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		BodyErr:    bodyErr,
	}
}

//GetIngress fetches ingress from api
func (api *APIEndpoint) GetIngress(namespace, name string) extensionsv1beta1.Ingress {
	response, err := api.Get(fmt.Sprintf("apis/extensions/v1beta1/namespaces/%s/ingresses/%s", namespace, name))
	ExpectHTTPOk(response, err, fmt.Sprintf("Error fetching ingress %s/%s from api, error: %v, response: %v", namespace, name, err, response))
	ingress := extensionsv1beta1.Ingress{}
	err = json.Unmarshal(response.Body, &ingress)
	gomega.Expect(err).To(gomega.BeNil(), fmt.Sprintf("Invalid response for ingress %s/%s from api, error: %v", namespace, name, err))
	return ingress
}

//GetElasticURL fetches ElasticSearch endpoint URL
func (api *APIEndpoint) GetElasticURL() string {
	ingress := api.GetIngress("verrazzano-system", "vmi-system-es-ingest")
	return fmt.Sprintf("https://%s", ingress.Spec.TLS[0].Hosts[0])
}
