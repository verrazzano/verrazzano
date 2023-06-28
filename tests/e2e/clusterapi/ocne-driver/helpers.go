// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/Jeffail/gabs/v2"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"go.uber.org/zap"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// Acts as a cache, mapping the cluster names to cluster IDs
var clusterIDMapping = map[string]string{}

// Creates the cloud credential through the Rancher REST API
func createCloudCredential(credentialName string, log *zap.SugaredLogger) (string, error) {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cloudcredentials", log)
	privateKeyContents, err := getFileContents(privateKeyPath, log)
	if err != nil {
		log.Infof("error reading private key file: %v", err)
		return "", err
	}

	var cloudCreds RancherCloudCred
	cloudCreds.Name = credentialName
	cloudCreds.InternalName = credentialName
	cloudCreds.Type = "provisioning.cattle.io/cloud-credential"
	cloudCreds.InternalType = "provisioning.cattle.io/cloud-credential"

	var cloudCredConfig RancherOcicredentialConfig
	cloudCredConfig.Fingerprint = fingerprint
	cloudCredConfig.PrivateKeyContents = privateKeyContents
	cloudCredConfig.TenancyID = tenancyID
	cloudCredConfig.UserID = userID
	cloudCredConfig.Region = region
	cloudCreds.RancherOcicredentialConfig = cloudCredConfig

	cloudCredsBdata, err := json.Marshal(cloudCreds)
	if err != nil {
		log.Errorf("json marshalling error: %v", zap.Error(err))
		return "", err
	}

	jsonBody, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusCreated, cloudCredsBdata, log)
	if err != nil {
		log.Errorf("error while retrieving http data: %v", zap.Error(err))
		return "", err
	}
	credID := fmt.Sprint(jsonBody.Path("id").Data())
	return credID, nil
}

// Sends a test request to check that the cloud credential is configured properly
func validateCloudCredential(credID string, log *zap.SugaredLogger) error {
	urlPath := fmt.Sprintf("meta/oci/nodeImages?cloudCredentialId=%s&compartment=%s&region=%s", credID, compartmentID, region)
	requestURL, adminToken := setupRequest(rancherURL, urlPath, log)
	log.Infof("validateCloudCredential URL = %s", requestURL)
	res, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusOK, nil, log)
	if err != nil {
		log.Errorf("error while retrieving http data: %v", zap.Error(err))
		return err
	}
	log.Infof("validate cloud credential response: %s", fmt.Sprint(res))
	return nil
}

// Deletes the cloud credential through the Rancher REST API
func deleteCredential(credID string, log *zap.SugaredLogger) {
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("%s%s", "v3/cloudCredentials/", credID), log)
	helpers.HTTPHelper(httpClient, "DELETE", requestURL, adminToken, "Bearer", http.StatusNoContent, nil, log)
}

// Returns true if the cloud credential is deleted/does not exist
func isCredentialDeleted(credID string, log *zap.SugaredLogger) (bool, error) {
	jsonBody, err := getCredential(credID, log)
	if err != nil {
		return false, err
	}
	// A deleted credential should have an empty "data" array
	data := jsonBody.Path("data").Children()
	if len(data) > 0 {
		err = fmt.Errorf("credential %s still has a non-empty data array from GET call to the API", credID)
		log.Error(err)
		return false, err
	}
	return true, nil
}

// Makes a GET request for the specified cloud credential
func getCredential(credID string, log *zap.SugaredLogger) (*gabs.Container, error) {
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("%s%s", "v3/cloudcredentials?id=", credID), log)
	return helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, log)
}

// Creates a single node OCNE Cluster through CAPI
func createSingleNodeCluster(clusterName string, log *zap.SugaredLogger) error {
	nodePublicKeyContents, err := getFileContents(nodePublicKeyPath, log)
	if err != nil {
		log.Infof("error reading node public key file: %v", err)
		return err
	}

	// Fill in the values for the create cluster API request body
	var rancherOCNEClusterConfig RancherOCNECluster
	var nodePoolSpec []string
	rancherOCNEClusterConfig.fillValues(clusterName, nodePublicKeyContents, cloudCredentialID, nodePoolSpec)

	return createCluster(clusterName, rancherOCNEClusterConfig, log)
}

