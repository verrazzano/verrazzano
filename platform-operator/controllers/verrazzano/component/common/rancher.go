// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package common

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Rancher HTTPS Configuration
const (
	// RancherName is the name of the component
	RancherName = "rancher"
	// CattleSystem is the namespace of the component
	CattleSystem         = "cattle-system"
	RancherIngressCAName = "tls-rancher-ingress"
	RancherAdminSecret   = "rancher-admin-secret"
	RancherCACert        = "ca.crt"
	contentTypeHeader    = "Content-Type"
	authorizationHeader  = "Authorization"
	applicationJSON      = "application/json"
	// Path to get a login token
	loginActionPath = "/v3-public/localProviders/local?action=login"
	// Template body to POST for a login token
	loginActionTmpl = `
{
  "Username": "admin",
  "Password": "%s"
}
`
	// Path to get an access token
	tokPath = "/v3/token"
	// Body to POST for an access token (login token should be Bearer token)
	tokPostBody = `
{
  "type": "token",
  "description": "automation"
}`

	// RancherServerURLPath Path to update server URL, as in PUT during PostInstall
	RancherServerURLPath = "/v3/settings/server-url"
	// rancherDeleteHostURLPath Path to delete Rancher localhost
	rancherDeleteHostURLPath = "/v3/clusters/local"
	// Template body to PUT a new server url
	serverURLTmpl = `
{
  "name": "server-url",
  "value": "https://%s"
}`
	// rancherActivateOCIDriverURLPath Path to activate Rancher OCI Driver
	rancherActivateOCIDriverURLPath = "/v3/nodeDrivers/oci?action=activate"
	// rancherActivateOKEDriverURLPath Path to activate Rancher OKE Driver
	rancherActivateOKEDriverURLPath = "/v3/kontainerDrivers/oraclecontainerengine?action=activate"
)

type (
	RESTClient struct {
		client      *http.Client
		do          func(hc *http.Client, req *http.Request) (*http.Response, error)
		hostname    string
		password    string
		loginToken  string
		accessToken string
	}
	// TokenResponse is the response format Rancher uses when sending tokens in HTTP responses
	TokenResponse struct {
		Token string `json:"token"`
	}
)

func NewClient(c client.Reader, hostname, password string) (*RESTClient, error) {
	hc, err := HTTPClient(c, hostname)
	if err != nil {
		return nil, err
	}

	return &RESTClient{
		client:   hc,
		do:       HTTPDo,
		hostname: hostname,
		password: password,
	}, nil
}

// GetAdminSecret fetches the Rancher admin secret
func GetAdminSecret(c client.Reader) (string, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: CattleSystem,
		Name:      RancherAdminSecret}

	if err := c.Get(context.TODO(), nsName, secret); err != nil {
		return "", err
	}
	return string(secret.Data["password"]), nil
}

// GetRootCA gets the root CA certificate from the Rancher TLS secret. If the secret does not exist, we
// return a nil slice.
func GetRootCA(c client.Reader) ([]byte, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: CattleSystem,
		Name:      RancherIngressCAName}

	if err := c.Get(context.TODO(), nsName, secret); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return secret.Data[RancherCACert], nil
}

// GetAdditionalCA fetches the Rancher additional CA secret
// returns empty byte array of the secret tls-ca-additional is not found
func GetAdditionalCA(c client.Reader) []byte {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: CattleSystem,
		Name:      constants.AdditionalTLS}

	if err := c.Get(context.TODO(), nsName, secret); err != nil {
		return []byte{}
	}

	return secret.Data[constants.AdditionalTLSCAKey]
}

func CertPool(certs ...[]byte) *x509.CertPool {
	certPool := x509.NewCertPool()
	for _, cert := range certs {
		if len(cert) > 0 {
			certPool.AppendCertsFromPEM(cert)
		}
	}
	return certPool
}

func HTTPClient(c client.Reader, hostname string) (*http.Client, error) {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 5
	httpClient := retryClient.StandardClient()
	rootCA, err := GetRootCA(c)
	if err != nil {
		return nil, err
	}
	additionalCA := GetAdditionalCA(c)

	if len(rootCA) < 1 && len(additionalCA) < 1 {
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				ServerName: hostname,
			},
		}
		return httpClient, nil
	}

	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			ServerName: hostname,
			RootCAs:    CertPool(rootCA, additionalCA),
		},
	}
	return httpClient, nil
}

