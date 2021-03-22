// Copyright (c) 2021, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	gabs "github.com/Jeffail/gabs/v2"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	k8net "k8s.io/api/networking/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	rancherNamespace   = "cattle-system"
	rancherIngressName = "rancher"
	rancherAdminSecret = "rancher-admin-secret"
	rancherTLSSecret   = "tls-rancher-ingress"

	clusterPath         = "/v3/cluster"
	clustersByNamePath  = "/v3/clusters?name="
	clusterRegTokenPath = "/v3/clusterregistrationtoken"
	manifestPath        = "/v3/import/"
	loginPath           = "/v3-public/localProviders/local?action=login"

	nginxNamespace      = "ingress-nginx"
	nginxIngressService = "ingress-controller-ingress-nginx-controller"
)

type rancherConfig struct {
	host                     string
	hostIP                   string
	hostPort                 int32
	baseURL                  string
	apiAccessToken           string
	certificateAuthorityData []byte
}

var defaultRetry = wait.Backoff{
	Steps:    10,
	Duration: 1 * time.Second,
	Factor:   2.0,
	Jitter:   0.1,
}

// requestSender is an interface for sending requests to Rancher that allows us to mock during unit testing
type requestSender interface {
	Do(httpClient *http.Client, req *http.Request) (*http.Response, error)
}

// httpRequestSender is an implementation of requestSender that uses http.Client to send requests
type httpRequestSender struct{}

// do is a function that simply delegates sending the request to the http.Client
func (*httpRequestSender) Do(httpClient *http.Client, req *http.Request) (*http.Response, error) {
	return httpClient.Do(req)
}

// rancherHTTPClient will be replaced with a mock in unit tests
var rancherHTTPClient requestSender = &httpRequestSender{}

// registerManagedClusterWithRancher registers a managed cluster with Rancher and returns a chunk of YAML that
// must be applied on the managed cluster to complete the registration.
func registerManagedClusterWithRancher(rdr client.Reader, clusterName string, log *zap.SugaredLogger) (string, error) {
	log.Infof("Registering managed cluster in Rancher with name: %s", clusterName)

	rc := &rancherConfig{}

	log.Debug("Getting ingress host IP")
	hostIP, err := getIngressHostIP(rdr)
	if err != nil {
		log.Errorf("Unable to get ingress host IP: %v", err)
		return "", err
	}
	rc.hostIP = hostIP

	log.Debug("Getting ingress host port")
	hostPort, err := getIngressHostPort(rdr)
	if err != nil {
		log.Errorf("Unable to get ingress host port: %v", err)
		return "", err
	}
	rc.hostPort = hostPort

	log.Debug("Getting Rancher ingress host name")
	hostname, err := getRancherIngressHostname(rdr)
	if err != nil {
		log.Errorf("Unable to get Rancher ingress host name: %v", err)
		return "", err
	}
	rc.host = hostname
	rc.baseURL = "https://" + rc.hostIP + ":" + strconv.Itoa(int(rc.hostPort))

	log.Debug("Getting Rancher TLS root CA")
	caCert, err := getRancherTLSRootCA(rdr)
	if err != nil {
		log.Errorf("Unable to get rancher TLS root CA: %v", err)
		return "", err
	}
	rc.certificateAuthorityData = caCert

	log.Debug("Getting admin token from Rancher")
	adminToken, err := getAdminTokenFromRancher(rdr, rc, log)
	if err != nil {
		log.Errorf("Unable to get admin token from Rancher: %v", err)
		return "", err
	}
	rc.apiAccessToken = adminToken

	log.Debug("Importing cluster to Rancher")
	clusterID, err := importClusterToRancher(rc, clusterName, log)
	if err != nil {
		log.Errorf("Unable to import cluster to Rancher: %v", err)
		return "", err
	}

	log.Debug("Getting registration YAML from Rancher")
	regYAML, err := getRegistrationYAMLFromRancher(rc, clusterName, clusterID, log)
	if err != nil {
		log.Errorf("Unable to get registration YAML from Rancher: %v", err)
		return "", err
	}

	log.Infof("Successfully registered managed cluster in Rancher with name: %s", clusterName)
	return regYAML, nil
}

// importClusterToRancher uses the Rancher API to import the cluster. The cluster will show as "pending" until the registration
// YAML is applied on the managed cluster.
func importClusterToRancher(rc *rancherConfig, clusterName string, log *zap.SugaredLogger) (string, error) {
	action := http.MethodPost
	payload := `{"type": "cluster",
		"name":"` + clusterName + `",
		"dockerRootDir": "/var/lib/docker",
		"enableClusterAlerting": "false",
		"enableClusterMonitoring": "false",
		"enableNetworkPolicy": "false"}`
	reqURL := rc.baseURL + clusterPath
	headers := map[string]string{"Content-Type": "application/json"}
	headers["Authorization"] = "Bearer " + rc.apiAccessToken

	response, responseBody, err := sendRequest(action, reqURL, headers, payload, rc, log)

	if response != nil && response.StatusCode == http.StatusUnprocessableEntity {
		// if we've already imported this cluster, we get an HTTP 422, so attempt to fetch the existing cluster
		// and get the cluster ID from the response
		log.Infof("Cluster %s already registered with Rancher, attempting to fetch it", clusterName)
		clusterID, err := getClusterIDFromRancher(rc, clusterName, log)
		if err != nil {
			return "", err
		}
		return clusterID, nil
	}

	if response != nil && response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("expected response code %d from POST but got %d: %v", http.StatusCreated, response.StatusCode, response)
	}
	if err != nil {
		return "", err
	}

	jsonString, err := gabs.ParseJSON([]byte(responseBody))
	if err != nil {
		return "", err
	}

	if clusterID, ok := jsonString.Path("id").Data().(string); ok {
		return clusterID, nil
	}

	return "", errors.New("unable to find cluster id in Rancher response")
}

