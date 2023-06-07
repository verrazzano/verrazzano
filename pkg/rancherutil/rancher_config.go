// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherutil

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/verrazzano/verrazzano/pkg/nginxutil"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	cons "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"github.com/verrazzano/verrazzano/platform-operator/controllers/verrazzano/component/common"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	rancherNamespace   = "cattle-system"
	rancherIngressName = "rancher"

	rancherAdminSecret   = "rancher-admin-secret" //nolint:gosec //#gosec G101
	rancherAdminUsername = "admin"

	loginPath  = "/v3-public/localProviders/local?action=login"
	tokensPath = "/v3/tokens" //nolint:gosec
)

// DefaultRancherIngressHostPrefix is the default internal Ingress host prefix used for Rancher API requests
const DefaultRancherIngressHostPrefix = "ingress-controller-ingress-nginx-controller."

type RancherConfig struct {
	Host                     string
	BaseURL                  string
	APIAccessToken           string
	CertificateAuthorityData []byte
	AdditionalCA             []byte
	User                     string
}

var DefaultRetry = wait.Backoff{
	Steps:    10,
	Duration: 1 * time.Second,
	Factor:   2.0,
	Jitter:   0.1,
}

// The userTokenCache stores rancher auth tokens for a given user if it exists
// This reuses tokens when possible instead of creating a new one every reconcile loop
var userTokenCache = make(map[string]string)
var userLock = &sync.RWMutex{}

// requestSender is an interface for sending requests to Rancher that allows us to mock during unit testing
type requestSender interface {
	Do(httpClient *http.Client, req *http.Request) (*http.Response, error)
}

// HTTPRequestSender is an implementation of requestSender that uses http.Client to send requests
type HTTPRequestSender struct{}

// RancherHTTPClient will be replaced with a mock in unit tests
var RancherHTTPClient requestSender = &HTTPRequestSender{}

// Do is a function that simply delegates sending the request to the http.Client
func (*HTTPRequestSender) Do(httpClient *http.Client, req *http.Request) (*http.Response, error) {
	return httpClient.Do(req)
}

// NewAdminRancherConfig creates A rancher config that authenticates with the admin user
func NewAdminRancherConfig(rdr client.Reader, host string, log vzlog.VerrazzanoLogger) (*RancherConfig, error) {
	secret, err := GetAdminSecret(rdr)
	if err != nil {
		return nil, log.ErrorfNewErr("Failed to get the admin secret from the cluster: %v", err)
	}
	return NewRancherConfigForUser(rdr, rancherAdminUsername, secret, host, log)
}

// NewVerrazzanoClusterRancherConfig creates A rancher config that authenticates with the Verrazzano cluster user
func NewVerrazzanoClusterRancherConfig(rdr client.Reader, host string, log vzlog.VerrazzanoLogger) (*RancherConfig, error) {
	secret, err := GetVerrazzanoClusterUserSecret(rdr)
	if err != nil {
		return nil, log.ErrorfNewErr("Failed to get the Verrazzano cluster secret from the cluster: %v", err)
	}
	return NewRancherConfigForUser(rdr, cons.VerrazzanoClusterRancherUsername, secret, host, log)
}

// NewRancherConfigForUser returns a populated RancherConfig struct that can be used to make calls to the Rancher API
func NewRancherConfigForUser(rdr client.Reader, username, password, host string, log vzlog.VerrazzanoLogger) (*RancherConfig, error) {
	rc := &RancherConfig{BaseURL: "https://" + host}
	// Needed to populate userToken[] map
	rc.User = username

	// Rancher host name is needed for TLS
	log.Debug("Getting Rancher ingress host name")
	hostname, err := getRancherIngressHostname(rdr)
	if err != nil {
		log.Errorf("Failed to get Rancher ingress host name: %v", err)
		return nil, err
	}
	rc.Host = hostname

	log.Debug("Getting Rancher TLS root CA")
	caCert, err := common.GetRootCA(rdr)
	if err != nil {
		log.Errorf("Failed to get Rancher TLS root CA: %v", err)
		return nil, err
	}
	rc.CertificateAuthorityData = caCert

	log.Debugf("Checking for Rancher additional CA in secret %s", cons.AdditionalTLS)
	rc.AdditionalCA = common.GetAdditionalCA(rdr)

	token, exists := getStoredToken(username)
	if !exists {
		token, err = getUserToken(rc, log, password, username)
		if err != nil {
			return nil, err
		}
		newStoredToken(username, token)
	}

	rc.APIAccessToken = token
	return rc, nil
}

// newStoredToken creates a new user:token key-pair in memory
func newStoredToken(username string, token string) {
	userLock.Lock()
	defer userLock.Unlock()
	userTokenCache[username] = token
}

// getStoredToken gets the token for the given user from memory
func getStoredToken(username string) (string, bool) {
	userLock.RLock()
	defer userLock.RUnlock()
	token, exists := userTokenCache[username]
	return token, exists
}