// Creates a OCNE Cluster with node pools through CAPI
func createNodePoolCluster(clusterName, nodePoolName string, log *zap.SugaredLogger) error {
	nodePublicKeyContents, err := getFileContents(nodePublicKeyPath, log)
	if err != nil {
		log.Infof("error reading node public key file: %v", err)
		return err
	}

	// Fill in the values for the create cluster API request body
	var rancherOCNEClusterConfig RancherOCNECluster
	nodePoolSpec := []string{
		fmt.Sprintf(
			"{\"name\":\"%s\",\"replicas\":1,\"memory\":32,\"ocpus\":2,\"volumeSize\":100,\"shape\":\"VM.Standard.E4.Flex\"}",
			nodePoolName),
	}
	rancherOCNEClusterConfig.fillValues(clusterName, nodePublicKeyContents, cloudCredentialID, nodePoolSpec)

	return createCluster(clusterName, rancherOCNEClusterConfig, log)
}

// Creates an OCNE cluster through ClusterAPI by making a request to the Rancher API
func createCluster(clusterName string, requestPayload RancherOCNECluster, log *zap.SugaredLogger) error {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cluster?_replace=true", log)
	clusterBData, err := json.Marshal(requestPayload)
	if err != nil {
		log.Errorf("json marshalling error: %v", zap.Error(err))
		return err
	}
	res, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusCreated, clusterBData, log)
	if res != nil {
		log.Infof("create cluster response body: %s", res.String())
	}
	if err != nil {
		log.Errorf("error while retrieving http data: %v", zap.Error(err))
		return err
	}
	return nil
}

// Deletes the OCNE cluster by sending a DELETE request to the Rancher API
func deleteCluster(clusterName string, log *zap.SugaredLogger) error {
	clusterID, err := getClusterIDFromName(clusterName, log)
	if err != nil {
		log.Infof("could not fetch cluster ID from cluster name %s: %s", clusterName, err)
		return err
	}
	log.Infof("clusterID for deletion: %s", clusterID)

	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("%s/%s", "v1/provisioning.cattle.io.clusters/fleet-default", clusterID), log)

	_, err = helpers.HTTPHelper(httpClient, "DELETE", requestURL, adminToken, "Bearer", http.StatusOK, nil, log)
	if err != nil {
		log.Errorf("error while deleting cluster: %v", err)
		return err
	}
	return nil
}

// Returns true if the cluster currently exists and is Active
func isClusterActive(clusterName string, log *zap.SugaredLogger) (bool, error) {
	jsonBody, err := getCluster(clusterName, log)
	if err != nil {
		return false, err
	}

	state := fmt.Sprint(jsonBody.Path("data.0.state").Data())
	log.Infof("State: %s", state)
	return state == "active", nil
}

// Returns true if the OCNE cluster is deleted/does not exist
func isClusterDeleted(clusterName string, log *zap.SugaredLogger) (bool, error) {
	jsonBody, err := getCluster(clusterName, log)
	if err != nil {
		return false, err
	}

	// Status logs
	state := fmt.Sprint(jsonBody.Path("data.0.state").Data())
	log.Infof("deleting cluster %s state: %s", clusterName, state)

	// A deleted cluster should have an empty "data" array
	data := jsonBody.Path("data").Children()
	if len(data) > 0 {
		err = fmt.Errorf("cluster %s still has a non-empty data array from GET call to the API", clusterName)
		log.Error(err)
		return false, err
	}

	// Check that the CAPI cluster object was deleted
	clusterID, err := getClusterIDFromName(clusterName, log)
	if err != nil {
		return false, err
	}
	clusterObjectFound, err := getClusterFromK8s(clusterID, clusterID, log)
	if clusterObjectFound {
		return false, err
	}
	return true, err
}

// This retrieves the clusters.cluster.x-k8s.io object and returns true if it exists.
func getClusterFromK8s(namespace, clusterID string, log *zap.SugaredLogger) (bool, error) {
	clusterFetched, err := getUnstructuredData("cluster.x-k8s.io", "v1beta1", "clusters", clusterID, namespace, log)
	if err != nil {
		log.Errorf("Unable to fetch CAPI cluster '%s' due to '%v'", clusterID, zap.Error(err))
		return false, err
	}

	if clusterFetched == nil {
		log.Infof("No CAPI clusters with id '%s' in namespace '%s' was detected", clusterID, namespace)
		return false, nil
	}
	return true, nil
}

