// Copyright (c) 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package rancherutil

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
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

	rancherAdminSecret = "rancher-admin-secret" //nolint:gosec //#gosec G101

	loginPath = "/v3-public/localProviders/local?action=login"

	// this host resolves to the cluster IP
	nginxIngressHostName = "ingress-controller-ingress-nginx-controller.ingress-nginx"
)

type RancherConfig struct {
	Host                     string
	BaseURL                  string
	APIAccessToken           string
	CertificateAuthorityData []byte
	AdditionalCA             []byte
}

var DefaultRetry = wait.Backoff{
	Steps:    10,
	Duration: 1 * time.Second,
	Factor:   2.0,
	Jitter:   0.1,
}

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

// NewRancherConfig returns a populated RancherConfig struct that can be used to make calls to the Rancher API
func NewRancherConfig(rdr client.Reader, admin bool, log vzlog.VerrazzanoLogger) (*RancherConfig, error) {
	rc := &RancherConfig{BaseURL: "https://" + nginxIngressHostName}

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

	if admin {
		log.Once("Getting admin token from Rancher")
		token, err := getAdminTokenFromRancher(rdr, rc, log)
		if err != nil {
			log.ErrorfThrottled("Failed to get admin token from Rancher: %v", err)
			return nil, err
		}
		rc.APIAccessToken = token
		return rc, nil
	}

	log.Once("Getting Verrazzano cluster user token from Rancher")
	token, err := getVerrazzanoClusterUserTokenFromRancher(rdr, rc, log)
	if err != nil {
		log.ErrorfThrottled("Failed to get Verrazzano cluster user token from Rancher: %v", err)
		return nil, err
	}
	rc.APIAccessToken = token
	return rc, nil
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

// getVerrazzanoClusterUserSecret fetches the Rancher Verrazzano user secret
func getVerrazzanoClusterUserSecret(rdr client.Reader) (string, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: constants.VerrazzanoMultiClusterNamespace,
		Name:      cons.VerrazzanoClusterRancherName}

	if err := rdr.Get(context.TODO(), nsName, secret); err != nil {
		return "", err
	}
	return string(secret.Data["password"]), nil
}

// getAdminSecret fetches the Rancher admin secret
func getAdminSecret(rdr client.Reader) (string, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: rancherNamespace,
		Name:      rancherAdminSecret}

	if err := rdr.Get(context.TODO(), nsName, secret); err != nil {
		return "", err
	}
	return string(secret.Data["password"]), nil
}

// getVerrazzanoClusterUserTokenFromRancher does a login with the verrazzano user so Rancher and returns the token from the response
func getVerrazzanoClusterUserTokenFromRancher(rdr client.Reader, rc *RancherConfig, log vzlog.VerrazzanoLogger) (string, error) {
	secret, err := getVerrazzanoClusterUserSecret(rdr)
	if err != nil {
		return "", err
	}
	return getUserToken(rc, log, secret)
}

// getAdminTokenFromRancher does a login with the admin user in Rancher and returns the token from the response
func getAdminTokenFromRancher(rdr client.Reader, rc *RancherConfig, log vzlog.VerrazzanoLogger) (string, error) {
	secret, err := getAdminSecret(rdr)
	if err != nil {
		return "", err
	}
	return getUserToken(rc, log, secret)
}

// getUserToken gets a user token from a secret
func getUserToken(rc *RancherConfig, log vzlog.VerrazzanoLogger, secret string) (string, error) {
	action := http.MethodPost
	payload := `{"Username": "` + cons.VerrazzanoClusterRancherUsername + `", "Password": "` + secret + `"}`
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

	return doRequest(req, rc, log)
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