// deleteStoredToken deletes the token for the given user from memory
func deleteStoredToken(username string) {
	userLock.Lock()
	defer userLock.Unlock()
	delete(userTokenCache, username)
}

// DeleteStoredTokens clears the map of stored tokens.
func DeleteStoredTokens() {
	userLock.Lock()
	defer userLock.Unlock()
	userTokenCache = make(map[string]string)
}

// getRancherIngressHostname gets the Rancher ingress host name. This is used to set the host for TLS.
func getRancherIngressHostname(rdr client.Reader) (string, error) {
	ingress := &k8net.Ingress{}
	nsName := types.NamespacedName{
		Namespace: rancherNamespace,
		Name:      rancherIngressName}
	if err := rdr.Get(context.TODO(), nsName, ingress); err != nil {
		return "", fmt.Errorf("Failed to get Rancher ingress %v: %v", nsName, err)
	}

	if len(ingress.Spec.Rules) > 0 {
		// the first host will do
		return ingress.Spec.Rules[0].Host, nil
	}

	return "", fmt.Errorf("Failed, Rancher ingress %v is missing host names", nsName)
}

// GetVerrazzanoClusterUserSecret fetches the Rancher Verrazzano user secret
func GetVerrazzanoClusterUserSecret(rdr client.Reader) (string, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      cons.VerrazzanoClusterRancherName}

	if err := rdr.Get(context.TODO(), nsName, secret); err != nil {
		return "", err
	}
	return string(secret.Data["password"]), nil
}

// GetAdminSecret fetches the Rancher admin secret
func GetAdminSecret(rdr client.Reader) (string, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: rancherNamespace,
		Name:      rancherAdminSecret}

	if err := rdr.Get(context.TODO(), nsName, secret); err != nil {
		return "", err
	}
	return string(secret.Data["password"]), nil
}

// getUserToken gets a user token from a secret
func getUserToken(rc *RancherConfig, log vzlog.VerrazzanoLogger, secret, username string) (string, error) {
	action := http.MethodPost
	payload := `{"Username": "` + username + `", "Password": "` + secret + `"}`
	reqURL := rc.BaseURL + loginPath
	headers := map[string]string{"Content-Type": "application/json"}

	response, responseBody, err := SendRequest(action, reqURL, headers, payload, rc, log)
	if err != nil {
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		return "", err
	}

	return httputil.ExtractFieldFromResponseBodyOrReturnError(responseBody, "token", "unable to find token in Rancher response")
}

type Payload struct {
	ClusterID string `json:"clusterID"`
	TTL       int    `json:"ttl"`
}

type TokenPostResponse struct {
	Token   string `json:"token"`
	Created string `json:"created"`
}

// CreateTokenWithTTL creates a user token with ttl (in minutes)
func CreateTokenWithTTL(rc *RancherConfig, log vzlog.VerrazzanoLogger, ttl, clusterID string) (string, string, error) {
	val, _ := strconv.Atoi(ttl)
	payload := &Payload{
		ClusterID: clusterID,
		TTL:       val * 60000,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	action := http.MethodPost
	reqURL := rc.BaseURL + tokensPath
	headers := map[string]string{"Authorization": "Bearer " + rc.APIAccessToken, "Content-Type": "application/json"}

	response, responseBody, err := SendRequest(action, reqURL, headers, string(data), rc, log)
	if err != nil {
		return "", "", err
	}
	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		return "", "", log.ErrorfNewErr("Failed to validate response: %v", err)
	}

	var tokenPostResponse TokenPostResponse
	err = json.Unmarshal([]byte(responseBody), &tokenPostResponse)
	if err != nil {
		return "", "", log.ErrorfNewErr("Failed to parse response: %v", err)
	}

	return tokenPostResponse.Token, tokenPostResponse.Created, nil
}

type TokenGetResponse struct {
	Created   string `json:"created"`
	ClusterID string `json:"clusterId"`
	ExpiresAt string `json:"expiresAt"`
}

// GetTokenWithFilter get created expiresAt attribute of a user token with filter
func GetTokenWithFilter(rc *RancherConfig, log vzlog.VerrazzanoLogger, userID, clusterID string) (string, string, error) {
	action := http.MethodGet
	reqURL := rc.BaseURL + tokensPath + "?userId=" + url.PathEscape(userID) + "&clusterId=" + url.PathEscape(clusterID)
	headers := map[string]string{"Authorization": "Bearer " + rc.APIAccessToken}

	response, responseBody, err := SendRequest(action, reqURL, headers, "", rc, log)
	if err != nil {
		return "", "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		return "", "", log.ErrorfNewErr("Failed to validate response: %v", err)
	}

	data, err := httputil.ExtractFieldFromResponseBodyOrReturnError(responseBody, "data", "failed to locate the data field of the response body")
	if err != nil {
		return "", "", log.ErrorfNewErr("Failed to find data in Rancher response: %v", err)
	}

	var items []TokenGetResponse
	json.Unmarshal([]byte(data), &items)
	if err != nil {
		return "", "", log.ErrorfNewErr("Failed to parse response: %v", err)
	}
	for _, item := range items {
		if item.ClusterID == clusterID {
			return item.Created, item.ExpiresAt, nil
		}
	}
	return "", "", log.ErrorfNewErr("Failed to find the token in Rancher response")
}

