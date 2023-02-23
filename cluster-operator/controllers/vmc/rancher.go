// Copyright (c) 2022, 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package vmc

import (
	"crypto/md5" //nolint:gosec //#gosec G501 // package used for caching only, not security
	"fmt"
	"github.com/Jeffail/gabs/v2"
	cons "github.com/verrazzano/verrazzano/pkg/constants"
	"github.com/verrazzano/verrazzano/pkg/httputil"
	"github.com/verrazzano/verrazzano/pkg/log/vzlog"
	"github.com/verrazzano/verrazzano/pkg/mcconstants"
	"github.com/verrazzano/verrazzano/pkg/rancherutil"
	"github.com/verrazzano/verrazzano/platform-operator/constants"
	"io"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/yaml"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	rancherNamespace   = "cattle-system"
	rancherIngressName = "rancher"
	rancherTLSSecret   = "tls-rancher-ingress" //nolint:gosec //#gosec G101

	clusterPath          = "/v3/cluster"
	clustersPath         = "/v3/clusters"
	clustersByNamePath   = "/v3/clusters?name="
	clusterRegTokenPath  = "/v3/clusterregistrationtoken" //nolint:gosec //#gosec G101
	manifestPath         = "/v3/import/"
	loginPath            = "/v3-public/localProviders/local?action=login"
	secretPathTemplate   = "/api/v1/namespaces/%s/secrets/%s" //nolint:gosec //#gosec G101
	secretCreateTemplate = "/api/v1/namespaces/%s/secrets"    //nolint:gosec //#gosec G101

	k8sClustersPath = "/k8s/clusters/"

	rancherClusterStateActive = "active"
)

type RancherCluster struct {
	Name string
	ID   string
}

// registerManagedClusterWithRancher registers a managed cluster with Rancher and returns a chunk of YAML that
// must be applied on the managed cluster to complete the registration.
func registerManagedClusterWithRancher(rc *rancherutil.RancherConfig, clusterName string, rancherClusterID string, log vzlog.VerrazzanoLogger) (string, string, error) {
	clusterID := rancherClusterID
	var err error
	if clusterID == "" {
		log.Oncef("Registering managed cluster in Rancher with name: %s", clusterName)
		clusterID, err = ImportClusterToRancher(rc, clusterName, nil, log)
		if err != nil {
			log.Errorf("Failed to import cluster to Rancher: %v", err)
			return "", "", err
		}
	}

	log.Oncef("Getting registration YAML from Rancher for cluster %s with id %s", clusterName, clusterID)
	regYAML, err := getRegistrationYAMLFromRancher(rc, clusterID, log)
	if err != nil {
		log.Errorf("Failed to get registration YAML from Rancher: %v", err)
		return "", "", err
	}

	return regYAML, clusterID, nil
}

