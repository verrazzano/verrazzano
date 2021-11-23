// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancher

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
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
)

func NewClient(c client.Reader, hostname, password string) (*RESTClient, error) {
	hc, err := HttpClient(c)
	if err != nil {
		return nil, err
	}

	return &RESTClient{
		client:   hc,
		do:       httpDo,
		hostname: hostname,
		password: password,
	}, nil
}

//GetAdminSecret fetches the Rancher admin secret
func GetAdminSecret(c client.Reader) (string, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: CattleSystem,
		Name:      AdminSecret}

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
		Name:      IngressCASecret}

	if err := c.Get(context.TODO(), nsName, secret); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return secret.Data[CACert], nil
}

func HttpClient(c client.Reader) (*http.Client, error) {
	rootCA, err := GetRootCA(c)
	if err != nil {
		return nil, err
	}
	if rootCA == nil {
		return nil, fmt.Errorf("root CA for rancher not found")
	}
	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(rootCA)

	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certPool,
			},
		},
	}, nil
}

func (r *RESTClient) SetLoginToken() error {
	loginTokenUrl := fmt.Sprintf("https://%s%s", r.hostname, loginActionPath)
	loginTokenBody := strings.NewReader(fmt.Sprintf(loginActionTmpl, r.password))
	req, err := http.NewRequest("POST", loginTokenUrl, loginTokenBody)
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

	accessTokenUrl := fmt.Sprintf("https://%s%s", r.hostname, tokenPath)
	req, err := http.NewRequest("POST", accessTokenUrl, strings.NewReader(tokenBody))
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
	url := fmt.Sprintf("https://%s%s", r.hostname, serverUrlPath)
	serverURLBody := strings.NewReader(fmt.Sprintf(serverUrlTmpl, r.hostname))
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
		return fmt.Errorf("failed to set server url: %s", resp.Status)
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