// Checks whether the cluster was created as expected. Returns nil if all is good.
func verifyCluster(clusterName string, numberNodes int, log *zap.SugaredLogger) error {
	// Check if the cluster looks good from the Rancher API
	var err error
	if err = verifyGetRequest(clusterName, numberNodes, log); err != nil {
		return err
	}

	// Get the kubeconfig of the cluster to look inside
	clusterID, err := getClusterIDFromName(clusterName, log)
	if err != nil {
		log.Errorf("could not get cluster ID for cluster %s", clusterName, log)
		return err
	}
	workloadKubeconfigPath, err := getWorkloadKubeconfig(clusterID, log)
	if err != nil {
		log.Errorf("could not get kubeconfig for cluster %s", clusterName)
		return err
	}

	// Check if the cluster has the expected nodes and pods running
	if err = verifyClusterNodes(clusterName, workloadKubeconfigPath, numberNodes, log); err != nil {
		return err
	}
	return verifyClusterPods(clusterName, workloadKubeconfigPath, log)
}

// Verifies that a GET request to the Rancher API for this cluster returns expected values.
// Intended to be called on clusters in Active state.
func verifyGetRequest(clusterName string, numberNodes int, log *zap.SugaredLogger) error {
	jsonBody, err := getCluster(clusterName, log)
	if err != nil {
		return err
	}
	jsonData := jsonBody.Path("data.0")

	// Assert that these attributes are as expected
	resourceType := jsonBody.Path("resourceType").Data()
	name := jsonData.Path("name").Data()
	nodeCount := jsonData.Path("nodeCount").Data()
	state := jsonData.Path("state").Data()
	transitioning := jsonData.Path("transitioning").Data()
	fleetNamespace := jsonData.Path("fleetWorkspaceName").Data()
	driver := jsonData.Path("driver").Data()

	attributes := []struct {
		actual   interface{}
		expected interface{}
		name     string
	}{
		{resourceType, "cluster", "resource type"},
		{name, clusterName, "cluster name"},
		{nodeCount, float64(numberNodes), "node count"},
		{state, "active", "state"},
		{transitioning, "no", "transitioning flag"},
		{fleetNamespace, "fleet-default", "fleet workspace"},
		{driver, "ociocne", "driver"},
	}
	for _, a := range attributes {
		if a.actual != a.expected {
			return fmt.Errorf("cluster %s has a %s value of %v but should be %v", clusterName, a.name, a.actual, a.expected)
		}
	}

	// Assert that these attributes are not nil
	caCert := jsonData.Path("caCert").Data()
	requestedResources := jsonData.Path("requested").Data()

	nonNilAttributes := []struct {
		value interface{}
		name  string
	}{
		{caCert, "CA certificate"},
		{requestedResources, "requested resources"},
	}
	for _, n := range nonNilAttributes {
		if n.value == nil {
			return fmt.Errorf("cluster %s should have a non-nil value for %s", clusterName, n.name)
		}
	}

	return nil
}

// Verifies that the workload cluster has the expected number of nodes.
func verifyClusterNodes(clusterName, kubeconfigPath string, expectedNumberNodes int, log *zap.SugaredLogger) error {
	numNodes, err := pkg.GetNodeCountInCluster(kubeconfigPath)
	if err != nil {
		log.Errorf("could not verify number of nodes in cluster %s: %s", clusterName, err)
		return err
	}
	if numNodes != expectedNumberNodes {
		err = fmt.Errorf("expected %v nodes in cluster %s but got %v", expectedNumberNodes, clusterName, numNodes)
		log.Error(err)
		return err
	}
	return nil
}

// Verifies that all expected pods in the workload cluster are active,
// given the cluster's kubeconfig path and the cluster name
func verifyClusterPods(clusterName, kubeconfigPath string, log *zap.SugaredLogger) error {
	// keys are the namespaces, and values are the pod name prefixes
	expectedPods := map[string][]string{
		"verrazzano-module-operator": {"verrazzano-module-operator"},
		"calico-apiserver":           {"calico-apiserver"},
		"calico-system":              {"calico-kube-controllers", "calico-node", "calico-typha", "csi-node-driver"},
		"cattle-fleet-system":        {"fleet-agent"},
		"cattle-system":              {"cattle-cluster-agent"},
		"default":                    {"tigera-operator"},
		"kube-system": {"coredns", "csi-oci-controller", "csi-oci-node", "etcd", "kube-apiserver",
			"kube-controller-manager", "kube-proxy", "kube-scheduler", "oci-cloud-controller-manager"},
	}

	// check the expected pods inside the workload cluster
	for namespace, namePrefixes := range expectedPods {
		podsRunning, err := pkg.PodsRunningInCluster(namespace, namePrefixes, kubeconfigPath)
		if err != nil {
			log.Errorf("error while verifying running pods: %s", err)
			return err
		}
		if !podsRunning {
			err = fmt.Errorf("there are missing pods in the %s namespace", namespace)
			log.Error(err)
			return err
		}
	}

	log.Infof("all expected pods in cluster %s are running", clusterName)
	return nil
}

