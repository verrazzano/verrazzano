// Copyright (c) 2021, 2022, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package clusters

import (
	"bytes"
	"context"
	"crypto/md5" //nolint:gosec //#gosec G501 // package used for caching only, not security
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/Jeffail/gabs/v2"
	cons "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
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
	rancherTLSSecret   = "tls-rancher-ingress"  //nolint:gosec //#gosec G101

	clusterPath         = "/v3/cluster"
	clustersPath        = "/v3/clusters"
	clustersByNamePath  = "/v3/clusters?name="
	clusterRegTokenPath = "/v3/clusterregistrationtoken" //nolint:gosec //#gosec G101
	manifestPath        = "/v3/import/"
	loginPath           = "/v3-public/localProviders/local?action=login"

	k8sClustersPath = "/k8s/clusters/"

	// this host resolves to the cluster IP
	nginxIngressHostName = "ingress-controller-ingress-nginx-controller.ingress-nginx"

	rancherClusterStateActive = "active"
)

type rancherConfig struct {
	host                     string
	baseURL                  string
	apiAccessToken           string
	certificateAuthorityData []byte
	additionalCA             []byte
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

// Do is a function that simply delegates sending the request to the http.Client
func (*httpRequestSender) Do(httpClient *http.Client, req *http.Request) (*http.Response, error) {
	return httpClient.Do(req)
}

// rancherHTTPClient will be replaced with a mock in unit tests
var rancherHTTPClient requestSender = &httpRequestSender{}

// registerManagedClusterWithRancher registers a managed cluster with Rancher and returns a chunk of YAML that
// must be applied on the managed cluster to complete the registration.
func registerManagedClusterWithRancher(rc *rancherConfig, clusterName string, rancherClusterID string, log vzlog.VerrazzanoLogger) (string, string, error) {
	clusterID := rancherClusterID
	var err error
	if clusterID == "" {
		log.Oncef("Registering managed cluster in Rancher with name: %s", clusterName)
		clusterID, err = importClusterToRancher(rc, clusterName, log)
		if err != nil {
			log.Errorf("Failed to import cluster to Rancher: %v", err)
			return "", "", err
		}
	}

	log.Once("Getting registration YAML from Rancher")
	regYAML, err := getRegistrationYAMLFromRancher(rc, clusterID, log)
	if err != nil {
		log.Errorf("Failed to get registration YAML from Rancher: %v", err)
		return "", "", err
	}

	return regYAML, clusterID, nil
}

// newRancherConfig returns a populated rancherConfig struct that can be used to make calls to the Rancher API
func newRancherConfig(rdr client.Reader, log vzlog.VerrazzanoLogger) (*rancherConfig, error) {
	rc := &rancherConfig{baseURL: "https://" + nginxIngressHostName}

	// Rancher host name is needed for TLS
	log.Debug("Getting Rancher ingress host name")
	hostname, err := getRancherIngressHostname(rdr)
	if err != nil {
		log.Errorf("Failed to get Rancher ingress host name: %v", err)
		return nil, err
	}
	rc.host = hostname

	log.Debug("Getting Rancher TLS root CA")
	caCert, err := common.GetRootCA(rdr)
	if err != nil {
		log.Errorf("Failed to get Rancher TLS root CA: %v", err)
		return nil, err
	}
	rc.certificateAuthorityData = caCert

	log.Debugf("Checking for Rancher additional CA in secret %s", cons.AdditionalTLS)
	rc.additionalCA = common.GetAdditionalCA(rdr)

	log.Once("Getting admin token from Rancher")
	adminToken, err := getAdminTokenFromRancher(rdr, rc, log)
	if err != nil {
		log.Errorf("Failed to get admin token from Rancher: %v", err)
		return nil, err
	}
	rc.apiAccessToken = adminToken

	return rc, nil
}

// importClusterToRancher uses the Rancher API to import the cluster. The cluster will show as "pending" until the registration
// YAML is applied on the managed cluster.
func importClusterToRancher(rc *rancherConfig, clusterName string, log vzlog.VerrazzanoLogger) (string, error) {
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
		log.Debugf("Cluster %s already registered with Rancher, attempting to fetch it", clusterName)
		clusterID, err := getClusterIDFromRancher(rc, clusterName, log)
		if err != nil {
			return "", err
		}
		return clusterID, nil
	}

	if err != nil {
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		return "", err
	}
	log.Oncef("Successfully registered managed cluster in Rancher with name: %s", clusterName)

	return httputil.ExtractFieldFromResponseBodyOrReturnError(responseBody, "id", "unable to find cluster id in Rancher response")
}

// getClusterIDFromRancher attempts to fetch the cluster from Rancher by name and pull out the cluster ID
func getClusterIDFromRancher(rc *rancherConfig, clusterName string, log vzlog.VerrazzanoLogger) (string, error) {
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

	return httputil.ExtractFieldFromResponseBodyOrReturnError(responseBody, "data.0.id", "unable to find clusterId in Rancher response")
}