// getClusterIDFromRancher attempts to fetch the cluster from Rancher by name and pull out the cluster ID
func getClusterIDFromRancher(rc *rancherConfig, clusterName string, log *zap.SugaredLogger) (string, error) {
	action := http.MethodGet

	reqURL := rc.baseURL + clustersByNamePath + clusterName
	headers := map[string]string{"Authorization": "Bearer " + rc.apiAccessToken}

	response, responseBody, err := sendRequest(action, reqURL, headers, "", rc, log)

	if response != nil && response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tried to get cluster from Rancher but failed, response code: %d", response.StatusCode)
	}
	if err != nil {
		return "", err
	}

	// parse cluster ID from response
	jsonString, err := gabs.ParseJSON([]byte(responseBody))
	if err != nil {
		return "", err
	}

	// response data is an array, but it should only contain one item
	if clusterID, ok := jsonString.Path("data.0.id").Data().(string); ok {
		return clusterID, nil
	}

	return "", errors.New("unable to find clusterId in Rancher response")
}

// getRegistrationYAMLFromRancher creates a registration token in Rancher for the managed cluster and uses the
// returned token to fetch the registration (manifest) YAML.
func getRegistrationYAMLFromRancher(rc *rancherConfig, clusterName string, rancherClusterID string, log *zap.SugaredLogger) (string, error) {
	action := http.MethodPost
	payload := `{"type": "clusterRegistrationToken", "clusterId": "` + rancherClusterID + `"}`
	reqURL := rc.baseURL + clusterRegTokenPath
	headers := map[string]string{"Content-Type": "application/json"}
	headers["Authorization"] = "Bearer " + rc.apiAccessToken

	response, manifestContent, err := sendRequest(action, reqURL, headers, payload, rc, log)

	if response != nil && response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("expected response code %d from POST but got %d: %v", http.StatusCreated, response.StatusCode, response)
	}
	if err != nil {
		return "", err
	}

	jsonString, err := gabs.ParseJSON([]byte(manifestContent))
	if err != nil {
		return "", err
	}

	// get the manifest token from the response, construct a URL, and fetch its contents
	token, ok := jsonString.Path("token").Data().(string)
	if !ok {
		return "", errors.New("unable to find manifest token in Rancher response")
	}

	manifestURL := rc.baseURL + manifestPath + token + ".yaml"

	action = http.MethodGet
	response, manifestContent, err = sendRequest(action, manifestURL, headers, "", rc, log)

	if response != nil && response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("expected response code %d from GET but got %d: %v", http.StatusOK, response.StatusCode, response)
	}
	if err != nil {
		return "", err
	}

	return manifestContent, nil
}

// getAdminSecret fetches the Rancher admin secret
func getAdminSecret(rdr client.Reader, rc *rancherConfig) (string, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: rancherNamespace,
		Name:      rancherAdminSecret}

	if err := rdr.Get(context.TODO(), nsName, secret); err != nil {
		return "", err
	}
	return string(secret.Data["password"]), nil
}

// getAdminTokenFromRancher does a login with Rancher and returns the token from the response
func getAdminTokenFromRancher(rdr client.Reader, rc *rancherConfig, log *zap.SugaredLogger) (string, error) {
	secret, err := getAdminSecret(rdr, rc)
	if err != nil {
		return "", err
	}

	action := http.MethodPost
	payload := `{"Username": "admin", "Password": "` + secret + `"}`
	reqURL := rc.baseURL + loginPath
	headers := map[string]string{"Content-Type": "application/json"}

	response, responseBody, err := sendRequest(action, reqURL, headers, payload, rc, log)

	if response != nil && response.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("expected response code %d from POST but got %d: %v", http.StatusCreated, response.StatusCode, response)
	}
	if err != nil {
		return "", err
	}

	jsonString, err := gabs.ParseJSON([]byte(responseBody))
	if err != nil {
		return "", err
	}
	if token, ok := jsonString.Path("token").Data().(string); ok {
		return token, nil
	}

	return "", errors.New("unable to find token in Rancher response")
}

// getIngressHostIP gets the local ingress host IP address
func getIngressHostIP(rdr client.Reader) (string, error) {
	labels := client.MatchingLabels{"app.kubernetes.io/name": "ingress-nginx", "app.kubernetes.io/component": "controller"}
	pods := corev1.PodList{}
	if err := rdr.List(context.TODO(), &pods, client.InNamespace(nginxNamespace), labels); err != nil {
		return "", err
	}

	if len(pods.Items) > 0 {
		return pods.Items[0].Status.HostIP, nil
	}

	return "", errors.New("no matching nginx ingress pods found")
}

