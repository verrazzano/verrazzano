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
	"net/http"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Rancher HTTPS Configuration
const (
	// RancherName is the name of the component
	RancherName = "rancher"
	// CattleSystem is the namespace of the component
	CattleSystem                   = "cattle-system"
	RancherIngressCAName           = "tls-rancher-ingress"
	RancherAdditionalIngressCAName = "tls-ca-additional"
	RancherAdminSecret             = "rancher-admin-secret"
	RancherCACert                  = "ca.crt"
	RancherCAAdditionalPem         = "ca-additional.pem"
	contentTypeHeader              = "Content-Type"
	authorizationHeader            = "Authorization"
	applicationJSON                = "application/json"
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
	// Template body to PUT a new server url
	serverURLTmpl = `
{
  "name": "server-url",
  "value": "https://%s"
}`
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

//GetAdminSecret fetches the Rancher admin secret
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
func GetAdditionalCA(c client.Reader) ([]byte, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: CattleSystem,
		Name:      RancherAdditionalIngressCAName}

	if err := c.Get(context.TODO(), nsName, secret); err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	return secret.Data[RancherCAAdditionalPem], nil
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
	rootCA, err := GetRootCA(c)
	if err != nil {
		return nil, err
	}
	additionalCA, err := GetAdditionalCA(c)
	if err != nil {
		return nil, err
	}
	if len(rootCA) < 1 && len(additionalCA) < 1 {
		return nil, errors.New("neither root nor additional CA Secrets were found for Rancher")
	}

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				ServerName: hostname,
				RootCAs:    CertPool(rootCA, additionalCA),
			},
		},
	}, nil
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
	url := fmt.Sprintf("https://%s%s", r.hostname, RancherServerURLPath)
	serverURLBody := strings.NewReader(fmt.Sprintf(serverURLTmpl, r.hostname))
	req, err := http.NewRequest("PUT", url, serverURLBody)
	if err != nil {
		return err
	}
	req.Header.Set(contentTypeHeader, applicationJSON)
	req.Header.Set(authorizationHeader, fmt.Sprintf("Bearer %s", r.accessToken))
	resp, err := r.do(r.client, req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Failed to set server url: %s", resp.Status)
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
