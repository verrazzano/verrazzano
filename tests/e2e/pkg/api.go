// Copyright (c) 2020, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/onsi/gomega"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Username - the username of the Verrazzano admin user
	Username               = "verrazzano"
	realm                  = "verrazzano-system"
	verrazzanoAPIURLPrefix = "20210501"
)

// APIEndpoint contains information needed to access an API
type APIEndpoint struct {
	AccessToken string `json:"access_token"`
	APIURL      string
	HTTPClient  *retryablehttp.Client
}

func EventuallyGetAPIEndpoint(kubeconfigPath string) *APIEndpoint {
	var api *APIEndpoint
	gomega.Eventually(func() (*APIEndpoint, error) {
		var err error
		api, err = GetAPIEndpoint(kubeconfigPath)
		return api, err
	}, waitTimeout, pollingInterval).ShouldNot(gomega.BeNil())
	return api
}

// GetAPIEndpoint returns the APIEndpoint stub with AccessToken, from the given cluster
func GetAPIEndpoint(kubeconfigPath string) (*APIEndpoint, error) {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	ingress, err := clientset.NetworkingV1().Ingresses("keycloak").Get(context.TODO(), "keycloak", v1.GetOptions{})
	if err != nil {
		return nil, err
	}
	keycloakHTTPClient, err := GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	var ingressRules = ingress.Spec.Rules
	keycloakURL := fmt.Sprintf("https://%s/auth/realms/%s/protocol/openid-connect/token", ingressRules[0].Host, realm)
	password, err := GetVerrazzanoPassword()
	if err != nil {
		return nil, err
	}
	body := fmt.Sprintf("username=%s&password=%s&grant_type=password&client_id=%s", Username, password, keycloakAPIClientID)
	resp, err := doReq(keycloakURL, "POST", "application/x-www-form-urlencoded", "", "", "", strings.NewReader(body), keycloakHTTPClient)
	if err != nil {
		return nil, err
	}
	var api APIEndpoint
	if resp.StatusCode == http.StatusOK {
		json.Unmarshal([]byte(resp.Body), &api)
	} else {
		msg := fmt.Sprintf("error getting API access token from %s: %d", keycloakURL, resp.StatusCode)
		Log(Error, msg)
		return nil, errors.New(msg)
	}
	api.APIURL, err = getAPIURL(kubeconfigPath)
	if err != nil {
		return nil, err
	}
	api.HTTPClient, err = GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	return &api, nil
}

// getAPIURL returns the Verrazzano REST API URL for the cluster whose kubeconfig is given as argument
func getAPIURL(kubeconfigPath string) (string, error) {
	clientset, err := GetKubernetesClientsetForCluster(kubeconfigPath)
	if err != nil {
		return "", err
	}

	ingress, err := clientset.NetworkingV1().Ingresses("verrazzano-system").Get(context.TODO(), "verrazzano-ingress", v1.GetOptions{})
	if err != nil {
		return "", err
	}
	var ingressRules = ingress.Spec.Rules
	return fmt.Sprintf("https://%s/%s", ingressRules[0].Host, verrazzanoAPIURLPrefix), nil
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
	return ProcessHTTPResponse(resp)
}

// ProcessHTTPResponse processes the HTTP response by reading and closing the body, then returning
// the HTTPResponse object.  This function is used to prevent file descriptor leaks
// and other problems.
// See https://github.com/golang/go/blob/master/src/net/http/response.go
//
// Params
//
//	resp: Http response returned by http call
//	httpErr: Http error returned by the http call
//
// Returns
//
//	HttpReponse which has the body and status code.
func ProcessHTTPResponse(resp *http.Response) (*HTTPResponse, error) {
	// Must read entire body and close it.  See http.Response.Body doc
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	processedResponse := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Header:     resp.Header,
		Body:       body,
	}
	return processedResponse, nil
}

// GetIngress fetches ingress from api
func (api *APIEndpoint) GetIngress(namespace, name string) (*networkingv1.Ingress, error) {
	response, err := api.Get(fmt.Sprintf("apis/networking.k8s.io/v1/namespaces/%s/ingresses/%s", namespace, name))
	if err != nil {
		Log(Error, fmt.Sprintf("Error fetching ingress %s/%s from api, error: %v", namespace, name, err))
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		Log(Error, fmt.Sprintf("Error fetching ingress %s/%s from api, response: %v", namespace, name, response))
		return nil, fmt.Errorf("unexpected HTTP status code: %d", response.StatusCode)
	}

	ingress := networkingv1.Ingress{}
	err = json.Unmarshal(response.Body, &ingress)
	if err != nil {
		Log(Error, fmt.Sprintf("Invalid response for ingress %s/%s from api, error: %v", namespace, name, err))
		return nil, err
	}

	return &ingress, nil
}

// GetElasticURL fetches ElasticSearch endpoint URL
func (api *APIEndpoint) GetElasticURL() (string, error) {
	ingress, err := api.GetIngress("verrazzano-system", "vmi-system-os-ingest")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host), nil
}

// GetVerrazzanoIngressURL fetches Verrazzano-Ingress endpoint URL
func (api *APIEndpoint) GetVerrazzanoIngressURL() (string, error) {
	ingress, err := api.GetIngress("verrazzano-system", "verrazzano-ingress")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("https://%s", ingress.Spec.Rules[0].Host), nil
}