// getIngressHostPort gets the local ingress host port
func getIngressHostPort(rdr client.Reader) (int32, error) {
	service := &corev1.Service{}
	nsName := types.NamespacedName{
		Namespace: nginxNamespace,
		Name:      nginxIngressService}
	if err := rdr.Get(context.TODO(), nsName, service); err != nil {
		return 0, err
	}

	for _, port := range service.Spec.Ports {
		if port.Name == "https" {
			return port.NodePort, nil
		}
	}

	return 0, errors.New("no nginx ingress service ports found")
}

// getRancherIngressHostname gets the Rancher ingress host name. This is used to set the host for TLS.
func getRancherIngressHostname(rdr client.Reader) (string, error) {
	ingress := &k8net.Ingress{}
	nsName := types.NamespacedName{
		Namespace: rancherNamespace,
		Name:      rancherIngressName}
	if err := rdr.Get(context.TODO(), nsName, ingress); err != nil {
		return "", err
	}

	if len(ingress.Spec.Rules) > 0 {
		// the first host will do
		return ingress.Spec.Rules[0].Host, nil
	}

	return "", errors.New("unable to get rancher ingress host name")
}

// getRancherTLSRootCA gets the root CA certificate from the Rancher TLS secret. If the secret does not exist, we
// return a nil slice.
func getRancherTLSRootCA(rdr client.Reader) ([]byte, error) {
	secret := &corev1.Secret{}
	nsName := types.NamespacedName{
		Namespace: rancherNamespace,
		Name:      rancherTLSSecret}

	if err := rdr.Get(context.TODO(), nsName, secret); err != nil {
		return nil, client.IgnoreNotFound(err)
	}
	return secret.Data["ca.crt"], nil
}

// sendRequest builds an HTTP request, sends it, and returns the response
func sendRequest(action string, reqURL string, headers map[string]string, payload string,
	rc *rancherConfig, log *zap.SugaredLogger) (*http.Response, string, error) {

	req, err := http.NewRequest(action, reqURL, strings.NewReader(payload))
	if err != nil {
		return nil, "", err
	}

	req.Header.Add("Accept", "*/*")

	for k := range headers {
		req.Header.Add(k, headers[k])
	}
	req.Header.Add("Host", rc.host)
	req.Host = rc.host

	return doRequest(req, rc, log)
}

// newCertPool creates a CertPool given certificate bytes
func newCertPool(certData []byte) *x509.CertPool {
	if len(certData) == 0 {
		return nil
	}

	certPool := x509.NewCertPool()
	certPool.AppendCertsFromPEM(certData)
	return certPool
}

// doRequest configures an HTTP transport (including TLS), sends an HTTP request with retries, and returns the response
func doRequest(req *http.Request, rc *rancherConfig, log *zap.SugaredLogger) (*http.Response, string, error) {
	log.Debugf("Attempting HTTP request: %v", req)

	proxyURL := getProxyURL()

	tlsConfig := &tls.Config{
		RootCAs:    newCertPool(rc.certificateAuthorityData),
		ServerName: rc.host,
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
	buffer, _ := ioutil.ReadAll(req.Body)

	retry(defaultRetry, log, func() (bool, error) {
		// update the body with the saved data to prevent the "zero length body" error
		req.Body = ioutil.NopCloser(bytes.NewBuffer(buffer))
		resp, err = rancherHTTPClient.Do(client, req)

		// check for a network error and retry
		if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
			log.Warnf("Temporary error executing HTTP request %v : %v, retrying", req, nerr)
			return false, err
		}

		// if err is another kind of network error that is not considered "temporary", then retry
		if err, ok := err.(*url.Error); ok {
			if err, ok := err.Err.(*net.OpError); ok {
				if derr, ok := err.Err.(*net.DNSError); ok {
					log.Warnf("DNS error: %v, retrying", derr)
					return false, err
				}
			}
		}

		// retry any HTTP 500 errors
		if resp != nil && resp.StatusCode >= 500 && resp.StatusCode <= 599 {
			log.Warnf("Got HTTP status %v executing HTTP request %v, retrying", resp.StatusCode, req)
			return false, err
		}

		// if err is some other kind of unexpected error, retry
		if err != nil {
			return false, err
		}
		return true, err
	})

	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	// extract the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return resp, string(body), err
}

// retry executes the provided function repeatedly, retrying until the function
// returns done = true, or exceeds the given timeout.
// errors will be logged, but will not trigger retry to stop
func retry(backoff wait.Backoff, log *zap.SugaredLogger, fn wait.ConditionFunc) error {
	var lastErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		done, err := fn()
		lastErr = err
		if err != nil {
			log.Warnf("Retrying after error: %v", err)
		}
		return done, nil
	})
	if err == wait.ErrWaitTimeout {
		if lastErr != nil {
			err = lastErr
		}
	}
	return err
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