// getAllClustersInRancher returns a slice of the cluster names registered with Rancher
func getAllClustersInRancher(rc *rancherConfig, log vzlog.VerrazzanoLogger) ([]string, []byte, error) {
	reqURL := rc.baseURL + clustersPath
	headers := map[string]string{"Authorization": "Bearer " + rc.apiAccessToken}

	hash := md5.New() //nolint:gosec //#gosec G401
	clusterNames := []string{}
	for {
		response, responseBody, err := sendRequest(http.MethodGet, reqURL, headers, "", rc, log)
		if response != nil && response.StatusCode != http.StatusOK {
			return nil, nil, fmt.Errorf("Unable to get clusters from Rancher, response code: %d", response.StatusCode)
		}

		if err != nil {
			return nil, nil, err
		}

		// parse the response and iterate over the items
		jsonString, err := gabs.ParseJSON([]byte(responseBody))
		if err != nil {
			return nil, nil, err
		}

		var items []interface{}
		var ok bool
		if items, ok = jsonString.Path("data").Data().([]interface{}); !ok {
			return nil, nil, fmt.Errorf("Unable to find expected data in Rancher clusters response: %v", jsonString)
		}

		for _, item := range items {
			i := item.(map[string]interface{})
			if name, ok := i["name"]; ok {
				clusterNames = append(clusterNames, name.(string))
			}
		}

		// add this response body to the hash
		io.WriteString(hash, responseBody)

		// if there is a "next page" link then use that to make another request
		if reqURL, err = httputil.ExtractFieldFromResponseBodyOrReturnError(responseBody, "pagination.next", ""); err != nil {
			break
		}
	}

	// unfortunately Rancher does not support ETags, so we return a hash of the response bodies which allows the caller to know if
	// there were any changes to the clusters
	return clusterNames, hash.Sum(nil), nil
}

// isManagedClusterActiveInRancher returns true if the managed cluster is active
func isManagedClusterActiveInRancher(rc *rancherConfig, clusterID string, log vzlog.VerrazzanoLogger) (bool, error) {
	reqURL := rc.baseURL + clustersPath + "/" + clusterID
	headers := map[string]string{"Authorization": "Bearer " + rc.apiAccessToken}

	response, responseBody, err := sendRequest(http.MethodGet, reqURL, headers, "", rc, log)

	if response != nil && response.StatusCode != http.StatusOK {
		return false, fmt.Errorf("tried to get cluster from Rancher but failed, response code: %d", response.StatusCode)
	}

	if err != nil {
		return false, err
	}

	state, err := httputil.ExtractFieldFromResponseBodyOrReturnError(responseBody, "state", "unable to find cluster state in Rancher response")
	if err != nil {
		return false, err
	}
	agentImage, err := httputil.ExtractFieldFromResponseBodyOrReturnError(responseBody, "agentImage", "unable to find agent image in Rancher response")
	if err != nil {
		return false, err
	}

	// Rancher temporarily sets the state of a new cluster to "active" before setting it to "pending", so we also check for the "agentImage" field
	// to know that the cluster is really active
	return state == rancherClusterStateActive && len(agentImage) > 0, nil
}

// getCACertFromManagedCluster attempts to get the CA cert from the managed cluster using the Rancher API proxy. It first checks for
// the Rancher TLS secret and if that is not found it looks for the Verrazzano system TLS secret.
func getCACertFromManagedCluster(rc *rancherConfig, clusterID string, log vzlog.VerrazzanoLogger) (string, error) {
	// first look for the Rancher TLS secret
	caCert, err := getCACertFromManagedClusterSecret(rc, clusterID, rancherNamespace, cons.AdditionalTLS, cons.AdditionalTLSCAKey, log)
	if err != nil {
		return "", err
	}

	if caCert != "" {
		return caCert, nil
	}

	// didn't find the Rancher secret so next look for the verrazzano-tls secret
	caCert, err = getCACertFromManagedClusterSecret(rc, clusterID, cons.VerrazzanoSystemNamespace, constants.VerrazzanoIngressSecret, mcconstants.CaCrtKey, log)
	if err != nil {
		return "", err
	}

	if caCert != "" {
		return caCert, nil
	}

	return "", nil
}