// ImportClusterToRancher uses the Rancher API to import the cluster. The cluster will show as "pending" until the registration
// YAML is applied on the managed cluster.
func ImportClusterToRancher(rc *rancherutil.RancherConfig, clusterName string, labels map[string]string, log vzlog.VerrazzanoLogger) (string, error) {
	action := http.MethodPost

	payload, err := makeClusterPayload(clusterName, labels)
	if err != nil {
		return "", err
	}

	reqURL := rc.BaseURL + clusterPath
	headers := map[string]string{"Content-Type": "application/json"}
	headers["Authorization"] = "Bearer " + rc.APIAccessToken

	response, responseBody, err := rancherutil.SendRequest(action, reqURL, headers, payload, rc, log)

	if response != nil && response.StatusCode == http.StatusUnprocessableEntity {
		// if we've already imported this cluster, we get an HTTP 422, so attempt to fetch the existing cluster
		// and get the cluster ID from the response
		log.Debugf("Cluster %s already registered with Rancher, attempting to fetch it", clusterName)
		clusterID, err := GetClusterIDFromRancher(rc, clusterName, log)
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

// makeClusterPayload returns the payload for Rancher cluster creation, given a cluster name
// and labels to apply to it
func makeClusterPayload(clusterName string, labels map[string]string) (string, error) {
	labelsJSONString, err := makeLabelsJSONString(labels)
	if err != nil {
		return "", err
	}
	payload := `{"type": "cluster",
		"name":"` + clusterName + `",
		"dockerRootDir": "/var/lib/docker",
		"enableClusterAlerting": "false",
		"enableClusterMonitoring": "false",
		"enableNetworkPolicy": "false"`

	if len(labelsJSONString) > 0 {
		payload = fmt.Sprintf(`%s, "labels": %s }`, payload, labelsJSONString)
	} else {
		payload = fmt.Sprintf("%s}", payload)
	}
	return payload, nil
}

func makeLabelsJSONString(labels map[string]string) (string, error) {
	if len(labels) == 0 {
		return "", nil
	}
	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return "", err
	}
	return string(labelsJSON), nil
}

// DeleteClusterFromRancher uses the Rancher API to delete a cluster in Rancher.
func DeleteClusterFromRancher(rc *rancherutil.RancherConfig, clusterID string, log vzlog.VerrazzanoLogger) (bool, error) {
	action := http.MethodDelete
	reqURL := rc.BaseURL + clustersPath + "/" + clusterID
	headers := map[string]string{"Authorization": "Bearer " + rc.APIAccessToken}

	response, _, err := rancherutil.SendRequest(action, reqURL, headers, "", rc, log)

	if response != nil && response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotFound {
		return false, fmt.Errorf("tried to delete cluster from Rancher but failed, response code: %d", response.StatusCode)
	}

	if err != nil {
		return false, err
	}

	log.Oncef("Successfully deleted cluster %s from Rancher", clusterID)
	return true, nil
}

// GetClusterIDFromRancher attempts to fetch the cluster from Rancher by name and pull out the cluster ID
func GetClusterIDFromRancher(rc *rancherutil.RancherConfig, clusterName string, log vzlog.VerrazzanoLogger) (string, error) {
	action := http.MethodGet

	reqURL := rc.BaseURL + clustersByNamePath + clusterName
	headers := map[string]string{"Authorization": "Bearer " + rc.APIAccessToken}

	response, responseBody, err := rancherutil.SendRequest(action, reqURL, headers, "", rc, log)

	if response != nil && response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tried to get cluster from Rancher but failed, response code: %d", response.StatusCode)
	}

	if err != nil {
		return "", err
	}

	return httputil.ExtractFieldFromResponseBodyOrReturnError(responseBody, "data.0.id", "unable to find clusterId in Rancher response")
}