func (r *RESTClient) SetLoginToken() error {
	loginTokenURL := fmt.Sprintf("https://%s%s", r.hostname, loginActionPath)
	loginTokenBody := strings.NewReader(fmt.Sprintf(loginActionTmpl, r.password))
	req, err := http.NewRequest("POST", loginTokenURL, loginTokenBody)
	if err != nil {
		return err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	resp, err := r.do(r.client, req)
	if err != nil {
		return err
	}
	loginToken, err := parseTokenResponse(resp)
	if err != nil {
		return err
	}
	r.loginToken = loginToken
	return nil
}

func (r *RESTClient) SetAccessToken() error {
	if r.loginToken == "" {
		if err := r.SetLoginToken(); err != nil {
			return err
		}
	}

	accessTokenURL := fmt.Sprintf("https://%s%s", r.hostname, tokPath)
	req, err := http.NewRequest("POST", accessTokenURL, strings.NewReader(tokPostBody))
	if err != nil {
		return err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set(authorizationHeader, fmt.Sprintf("Bearer %s", r.loginToken))
	resp, err := r.do(r.client, req)
	if err != nil {
		return err
	}
	accessToken, err := parseTokenResponse(resp)
	if err != nil {
		return err
	}
	r.accessToken = accessToken
	return nil
}

func (r *RESTClient) GetLoginToken() string {
	return r.loginToken
}

func (r *RESTClient) GetAccessToken() string {
	return r.accessToken
}

func (r *RESTClient) PutServerURL() error {
	serverURLBody := strings.NewReader(fmt.Sprintf(serverURLTmpl, r.hostname))
	resp, err := r.invokeRancherEndpoint(RancherServerURLPath, http.MethodPut, serverURLBody)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to set server url: %s", resp.Status)
	}
	return nil
}

func (r *RESTClient) DeleteLocalHost() error {
	resp, err := r.invokeRancherEndpoint(rancherDeleteHostURLPath, http.MethodDelete, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed calling https DELETE url %s: %v", fmt.Sprintf("https://%s%s", r.hostname, rancherDeleteHostURLPath), resp.Status)
	}
	return nil
}

func parseTokenResponse(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	tokenResponse := &TokenResponse{}
	if err := json.NewDecoder(resp.Body).Decode(tokenResponse); err != nil {
		return "", err
	}
	if tokenResponse.Token == "" {
		return "", errors.New("token not found in response")
	}
	return tokenResponse.Token, nil
}

//ActivateOCIDriver activates the OCI Node Driver in Rancher by invoking the /v3/nodeDrivers rest api.
func (r *RESTClient) ActivateOCIDriver() error {
	resp, err := r.invokeRancherEndpoint(rancherActivateOCIDriverURLPath, http.MethodPost, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed activating rancher oci driver, url %s: %v", fmt.Sprintf("https://%s%s", r.hostname, rancherActivateOCIDriverURLPath), resp.Status)
	}
	return nil
}

//ActivateOKEDriver activates the OKE Kubernetes Driver in Rancher by invoking the /v3/kontainerDrivers rest api.
func (r *RESTClient) ActivateOKEDriver() error {
	resp, err := r.invokeRancherEndpoint(rancherActivateOKEDriverURLPath, http.MethodPost, nil)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed activating rancher oke driver, url %s: %v", fmt.Sprintf("https://%s%s", r.hostname, rancherActivateOKEDriverURLPath), resp.Status)
	}
	return nil
}

func (r *RESTClient) invokeRancherEndpoint(path string, httpMethod string, requestBody io.Reader) (*http.Response, error) {
	url := fmt.Sprintf("https://%s%s", r.hostname, path)
	req, err := http.NewRequest(httpMethod, url, requestBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set(authorizationHeader, fmt.Sprintf("Bearer %s", r.accessToken))
	resp, err := r.do(r.client, req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	return resp, nil
}
