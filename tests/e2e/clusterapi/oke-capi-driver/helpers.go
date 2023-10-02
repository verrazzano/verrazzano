// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package okecapidriver

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

type clusterState string
type transitioningFlag string

const (
	provisioningClusterState clusterState      = "provisioning"
	activeClusterState       clusterState      = "active"
	transitioningFlagNo      transitioningFlag = "no"
	transitioningFlagError   transitioningFlag = "error"
)

// Creates the cloud credential through the Rancher REST API
func createCloudCredential(credentialName string, log *zap.SugaredLogger) (string, error) {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cloudcredentials", log)
	privateKeyContents, err := getFileContents(privateKeyPath, log)
	if err != nil {
		log.Errorf("error reading private key file: %v", err)
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

type mutateRancherOKECAPIClusterFunc func(config *RancherOKECluster)

// This returns a mutateRancherOKECAPIClusterFunc, which edits a cluster config to have a node pool with the specified name and number of replicas.
// Node pool nodes use the specified volume size, number of ocpus, and memory.
func getMutateFnNodePoolsAndResourceUsage(nodePoolName, version string, poolReplicas, volumeSize, ocpus, memory int) mutateRancherOKECAPIClusterFunc {
	return func(config *RancherOKECluster) {
		config.OKECAPIEngineConfig.NodePools = []string{
			getNodePoolSpec(nodePoolName, version, nodeShape, poolReplicas, memory, ocpus, volumeSize),
		}
	}
}

// Creates a CAPI based OKE Cluster, and returns an error if not successful. Creates a single node cluster by default.
// `config` is expected to point to an empty RancherOKECluster struct, which is populated with values by this function.
// `mutateFn`, if not nil, can be used to make additional changes to the cluster config before the cluster creation request is made.
func createClusterAndFillConfig(clusterName string, config *RancherOKECluster, log *zap.SugaredLogger, mutateFn mutateRancherOKECAPIClusterFunc) error {
	nodePublicKeyContents, err := getFileContents(nodePublicKeyPath, log)
	if err != nil {
		log.Errorf("error reading node public key file: %v", err)
		return err
	}

	// Fill in the values for the create cluster API request body
	config.fillCommonValues()
	config.OKECAPIEngineConfig.CloudCredentialID = cloudCredentialID
	config.OKECAPIEngineConfig.DisplayName = clusterName
	config.OKECAPIEngineConfig.NodePublicKeyContents = nodePublicKeyContents
	config.OKECAPIEngineConfig.NodePools = []string{}
	config.CloudCredentialID = cloudCredentialID
	config.Name = clusterName

	// Make additional changes to the cluster config
	if mutateFn != nil {
		mutateFn(config)
	}

	return createCluster(clusterName, *config, log)
}

// Creates an OKE cluster through ClusterAPI by making a request to the Rancher API
func createCluster(clusterName string, requestPayload RancherOKECluster, log *zap.SugaredLogger) error {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cluster?replace=true", log)
	clusterBData, err := json.Marshal(requestPayload)
	if err != nil {
		log.Errorf("json marshalling error: %v", zap.Error(err))
		return err
	}
	log.Infof("create cluster body: %s", string(clusterBData))
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

// Deletes the OKE cluster by sending a DELETE request to the Rancher API
func deleteCluster(clusterName string, log *zap.SugaredLogger) error {
	clusterID, err := getClusterIDFromName(clusterName, log)
	if err != nil {
		log.Errorf("could not fetch cluster ID from cluster name %s: %s", clusterName, err)
		return err
	}
	if clusterID == "" || clusterID == "<nil>" {
		log.Errorf("Cluster %s does not exist", clusterName)
		return nil
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

// This function takes in the cluster config of an existing cluster, and changes the fields required to make the update.
// Then, this triggers an update for the OKE cluster.
func updateConfigAndCluster(config *RancherOKECluster, mutateFn mutateRancherOKECAPIClusterFunc, log *zap.SugaredLogger) error {
	if mutateFn == nil {
		err := fmt.Errorf("cannot provide a nil mutate function to update the cluster")
		log.Error(err)
		return err
	}

	clusterName := config.Name
	mutateFn(config)
	return updateCluster(clusterName, *config, log)
}

// Requests an update to the node pool configuration of the OKE cluster
// via a PUT request to the Rancher API
func updateCluster(clusterName string, requestPayload RancherOKECluster, log *zap.SugaredLogger) error {
	clusterID, err := getClusterIDFromName(clusterName, log)
	if err != nil {
		log.Errorf("Could not fetch cluster ID from cluster name %s: %s", clusterName, err)
		return err
	}
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("v3/clusters/%s?replace=true", clusterID), log)
	clusterBData, err := json.Marshal(requestPayload)
	if err != nil {
		log.Errorf("json marshalling error: %v", zap.Error(err))
		return err
	}
	log.Infof("update cluster body: %s", string(clusterBData))
	_, err = helpers.HTTPHelper(httpClient, "PUT", requestURL, adminToken, "Bearer", http.StatusOK, clusterBData, log)
	if err != nil {
		log.Errorf("error while retrieving http data: %v", zap.Error(err))
		return err
	}
	return nil
}

// Returns true if the cluster currently exists and is Active
func isClusterActive(clusterName string, log *zap.SugaredLogger) (bool, error) {
	clusterID, err := getClusterIDFromName(clusterName, log)
	if err != nil {
		log.Errorf("Could not fetch cluster ID from cluster name %s: %s", clusterName, err)
		return false, err
	}

	// Debug logging
	var cmd helpers.BashCommand
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl", "get", "clusters.cluster.x-k8s.io", "-A")
	cmd.CommandArgs = cmdArgs
	response := helpers.Runner(&cmd, log)
	log.Infof("All CAPI clusters =  %s", (&response.StandardOut).String())

	cmdArgs = []string{}
	cmdArgs = append(cmdArgs, "kubectl", "get", "clusters.management.cattle.io")
	cmd.CommandArgs = cmdArgs
	response = helpers.Runner(&cmd, log)
	log.Infof("All management clusters =  %s", (&response.StandardOut).String())

	cmdArgs = []string{}
	cmdArgs = append(cmdArgs, "kubectl", "get", "clusters.provisioning.cattle.io", "-A")
	cmd.CommandArgs = cmdArgs
	response = helpers.Runner(&cmd, log)
	log.Infof("All provisioning clusters =  %s", (&response.StandardOut).String())

	cmdArgs = []string{}
	cmdArgs = append(cmdArgs, "kubectl", "get", "mp", "-A")
	cmd.CommandArgs = cmdArgs
	response = helpers.Runner(&cmd, log)
	log.Infof("All CAPI machine pools =  %s", (&response.StandardOut).String())

	kubeconfigPath, err := getWorkloadKubeconfig(clusterID, log)
	if err != nil {
		log.Error("could not download kubeconfig from rancher")
	}

	cmdArgs = []string{}
	cmdArgs = append(cmdArgs, "kubectl", "--kubeconfig", kubeconfigPath, "get", "nodes", "-o", "wide")
	cmd.CommandArgs = cmdArgs
	response = helpers.Runner(&cmd, log)
	log.Infof("All nodes in workload cluster =  %s", (&response.StandardOut).String())

	cmdArgs = []string{}
	cmdArgs = append(cmdArgs, "kubectl", "--kubeconfig", kubeconfigPath, "get", "pod", "-A", "-o", "wide")
	cmd.CommandArgs = cmdArgs
	response = helpers.Runner(&cmd, log)
	log.Infof("All pods in workload cluster =  %s", (&response.StandardOut).String())

	// Check if the cluster is active
	return checkProvisioningClusterReady("fleet-default", clusterID, log)
}

// Returns true if the OKE cluster is deleted/does not exist
func isClusterDeleted(clusterName string, log *zap.SugaredLogger) (bool, error) {
	// Check that the CAPI cluster object was deleted
	clusterID, err := getClusterIDFromName(clusterName, log)
	if err != nil {
		return false, err
	}
	clusterObjectFound, err := checkClusterExistsFromK8s(clusterID, clusterID, log)
	return !clusterObjectFound, err
}

// Returns true if the requested provisioning cluster object has a ready status set to true
func checkProvisioningClusterReady(namespace, clusterID string, log *zap.SugaredLogger) (bool, error) {
	provClusterFetched, err := fetchProvisioningClusterFromK8s(namespace, clusterID, log)
	if err != nil {
		return false, err
	}
	if provClusterFetched == nil {
		err = fmt.Errorf("no provisioning cluster %s found", clusterID)
		log.Error(err)
		return false, err
	}

	// convert the fetched unstructured object to a provisioning cluster struct
	var provCluster ProvisioningCluster
	bdata, err := json.Marshal(provClusterFetched)
	if err != nil {
		log.Errorf("json marshalling error %v", zap.Error(err))
		return false, err
	}
	err = json.Unmarshal(bdata, &provCluster)
	if err != nil {
		log.Errorf("json unmarshall error %v", zap.Error(err))
		return false, err
	}

	if provCluster.Status.Ready {
		log.Infof("provisioning cluster %s is ready", clusterID)
	}
	return provCluster.Status.Ready, err
}

// Fetches the provisioning cluster object
func fetchProvisioningClusterFromK8s(namespace, clusterID string, log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
	clusterFetched, err := getUnstructuredData("provisioning.cattle.io", "v1", "clusters", clusterID, namespace, log)
	if err != nil {
		log.Errorf("unable to fetch provisioning cluster '%s' due to '%v'", clusterID, zap.Error(err))
		return nil, err
	}
	return clusterFetched, nil
}

// This retrieves the clusters.cluster.x-k8s.io object and returns true if it exists.
func checkClusterExistsFromK8s(namespace, clusterID string, log *zap.SugaredLogger) (bool, error) {
	clusterFetched, err := fetchClusterFromK8s(namespace, clusterID, log)
	if err != nil {
		return false, err
	}
	if clusterFetched == nil {
		log.Infof("No CAPI clusters with id '%s' in namespace '%s' was detected", clusterID, namespace)
		return false, nil
	}
	return true, nil
}

// This fetches the clusters.cluster.x-k8s.io object
func fetchClusterFromK8s(namespace, clusterID string, log *zap.SugaredLogger) (*unstructured.Unstructured, error) {
	clusterFetched, err := getUnstructuredData("cluster.x-k8s.io", "v1beta1", "clusters", clusterID, namespace, log)
	if err != nil {
		log.Errorf("unable to fetch CAPI cluster '%s' due to '%v'", clusterID, zap.Error(err))
		return nil, err
	}
	return clusterFetched, nil
}

// Checks whether the cluster was created as expected. Returns nil if all is good.
func verifyCluster(clusterName string, expectedNodes int, expectedClusterState clusterState, expectedTransitioning transitioningFlag, log *zap.SugaredLogger) error {
	// Check if the cluster looks good from the Rancher API
	var err error
	if err = verifyGetRequest(clusterName, expectedNodes, expectedClusterState, expectedTransitioning, log); err != nil {
		log.Errorf("error validating GET request for cluster %s: %s", clusterName, err)
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

	if expectedNodes > 0 {
		// Check if the cluster has the expected nodes and pods running
		if err = verifyClusterNodes(clusterName, workloadKubeconfigPath, expectedNodes, log); err != nil {
			log.Errorf("error validating number of nodes in %s: %s", clusterName, err)
			return err
		}
		return verifyClusterPods(clusterName, workloadKubeconfigPath, log)
	}
	return nil
}

// Verifies that a GET request to the Rancher API for this cluster returns expected values.
// Intended to be called on clusters with expected number of nodes and cluster state.
func verifyGetRequest(clusterName string, expectedNodes int, expectedClusterState clusterState, expectedTransitioning transitioningFlag, log *zap.SugaredLogger) error {
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
		{nodeCount, float64(expectedNodes), "node count"},
		{state, string(expectedClusterState), "state"},
		{transitioning, string(expectedTransitioning), "transitioning flag"},
		{fleetNamespace, "fleet-default", "fleet workspace"},
		{driver, "okecapi", "driver"},
	}
	for _, a := range attributes {
		if a.actual != a.expected {
			return fmt.Errorf("cluster %s has a %s value of %v but should be %v", clusterName, a.name, a.actual, a.expected)
		}
	}

	// Assert that these attributes are not nil
	caCert := jsonData.Path("caCert").Data()
	requestedResources := jsonData.Path("requested").Data()

	if expectedClusterState == activeClusterState {
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
	}

	log.Infof("cluster %s looks as expected from a GET call to the Rancher API", clusterName)
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
	log.Infof("cluster %s had the expected number of nodes", clusterName)
	return nil
}

// Verifies that all expected pods in the workload cluster are active,
// given the cluster's kubeconfig path and the cluster name
func verifyClusterPods(clusterName, kubeconfigPath string, log *zap.SugaredLogger) error {
	// keys are the namespaces, and values are the pod name prefixes
	expectedPods := map[string][]string{
		"kube-system": {"coredns", "csi-oci-node", "kube-proxy"},
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

// Gets a specified cluster by using the Rancher REST API.
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

// Returns a string representing a node pool
func getNodePoolSpec(name, version, shape string, replicas, memory, ocpus, volumeSize int) string {
	return fmt.Sprintf("{\"name\":\"%s\",\"version\":\"%s\",\"replicas\":%d,\"memory\":%d,\"ocpus\":%d,\"volumeSize\":%d,\"shape\":\"%s\"}",
		name, version, replicas, memory, ocpus, volumeSize, shape)
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