// Gets a specified cluster by using the Rancher REST API
func getCluster(clusterName string, log *zap.SugaredLogger) (*gabs.Container, error) {
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("v3/cluster?name=%s", clusterName), log)
	return helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, log)
}

// Returns the cluster ID corresponding the given name
func getClusterIDFromName(clusterName string, log *zap.SugaredLogger) (string, error) {
	// Check if we already have this value cached
	if id, ok := clusterIDMapping[clusterName]; ok {
		return id, nil
	}

	// Get the cluster ID from the Rancher REST API
	jsonBody, err := getCluster(clusterName, log)
	if err != nil {
		log.Errorf("failed getting cluster ID from GET call to Rancher API: %s", err)
		return "", err
	}
	id := fmt.Sprint(jsonBody.Path("data.0.id").Data())

	// Cache this value and return
	clusterIDMapping[clusterName] = id
	return id, nil
}

// Generates the kubeconfig of the workload cluster with an ID of `clusterID`
// Writes the kubeconfig to a file inside /tmp. Returns the path of the kubeconfig file.
func getWorkloadKubeconfig(clusterID string, log *zap.SugaredLogger) (string, error) {
	outputPath := fmt.Sprintf("/tmp/%s-kubeconfig", clusterID)

	// First check if the kubeconfig is already present on our filesystem
	if _, err := os.Stat(outputPath); err == nil {
		return outputPath, nil
	}

	// Otherwise, download the kubeconfig through an API call
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("v3/clusters/%s?action=generateKubeconfig", clusterID), log)
	jsonBody, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusOK, nil, log)
	if err != nil {
		log.Errorf("error while retrieving http data: %v", zap.Error(err))
		return "", err
	}

	workloadKubeconfig := fmt.Sprint(jsonBody.Path("config").Data())
	err = os.WriteFile(outputPath, []byte(workloadKubeconfig), 0600)
	if err != nil {
		log.Errorf("error writing workload cluster kubeconfig to a file: %v", zap.Error(err))
		return "", err
	}
	return outputPath, nil
}

// Given a file path, returns the file's contents as a string
func getFileContents(file string, log *zap.SugaredLogger) (string, error) {
	data, err := os.ReadFile(file)
	fmt.Println("========= file: " + file)
	if err != nil {
		log.Errorf("failed reading file contents: %v", err)
		return "", err
	}
	return string(data), nil
}

// Given the base Rancher URL and the additional URL path,
// returns the full URL and a refreshed Bearer token
func setupRequest(rancherBaseURL, urlPath string, log *zap.SugaredLogger) (string, string) {
	adminToken := helpers.GetRancherLoginToken(log)
	log.Infof("adminToken: %s", adminToken)
	requestURL := fmt.Sprintf("%s/%s", rancherBaseURL, urlPath)
	log.Infof("requestURL: %s", requestURL)
	return requestURL, adminToken
}

// getUnstructuredData common utility to fetch unstructured data
func getUnstructuredData(group, version, resource, resourceName, nameSpaceName string, log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
	var dataFetched *unstructured.Unstructured
	var err error
	config, err := k8sutil.GetKubeConfig()
	if err != nil {
		log.Errorf("Unable to fetch kubeconfig %v", zap.Error(err))
		return nil, err
	}
	dclient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Errorf("Unable to create dynamic client %v", zap.Error(err))
		return nil, err
	}

	gvr := schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}

	if nameSpaceName != "" {
		log.Infof("fetching '%s' '%s' in namespace '%s'", resource, resourceName, nameSpaceName)
		dataFetched, err = dclient.Resource(gvr).Namespace(nameSpaceName).Get(context.TODO(), resourceName, metav1.GetOptions{})
	} else {
		log.Infof("fetching '%s' '%s'", resource, resourceName)
		dataFetched, err = dclient.Resource(gvr).Get(context.TODO(), resourceName, metav1.GetOptions{})
	}
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Errorf("resource %s %s not found", resource, resourceName)
			return nil, nil
		}
		log.Errorf("Unable to fetch %s %s due to '%v'", resource, resourceName, zap.Error(err))
		return nil, err
	}
	return dataFetched, nil
}
