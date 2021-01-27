// Copyright (c) 2020, 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package pkg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/onsi/ginkgo"
)

type ApiEndpoint struct {
	AccessToken string `json:"access_token"`
	apiUrl      string
	httpClient  *retryablehttp.Client
}

// GetApiEndpoint returns the ApiEndpoint stub with AccessToken
func GetApiEndpoint() *ApiEndpoint {
	keycloakURL := fmt.Sprintf("https://keycloak.%s.%s/auth/realms/%s/protocol/openid-connect/token", EnvName, DnsZone, realm)
	body := fmt.Sprintf("username=%s&password=%s&grant_type=password&client_id=%s", username, GetVerrazzanoPassword(), clientId)
	status, resp := postWithCLient(keycloakURL, "application/x-www-form-urlencoded", strings.NewReader(body), GetKeycloakHTTPClient())
	var api ApiEndpoint
	if status == http.StatusOK {
		json.Unmarshal([]byte(resp), &api)
	} else {
		if status != http.StatusNotFound { //old installder may still using realm=sauron
			ginkgo.Fail(fmt.Sprintf("%v error getting API access token from %v", status, keycloakURL))
		}
	}
	api.apiUrl = getAPIURL()
	api.httpClient = GetVerrazzanoHTTPClient()
	return &api
}

// getAPIURL returns the Verrazzano REST API URL
func getAPIURL() string {
	return fmt.Sprintf("https://verrazzano.%s.%s/%s", EnvName, DnsZone, verrazzanoApiUriPrefix)
}

// GetSecrets
func (api *ApiEndpoint) GetSecrets() (*HttpResponse, error) {
	return api.get("secrets")
}

// CreateSecret
func (api *ApiEndpoint) CreateSecret(payload string) (*HttpResponse, error) {
	b := bytes.NewBuffer([]byte(payload))
	url := fmt.Sprintf("%s/secrets", api.apiUrl)
	return api.request("POST", url, b)
}

// DeleteSecret
func (api *ApiEndpoint) DeleteSecret(id string) (*HttpResponse, error) {
	url := fmt.Sprintf("%s/secrets/%s", api.apiUrl, id)
	return api.request("DELETE", url, nil)
}

// PatchSecret
func (api *ApiEndpoint) PatchSecret(id, body string) (*HttpResponse, error) {
	b := bytes.NewBuffer([]byte(body))
	url := fmt.Sprintf("%s/secrets/%s", api.apiUrl, id)
	return api.request("PATCH", url, b)
}

func (api *ApiEndpoint) get(path string) (*HttpResponse, error) {
	url := fmt.Sprintf("%s/%s", api.apiUrl, path)
	return api.request("GET", url, nil)
}

// Get
func (api *ApiEndpoint) Get() (*HttpResponse, error) {
	return api.get("")
}

// Get Instance
func (api *ApiEndpoint) GetInstance() (*HttpResponse, error) {
	return api.get("instance")
}

func (api *ApiEndpoint) request(method, url string, body io.Reader) (*HttpResponse, error) {
	req, _ := retryablehttp.NewRequest(method, url, body)
	if api.AccessToken != "" {
		value := fmt.Sprintf("Bearer %v", api.AccessToken)
		req.Header.Set("Authorization", value)
	}
	resp, err := api.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	return ProcHttpResponse(resp, err), nil
}

// Process the HTTP response by reading and closing the body, then returning
// the HttpResponse object.  This function is used to prevent file descriptor leaks
// and other problems.
// See https://github.com/golang/go/blob/master/src/net/http/response.go
//
// Params
//   resp: Http response returned by http call
//   httpErr: Http error returned by the http call
// Returns
//   HttpReponse which has the body and status code.
//
func ProcHttpResponse(resp *http.Response, httpErr error) *HttpResponse {
	if httpErr != nil {
		return &HttpResponse{}
	}

	// Must read entire body and close it.  See http.Response.Body doc
	defer resp.Body.Close()
	body, bodyErr := ioutil.ReadAll(resp.Body)
	return &HttpResponse{
		StatusCode: resp.StatusCode,
		Body:       body,
		BodyErr:    bodyErr,
	}
}