// GetAllClustersInRancher returns cluster information for every cluster registered with Rancher
func GetAllClustersInRancher(rc *rancherutil.RancherConfig, log vzlog.VerrazzanoLogger) ([]RancherCluster, []byte, error) {
	reqURL := rc.BaseURL + clustersPath
	headers := map[string]string{"Authorization": "Bearer " + rc.APIAccessToken}

	hash := md5.New() //nolint:gosec //#gosec G401
	clusters := []RancherCluster{}
	for {
		response, responseBody, err := rancherutil.SendRequest(http.MethodGet, reqURL, headers, "", rc, log)
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
			var i map[string]interface{}
			var ok bool
			if i, ok = item.(map[string]interface{}); !ok {
				log.Infof("Expected item to be of type 'map[string]interface{}': %s", responseBody)
				continue
			}
			var name, id interface{}
			if name, ok = i["name"]; !ok {
				log.Infof("Expected to find 'name' field in Rancher cluster data: %s", responseBody)
				continue
			}
			if id, ok = i["id"]; !ok {
				log.Infof("Expected to find 'id' field in Rancher cluster data: %s", responseBody)
				continue
			}
			cluster := RancherCluster{Name: name.(string), ID: id.(string)}
			clusters = append(clusters, cluster)
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
	return clusters, hash.Sum(nil), nil
}

// isManagedClusterActiveInRancher returns true if the managed cluster is active
func isManagedClusterActiveInRancher(rc *rancherutil.RancherConfig, clusterID string, log vzlog.VerrazzanoLogger) (bool, error) {
	reqURL := rc.BaseURL + clustersPath + "/" + clusterID
	headers := map[string]string{"Authorization": "Bearer " + rc.APIAccessToken}

	response, responseBody, err := rancherutil.SendRequest(http.MethodGet, reqURL, headers, "", rc, log)

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
func getCACertFromManagedCluster(rc *rancherutil.RancherConfig, clusterID string, log vzlog.VerrazzanoLogger) (string, error) {
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
func getCACertFromManagedClusterSecret(rc *rancherutil.RancherConfig, clusterID, namespace, secretName, secretKey string, log vzlog.VerrazzanoLogger) (string, error) {
	const k8sAPISecretPattern = "%s/api/v1/namespaces/%s/secrets/%s" //nolint:gosec //#gosec G101

	// use the Rancher API proxy on the managed cluster to fetch the secret
	baseReqURL := rc.BaseURL + k8sClustersPath + clusterID
	headers := map[string]string{"Authorization": "Bearer " + rc.APIAccessToken}

	reqURL := fmt.Sprintf(k8sAPISecretPattern, baseReqURL, namespace, secretName)
	response, responseBody, err := rancherutil.SendRequest(http.MethodGet, reqURL, headers, "", rc, log)

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
func getRegistrationYAMLFromRancher(rc *rancherutil.RancherConfig, rancherClusterID string, log vzlog.VerrazzanoLogger) (string, error) {

	reqURL := rc.BaseURL + clusterRegTokenPath
	headers := map[string]string{"Content-Type": "application/json"}
	headers["Authorization"] = "Bearer " + rc.APIAccessToken

	var token string
	token, err := getRegistrationTokenFromRancher(rc, rancherClusterID, log)
	if err != nil {
		return "", err
	}
	if token == "" {
		action := http.MethodPost
		payload := `{"type": "clusterRegistrationToken", "clusterId": "` + rancherClusterID + `"}`

		response, manifestContent, err := rancherutil.SendRequest(action, reqURL, headers, payload, rc, log)
		if err != nil {
			return "", err
		}

		err = httputil.ValidateResponseCode(response, http.StatusCreated)
		if err != nil {
			return "", err
		}

		// get the manifest token from the response, construct a URL, and fetch its contents
		token, err = httputil.ExtractFieldFromResponseBodyOrReturnError(manifestContent, "token", "unable to find manifest token in Rancher response")
		if err != nil {
			return "", err
		}
	}
	// Rancher 2.5.x added the cluster ID to the manifest URL.
	manifestURL := rc.BaseURL + manifestPath + token + "_" + rancherClusterID + ".yaml"
	action := http.MethodGet
	response, manifestContent, err := rancherutil.SendRequest(action, manifestURL, headers, "", rc, log)

	if err != nil {
		return "", err
	}

	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		return "", err
	}

	return manifestContent, nil
}

type ClusterRegistrationTokens struct {
	Created   string `json:"created"`
	ClusterID string `json:"clusterId"`
	ExpiresAt string `json:"expiresAt"`
	State     string `json:"state"`
	Token     string `json:"token"`
}

func getRegistrationTokenFromRancher(rc *rancherutil.RancherConfig, rancherClusterID string, log vzlog.VerrazzanoLogger) (string, error) {

	log.Infof("Calling getRegistrationTokenFromRancher")
	action := http.MethodGet
	reqURL := rc.BaseURL + clusterRegTokenPath + "?state=active&&clusterId=" + rancherClusterID
	headers := map[string]string{"Content-Type": "application/json"}
	headers["Authorization"] = "Bearer " + rc.APIAccessToken

	response, manifestContent, err := rancherutil.SendRequest(action, reqURL, headers, "{}", rc, log)
	log.Infof("Called getRegistrationTokenFromRancher")
	log.Info(response)
	if err != nil {
		return "", err
	}
	err = httputil.ValidateResponseCode(response, http.StatusOK)
	if err != nil {
		return "", err
	}

	data, err := httputil.ExtractFieldFromResponseBodyOrReturnError(manifestContent, "data", "unable to find data token in Rancher response")
	log.Infof("Extracted data")
	if err != nil {
		return "", err
	}

	var items []ClusterRegistrationTokens
	json.Unmarshal([]byte(data), &items)
	for _, item := range items {
		if item.ClusterID == rancherClusterID && item.State == "active" {
			log.Oncef("ClusterRegistrationToken exists for the cluster %s", rancherClusterID)
			return item.Token, nil
		}
	}

	log.Infof("No active clusterRegistrationToken found")
	return "", err
}

// createOrUpdateSecretRancherProxy simulates the controllerutil create or update function through the Rancher Proxy API for secrets
func createOrUpdateSecretRancherProxy(secret *corev1.Secret, rc *rancherutil.RancherConfig, clusterID string, f controllerutil.MutateFn, log vzlog.VerrazzanoLogger) (controllerutil.OperationResult, error) {
	log.Debugf("Creating or Updating Secret %s/%s", secret.GetNamespace(), secret.GetName())
	if err := rancherSecretGet(secret, rc, clusterID, log); err != nil {
		if !apierrors.IsNotFound(err) {
			return controllerutil.OperationResultNone, err
		}
		if err := rancherSecretMutate(f, secret, log); err != nil {
			return controllerutil.OperationResultNone, err
		}
		if err := rancherSecretCreate(secret, rc, clusterID, log); err != nil {
			return controllerutil.OperationResultNone, err
		}
		return controllerutil.OperationResultCreated, nil
	}

	existingSec := secret.DeepCopyObject()
	if err := rancherSecretMutate(f, secret, log); err != nil {
		return controllerutil.OperationResultNone, err
	}
	if equality.Semantic.DeepEqual(existingSec, secret) {
		return controllerutil.OperationResultNone, nil
	}
	if err := rancherSecretUpdate(secret, rc, clusterID, log); err != nil {
		return controllerutil.OperationResultNone, err
	}
	return controllerutil.OperationResultUpdated, nil
}

// rancherSecretMutate mutates the rancher secret from the given Mutate function
func rancherSecretMutate(f controllerutil.MutateFn, secret *corev1.Secret, log vzlog.VerrazzanoLogger) error {
	key := client.ObjectKeyFromObject(secret)
	if err := f(); err != nil {
		return err
	}
	if newKey := client.ObjectKeyFromObject(secret); key != newKey {
		return log.ErrorfNewErr("MutateFn cannot mutate secret name and/or secret namespace")
	}
	return nil
}

// rancherSecretGet simulates a client get request through the Rancher proxy for secrets
func rancherSecretGet(secret *corev1.Secret, rc *rancherutil.RancherConfig, clusterID string, log vzlog.VerrazzanoLogger) error {
	if secret == nil {
		return log.ErrorNewErr("Failed to get secret, nil value passed to get request")
	}
	reqURL := constructSecretURL(secret, rc.Host, clusterID, false)
	headers := map[string]string{"Authorization": "Bearer " + rc.APIAccessToken}
	resp, body, err := rancherutil.SendRequest(http.MethodGet, reqURL, headers, "", rc, log)
	if err != nil && (resp == nil || resp.StatusCode != 404) {
		return err
	}
	if resp == nil {
		return log.ErrorfNewErr("Failed to find response from GET request %s", secret.GetNamespace(), secret.GetName(), reqURL)
	}
	if resp.StatusCode == http.StatusNotFound {
		return apierrors.NewNotFound(schema.ParseGroupResource("Secret"), secret.GetName())
	}
	if resp.StatusCode != http.StatusOK {
		return log.ErrorfNewErr("Failed to get secret %s/%s from GET request %s with code %d", secret.GetNamespace(), secret.GetName(), reqURL, resp.StatusCode)
	}

	// Unmarshall the response body into the secret object, simulating a typical Get request
	err = yaml.Unmarshal([]byte(body), secret)
	if err != nil {
		return log.ErrorfNewErr("Failed to unmarshall response body into secret %s/%s from GET request %s: %v", secret.GetNamespace(), secret.GetName(), reqURL, err)
	}
	return nil
}

// rancherSecretCreate simulates a client create request through the Rancher proxy for secrets
func rancherSecretCreate(secret *corev1.Secret, rc *rancherutil.RancherConfig, clusterID string, log vzlog.VerrazzanoLogger) error {
	if secret == nil {
		return log.ErrorNewErr("Failed to create secret, nil value passed to create request")
	}
	reqURL := constructSecretURL(secret, rc.Host, clusterID, true)
	payload, err := json.Marshal(secret)
	if err != nil {
		return log.ErrorfNewErr("Failed to marshall secret %s/%s: %v", secret.GetNamespace(), secret.GetName(), err)
	}
	headers := map[string]string{
		"Authorization": "Bearer " + rc.APIAccessToken,
		"Content-Type":  "application/json",
	}
	resp, _, err := rancherutil.SendRequest(http.MethodPost, reqURL, headers, string(payload), rc, log)
	if err != nil {
		return err
	}

	if resp == nil {
		return log.ErrorfNewErr("Failed to find response from POST request %s", secret.GetNamespace(), secret.GetName(), reqURL)
	}
	if resp.StatusCode != http.StatusCreated {
		return log.ErrorfNewErr("Failed to create secret %s/%s from POST request %s with code %d", secret.GetNamespace(), secret.GetName(), reqURL, resp.StatusCode)
	}
	return nil
}

// rancherSecretUpdate simulates a client update request through the Rancher proxy for secrets
func rancherSecretUpdate(secret *corev1.Secret, rc *rancherutil.RancherConfig, clusterID string, log vzlog.VerrazzanoLogger) error {
	if secret == nil {
		return log.ErrorNewErr("Failed to update secret, nil value passed to update request")
	}
	reqURL := constructSecretURL(secret, rc.Host, clusterID, false)
	payload, err := json.Marshal(secret)
	if err != nil {
		return log.ErrorfNewErr("Failed to marshall secret %s/%s: %v", secret.GetNamespace(), secret.GetName(), err)
	}
	headers := map[string]string{
		"Authorization": "Bearer " + rc.APIAccessToken,
		"Content-Type":  "application/json",
	}
	resp, _, err := rancherutil.SendRequest(http.MethodPut, reqURL, headers, string(payload), rc, log)
	if err != nil {
		return err
	}

	if resp == nil {
		return log.ErrorfNewErr("Failed to find response from PUT request %s", secret.GetNamespace(), secret.GetName(), reqURL)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return log.ErrorfNewErr("Failed to create secret %s/%s from PUT request %s with code %d", secret.GetNamespace(), secret.GetName(), reqURL, resp.StatusCode)
	}
	return nil
}

// constructSecretURL returns a formatted url string from path requirements and objects
func constructSecretURL(secret *corev1.Secret, host, clusterID string, create bool) string {
	if create {
		return "https://" + host + k8sClustersPath + clusterID + fmt.Sprintf(secretCreateTemplate, secret.GetNamespace())
	}
	return "https://" + host + k8sClustersPath + clusterID + fmt.Sprintf(secretPathTemplate, secret.GetNamespace(), secret.GetName())
}