// getCACertFromManagedClusterSecret attempts to get the CA cert from a secret on the managed cluster using the Rancher API proxy
func getCACertFromManagedClusterSecret(rc *rancherConfig, clusterID, namespace, secretName, secretKey string, log vzlog.VerrazzanoLogger) (string, error) {
	const k8sAPISecretPattern = "%s/api/v1/namespaces/%s/secrets/%s" //nolint:gosec //#gosec G101

	// use the Rancher API proxy on the managed cluster to fetch the secret
	baseReqURL := rc.baseURL + k8sClustersPath + clusterID
	headers := map[string]string{"Authorization": "Bearer " + rc.apiAccessToken}

	reqURL := fmt.Sprintf(k8sAPISecretPattern, baseReqURL, namespace, secretName)
	response, responseBody, err := sendRequest(http.MethodGet, reqURL, headers, "", rc, log)

	if response != nil {
		if response.StatusCode == http.StatusNotFound {
			return "", nil
		}
		if response.StatusCode != http.StatusOK {
			return "", fmt.Errorf("tried to get managed cluster CA cert %s/%s from Rancher but failed, response code: %d", namespace, secretName, response.StatusCode)
		}
	}
	if err != nil {
		return "", err
	}

	// parse the response and pull out the secretKey value from the secret data
	jsonString, err := gabs.ParseJSON([]byte(responseBody))
	if err != nil {
		return "", err
	}

	if data, ok := jsonString.Path("data").Data().(map[string]interface{}); ok {
		if caCert, ok := data[secretKey].(string); ok {
			return caCert, nil
		}
	}

	return "", nil
}

// getRegistrationYAMLFromRancher creates a registration token in Rancher for the managed cluster and uses the
// returned token to fetch the registration (manifest) YAML.
func getRegistrationYAMLFromRancher(rc *rancherConfig, rancherClusterID string, log vzlog.VerrazzanoLogger) (string, error) {
	action := http.MethodPost
	payload := `{"type": "clusterRegistrationToken", "clusterId": "` + rancherClusterID + `"}`
	reqURL := rc.baseURL + clusterRegTokenPath
	headers := map[string]string{"Content-Type": "application/json"}
	headers["Authorization"] = "Bearer " + rc.apiAccessToken

	response, manifestContent, err := sendRequest(action, reqURL, headers, payload, rc, log)

	if err != nil {
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		return "", err
	}

	// get the manifest token from the response, construct a URL, and fetch its contents
	token, err := httputil.ExtractFieldFromResponseBodyOrReturnError(manifestContent, "token", "unable to find manifest token in Rancher response")
	if err != nil {
		return "", err
	}

	// Rancher 2.5.x added the cluster ID to the manifest URL.
	manifestURL := rc.baseURL + manifestPath + token + "_" + rancherClusterID + ".yaml"

	action = http.MethodGet
	response, manifestContent, err = sendRequest(action, manifestURL, headers, "", rc, log)

	if err != nil {
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		return "", err
	}

	return manifestContent, nil
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

// getAdminTokenFromRancher does a login with Rancher and returns the token from the response
func getAdminTokenFromRancher(rdr client.Reader, rc *rancherConfig, log vzlog.VerrazzanoLogger) (string, error) {
	secret, err := getAdminSecret(rdr)
	if err != nil {
		return "", err
	}

	action := http.MethodPost
	payload := `{"Username": "admin", "Password": "` + secret + `"}`
	reqURL := rc.baseURL + loginPath
	headers := map[string]string{"Content-Type": "application/json"}

	response, responseBody, err := sendRequest(action, reqURL, headers, payload, rc, log)
	if err != nil {
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusCreated)
	if err != nil {
		return "", err
	}

	return httputil.ExtractFieldFromResponseBodyOrReturnError(responseBody, "token", "unable to find token in Rancher response")
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

// sendRequest builds an HTTP request, sends it, and returns the response
func sendRequest(action string, reqURL string, headers map[string]string, payload string,
	rc *rancherConfig, log vzlog.VerrazzanoLogger) (*http.Response, string, error) {

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

// doRequest configures an HTTP transport (including TLS), sends an HTTP request with retries, and returns the response
func doRequest(req *http.Request, rc *rancherConfig, log vzlog.VerrazzanoLogger) (*http.Response, string, error) {
	log.Debugf("Attempting HTTP request: %v", req)

	proxyURL := getProxyURL()

	var tlsConfig *tls.Config
	if len(rc.certificateAuthorityData) < 1 && len(rc.additionalCA) < 1 {
		tlsConfig = &tls.Config{
			ServerName: rc.host,
			MinVersion: tls.VersionTLS12,
		}
	} else {
		tlsConfig = &tls.Config{
			RootCAs:    common.CertPool(rc.certificateAuthorityData, rc.additionalCA),
			ServerName: rc.host,
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

	retry(defaultRetry, log, func() (bool, error) {
		// update the body with the saved data to prevent the "zero length body" error
		req.Body = io.NopCloser(bytes.NewBuffer(buffer))
		resp, err = rancherHTTPClient.Do(client, req)

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
			log.Infof("HTTP status %v executing HTTP request %v, retrying", resp.StatusCode, req)
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}

	return resp, string(body), err
}

// retry executes the provided function repeatedly, retrying until the function
// returns done = true, or exceeds the given timeout.
// errors will be logged, but will not trigger retry to stop
func retry(backoff wait.Backoff, log vzlog.VerrazzanoLogger, fn wait.ConditionFunc) error {
	var lastErr error
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		done, err := fn()
		lastErr = err
		if err != nil {
			log.Infof("Retrying after error: %v", err)
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