// getProxyURL returns an HTTP proxy from the environment if one is set, otherwise an empty string
func getProxyURL() string {
	if proxyURL := os.Getenv("https_proxy"); proxyURL != "" {
		return proxyURL
	}
	if proxyURL := os.Getenv("HTTPS_PROXY"); proxyURL != "" {
		return proxyURL
	}
	if proxyURL := os.Getenv("http_proxy"); proxyURL != "" {
		return proxyURL
	}
	if proxyURL := os.Getenv("HTTP_PROXY"); proxyURL != "" {
		return proxyURL
	}
	return ""
}

// SendRequest builds an HTTP request, sends it, and returns the response
func SendRequest(action string, reqURL string, headers map[string]string, payload string,
	rc *RancherConfig, log vzlog.VerrazzanoLogger) (*http.Response, string, error) {

	req, err := http.NewRequest(action, reqURL, strings.NewReader(payload))
	if err != nil {
		return nil, "", err
	}

	req.Header.Add("Accept", "*/*")

	for k := range headers {
		req.Header.Add(k, headers[k])
	}
	req.Header.Add("Host", rc.Host)
	req.Host = rc.Host

	response, body, err := doRequest(req, rc, log)
	// If we get an unauthorized response, remove the token from the cache
	if response != nil && response.StatusCode == http.StatusUnauthorized {
		deleteStoredToken(rc.User)
	}
	return response, body, err
}

// doRequest configures an HTTP transport (including TLS), sends an HTTP request with retries, and returns the response
func doRequest(req *http.Request, rc *RancherConfig, log vzlog.VerrazzanoLogger) (*http.Response, string, error) {
	log.Debugf("Attempting HTTP request: %v", req)

	proxyURL := getProxyURL()

	var tlsConfig *tls.Config
	if len(rc.CertificateAuthorityData) < 1 && len(rc.AdditionalCA) < 1 {
		tlsConfig = &tls.Config{
			ServerName: rc.Host,
			MinVersion: tls.VersionTLS12,
		}
	} else {
		tlsConfig = &tls.Config{
			RootCAs:    common.CertPool(rc.CertificateAuthorityData, rc.AdditionalCA),
			ServerName: rc.Host,
			MinVersion: tls.VersionTLS12,
		}
	}
	tr := &http.Transport{
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// if we have a proxy, then set it in the transport
	if proxyURL != "" {
		u := url.URL{}
		proxy, err := u.Parse(proxyURL)
		if err != nil {
			return nil, "", err
		}
		tr.Proxy = http.ProxyURL(proxy)
	}

	client := &http.Client{Transport: tr, Timeout: 30 * time.Second}
	var resp *http.Response
	var err error

	// resp.Body is consumed by the first try, and then no longer available (empty)
	// so we need to read the body and save it so we can use it in each retry
	buffer, _ := io.ReadAll(req.Body)

	common.Retry(DefaultRetry, log, true, func() (bool, error) {
		// update the body with the saved data to prevent the "zero length body" error
		req.Body = io.NopCloser(bytes.NewBuffer(buffer))
		resp, err = RancherHTTPClient.Do(client, req)

		// check for a network error and retry
		if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
			log.Infof("Temporary error executing HTTP request %v : %v, retrying", req, nerr)
			return false, err
		}

		// if err is another kind of network error that is not considered "temporary", then retry
		if err, ok := err.(*url.Error); ok {
			if err, ok := err.Err.(*net.OpError); ok {
				if derr, ok := err.Err.(*net.DNSError); ok {
					log.Infof("DNS error: %v, retrying", derr)
					return false, err
				}
			}
		}

		// retry any HTTP 500 errors
		if resp != nil && resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			log.ErrorfThrottled("HTTP status %v executing HTTP request %v, retrying", resp.StatusCode, req)
			return false, err
		}

		// if err is some other kind of unexpected error, retry
		if err != nil {
			return false, err
		}
		return true, err
	})

	if err != nil {
		return resp, "", err
	}
	defer resp.Body.Close()

	// extract the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return resp, string(body), err
}

// RancherIngressServiceHost returns the internal service host name of the Rancher ingress
func RancherIngressServiceHost() string {
	return DefaultRancherIngressHostPrefix + nginxutil.IngressNGINXNamespace()
}
