// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Jeffail/gabs/v2"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/verrazzano/verrazzano/pkg/k8sutil"
	"github.com/verrazzano/verrazzano/tests/e2e/backup/helpers"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg"
	"github.com/verrazzano/verrazzano/tests/e2e/pkg/test/framework"
	"go.uber.org/zap"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// nolint: gosec // auth constants, not credentials
// gosec: G101: Potential hardcoded credentials
const (
	shortWaitTimeout     = 10 * time.Minute
	shortPollingInterval = 10 * time.Second
	waitTimeout          = 45 * time.Minute
	pollingInterval      = 30 * time.Second
)

var (
	t                     = framework.NewTestFramework("capi-ocne-driver")
	httpClient            *retryablehttp.Client
	rancherURL            string
	cloudCredentialID     string
	clusterNameSingleNode string
	clusterNameNodePool   string
	cloudCredentialName   string
)

type RancherOcicredentialConfig struct {
	Fingerprint        string `json:"fingerprint"`
	PrivateKeyContents string `json:"privateKeyContents"`
	Region             string `json:"region"`
	TenancyID          string `json:"tenancyId"`
	UserID             string `json:"userId"`
}

type RancherCloudCred struct {
	Type     string `json:"type"`
	Metadata struct {
		GenerateName string `json:"generateName"`
		Namespace    string `json:"namespace"`
	} `json:"metadata"`
	InternalName string `json:"_name"`
	Annotations  struct {
		ProvisioningCattleIoDriver string `json:"provisioning.cattle.io/driver"`
	} `json:"annotations"`
	RancherOcicredentialConfig `json:"ocicredentialConfig"`
	InternalType               string `json:"_type"`
	Name                       string `json:"name"`
}

type RancherOCIOCNEEngine struct {
	CalicoImagePath       string        `json:"calicoImagePath"`
	CloudCredentialID     string        `json:"cloudCredentialId"`
	ClusterCidr           string        `json:"clusterCidr"`
	CompartmentID         string        `json:"compartmentId"`
	ControlPlaneMemoryGbs int           `json:"controlPlaneMemoryGbs"`
	ControlPlaneOcpus     int           `json:"controlPlaneOcpus"`
	ControlPlaneShape     string        `json:"controlPlaneShape"`
	ControlPlaneSubnet    string        `json:"controlPlaneSubnet"`
	ControlPlaneVolumeGbs int           `json:"controlPlaneVolumeGbs"`
	CorednsImageTag       string        `json:"corednsImageTag"`
	DisplayName           string        `json:"displayName"`
	DriverName            string        `json:"driverName"`
	EtcdImageTag          string        `json:"etcdImageTag"`
	ImageDisplayName      string        `json:"imageDisplayName"`
	ImageID               string        `json:"imageId"`
	InstallCalico         bool          `json:"installCalico"`
	InstallCcm            bool          `json:"installCcm"`
	InstallVerrazzano     bool          `json:"installVerrazzano"`
	KubernetesVersion     string        `json:"kubernetesVersion"`
	LoadBalancerSubnet    string        `json:"loadBalancerSubnet"`
	Name                  string        `json:"name"`
	NodePublicKeyContents string        `json:"nodePublicKeyContents"`
	NumControlPlaneNodes  int           `json:"numControlPlaneNodes"`
	OcneVersion           string        `json:"ocneVersion"`
	PodCidr               string        `json:"podCidr"`
	PrivateRegistry       string        `json:"privateRegistry"`
	ProxyEndpoint         string        `json:"proxyEndpoint"`
	Region                string        `json:"region"`
	SkipOcneInstall       bool          `json:"skipOcneInstall"`
	TigeraImageTag        string        `json:"tigeraImageTag"`
	UseNodePvEncryption   bool          `json:"useNodePvEncryption"`
	VcnID                 string        `json:"vcnId"`
	VerrazzanoResource    string        `json:"verrazzanoResource"`
	VerrazzanoTag         string        `json:"verrazzanoTag"`
	VerrazzanoVersion     string        `json:"verrazzanoVersion"`
	WorkerNodeSubnet      string        `json:"workerNodeSubnet"`
	Type                  string        `json:"type"`
	ClusterName           string        `json:"clusterName"`
	NodeShape             string        `json:"nodeShape"`
	NumWorkerNodes        int           `json:"numWorkerNodes"`
	NodePools             []interface{} `json:"nodePools"`
	ApplyYamls            []interface{} `json:"applyYamls"`
}

type RancherOCNECluster struct {
	DockerRootDir           string               `json:"dockerRootDir"`
	EnableClusterAlerting   bool                 `json:"enableClusterAlerting"`
	EnableClusterMonitoring bool                 `json:"enableClusterMonitoring"`
	EnableNetworkPolicy     bool                 `json:"enableNetworkPolicy"`
	WindowsPreferedCluster  bool                 `json:"windowsPreferedCluster"`
	Type                    string               `json:"type"`
	Name                    string               `json:"name"`
	OciocneEngineConfig     RancherOCIOCNEEngine `json:"ociocneEngineConfig"`
	CloudCredentialID       string               `json:"cloudCredentialId"`
	Labels                  struct {
	} `json:"labels"`
}

// Part of SynchronizedBeforeSuite, run by only one process
func sbsProcess1Func() []byte {
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())
	if !pkg.IsRancherEnabled(kubeconfigPath) || !pkg.IsClusterAPIEnabled(kubeconfigPath) {
		AbortSuite("skipping ocne cluster driver test suite since either of rancher and capi components are not enabled")
	}

	httpClient, err = pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("failed getting http client: %v", err))
	}

	rancherURL, err = helpers.GetRancherURL(t.Logs)
	if err != nil {
		AbortSuite(fmt.Sprintf("failed getting rancherURL: %v", err))
	}

	// Create the cloud credential to be used for all tests
	cloudCredentialName = fmt.Sprintf("strudel-cred-%s", ocneClusterNameSuffix)
	var credentialID string
	Eventually(func() error {
		var err error
		credentialID, err = createCloudCredential(cloudCredentialName)
		return err
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	Eventually(func() error {
		return validateCloudCredential(credentialID)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	// Return byte encoded cloud credential ID to be shared across all processes
	return []byte(credentialID)
}

// Part of SynchronizedBeforeSuite, run by all processes
func sbsAllProcessesFunc(credentialIDBytes []byte) {
	// Define global variables for all processes
	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())

	httpClient, err = pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("failed getting http client: %v", err))
	}

	rancherURL, err = helpers.GetRancherURL(t.Logs)
	if err != nil {
		AbortSuite(fmt.Sprintf("failed getting rancherURL: %v", err))
	}

	cloudCredentialID = string(credentialIDBytes)
	clusterNameSingleNode = fmt.Sprintf("strudel-single-%s", ocneClusterNameSuffix)
	clusterNameNodePool = fmt.Sprintf("strudel-pool-%s", ocneClusterNameSuffix)
}

var _ = t.SynchronizedBeforeSuite(sbsProcess1Func, sbsAllProcessesFunc)

// Part of SynchronizedAfterSuite, run by only one process
func sasProcess1Func() {
	// Delete the clusters concurrently
	clusterNames := [...]string{clusterNameSingleNode, clusterNameNodePool}
	var wg sync.WaitGroup
	wg.Add(len(clusterNames))
	for _, clusterName := range clusterNames {
		go func(name string) {
			defer wg.Done()
			// Delete the OCNE cluster
			Eventually(func() error {
				return deleteCluster(name)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

			// Verify the cluster is deleted
			Eventually(func() (bool, error) { return isClusterDeleted(name) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not deleted", name))
		}(clusterName)
	}
	wg.Wait()

	// Delete the credential
	deleteCredential(cloudCredentialID)

	// Verify the credential is deleted
	Eventually(func() (bool, error) { return isCredentialDeleted(cloudCredentialID) }, waitTimeout, pollingInterval).Should(
		BeTrue(), fmt.Sprintf("cloud credential %s is not deleted", cloudCredentialID))
}

var _ = t.SynchronizedAfterSuite(func() {}, sasProcess1Func)

var _ = t.Describe("OCNE Cluster Driver", Label("f:rancher-capi:ocne-cluster-driver"), func() {
	t.Context("OCNE cluster creation with single node", Ordered, func() {
		t.It("create OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				return createSingleNodeCluster(clusterNameSingleNode)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check OCNE cluster is active", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return isClusterActive(clusterNameSingleNode) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameSingleNode))

			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameSingleNode, 1)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameSingleNode))
		})
	})

	t.Context("OCNE cluster creation with node pools", Ordered, func() {
		t.It("create OCNE cluster", func() {
			nodePoolName := fmt.Sprintf("pool-%s", ocneClusterNameSuffix)
			// Create the cluster
			Eventually(func() error {
				return createNodePoolCluster(clusterNameNodePool, nodePoolName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("check OCNE cluster is active", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return isClusterActive(clusterNameNodePool) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("cluster %s is not active", clusterNameNodePool))

			// Verify that the cluster is configured correctly
			Eventually(func() error {
				return verifyCluster(clusterNameNodePool, 1)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil(), fmt.Sprintf("could not verify cluster %s", clusterNameNodePool))
		})
	})
})

// Creates the cloud credential through the Rancher REST API
func createCloudCredential(credentialName string) (string, error) {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cloudcredentials")
	privateKeyContents, err := getFileContents(privateKeyPath)
	if err != nil {
		t.Logs.Infof("error reading private key file: %v", err)
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
		t.Logs.Errorf("json marshalling error: %v", zap.Error(err))
		return "", err
	}

	jsonBody, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusCreated, cloudCredsBdata, t.Logs)
	if err != nil {
		t.Logs.Errorf("error while retrieving http data: %v", zap.Error(err))
		return "", err
	}
	credID := fmt.Sprint(jsonBody.Path("id").Data())
	return credID, nil
}

// Sends a test request to check that the cloud credential is configured properly
func validateCloudCredential(credID string) error {
	urlPath := fmt.Sprintf("meta/oci/nodeImages?cloudCredentialId=%s&compartment=%s&region=%s", credID, compartmentID, region)
	requestURL, adminToken := setupRequest(rancherURL, urlPath)
	t.Logs.Infof("validateCloudCredential URL = %s", requestURL)
	res, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
	if err != nil {
		t.Logs.Errorf("error while retrieving http data: %v", zap.Error(err))
		return err
	}
	t.Logs.Infof("validate cloud credential response: %s", fmt.Sprint(res))
	return nil
}

// Deletes the cloud credential through the Rancher REST API
func deleteCredential(credID string) {
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("%s%s", "v3/cloudCredentials/", credID))
	helpers.HTTPHelper(httpClient, "DELETE", requestURL, adminToken, "Bearer", http.StatusNoContent, nil, t.Logs)
}

// Returns true if the cloud credential is deleted/does not exist
func isCredentialDeleted(credID string) (bool, error) {
	jsonBody, err := getCredential(credID)
	if err != nil {
		return false, err
	}
	data := fmt.Sprint(jsonBody.Path("data").Data())
	// fmt.Println("Delete credential json data: " + data)
	return data == "[]", nil
}

// Makes a GET request for the specified cloud credential
func getCredential(credID string) (*gabs.Container, error) {
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("%s%s", "v3/cloudcredentials?id=", credID))
	return helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
}

// Creates a single node OCNE Cluster through CAPI
func createSingleNodeCluster(clusterName string) error {
	nodePublicKeyContents, err := getFileContents(nodePublicKeyPath)
	if err != nil {
		t.Logs.Infof("error reading node public key file: %v", err)
		return err
	}

	// Fill in the values for the create cluster API request body
	var rancherOCNEEngineConfig RancherOCIOCNEEngine
	rancherOCNEEngineConfig.CalicoImagePath = "olcne"
	rancherOCNEEngineConfig.CloudCredentialID = cloudCredentialID
	rancherOCNEEngineConfig.ClusterCidr = "10.96.0.0/16"
	rancherOCNEEngineConfig.CompartmentID = compartmentID
	rancherOCNEEngineConfig.ControlPlaneMemoryGbs = 16
	rancherOCNEEngineConfig.ControlPlaneOcpus = 2
	rancherOCNEEngineConfig.ControlPlaneShape = "VM.Standard.E4.Flex"
	rancherOCNEEngineConfig.ControlPlaneSubnet = controlPlaneSubnet
	rancherOCNEEngineConfig.ControlPlaneVolumeGbs = 100
	rancherOCNEEngineConfig.CorednsImageTag = "v1.9.3"
	rancherOCNEEngineConfig.DisplayName = clusterName
	rancherOCNEEngineConfig.DriverName = "ociocneengine"
	rancherOCNEEngineConfig.EtcdImageTag = "3.5.6"
	rancherOCNEEngineConfig.ImageDisplayName = "Oracle-Linux-8.7-2023.05.24-0"
	rancherOCNEEngineConfig.ImageID = ""
	rancherOCNEEngineConfig.InstallCalico = true
	rancherOCNEEngineConfig.InstallCcm = true
	rancherOCNEEngineConfig.InstallVerrazzano = false
	rancherOCNEEngineConfig.KubernetesVersion = "v1.25.7"
	rancherOCNEEngineConfig.LoadBalancerSubnet = loadBalancerSubnet
	rancherOCNEEngineConfig.Name = ""
	rancherOCNEEngineConfig.NodePublicKeyContents = nodePublicKeyContents
	rancherOCNEEngineConfig.NumControlPlaneNodes = 1
	rancherOCNEEngineConfig.OcneVersion = "1.6"
	rancherOCNEEngineConfig.PodCidr = "10.244.0.0/16"
	rancherOCNEEngineConfig.PrivateRegistry = ""
	rancherOCNEEngineConfig.ProxyEndpoint = ""
	rancherOCNEEngineConfig.Region = region
	rancherOCNEEngineConfig.SkipOcneInstall = false
	rancherOCNEEngineConfig.TigeraImageTag = "v1.29.0"
	rancherOCNEEngineConfig.UseNodePvEncryption = true
	rancherOCNEEngineConfig.VcnID = vcnID
	rancherOCNEEngineConfig.VerrazzanoResource = "apiVersion: install.verrazzano.io/v1beta1\nkind: Verrazzano\nmetadata:\n  name: managed\n  namespace: default\nspec:\n  profile: managed-cluster"
	rancherOCNEEngineConfig.VerrazzanoTag = "v1.6.0-20230609132620-44e8f4d1"
	rancherOCNEEngineConfig.VerrazzanoVersion = "1.6.0-4574+44e8f4d1"
	rancherOCNEEngineConfig.WorkerNodeSubnet = workerNodeSubnet
	rancherOCNEEngineConfig.Type = "ociocneEngineConfig"
	rancherOCNEEngineConfig.ClusterName = ""
	rancherOCNEEngineConfig.NodeShape = "VM.Standard.E4.Flex"
	rancherOCNEEngineConfig.NumWorkerNodes = 1
	rancherOCNEEngineConfig.NodePools = []interface{}{}
	rancherOCNEEngineConfig.ApplyYamls = []interface{}{}

	var rancherOCNEClusterConfig RancherOCNECluster
	rancherOCNEClusterConfig.DockerRootDir = "/var/lib/docker"
	rancherOCNEClusterConfig.EnableClusterAlerting = false
	rancherOCNEClusterConfig.EnableClusterMonitoring = false
	rancherOCNEClusterConfig.EnableNetworkPolicy = false
	rancherOCNEClusterConfig.WindowsPreferedCluster = false
	rancherOCNEClusterConfig.Type = "cluster"
	rancherOCNEClusterConfig.Name = clusterName
	rancherOCNEClusterConfig.OciocneEngineConfig = rancherOCNEEngineConfig
	rancherOCNEClusterConfig.CloudCredentialID = cloudCredentialID
	rancherOCNEClusterConfig.Labels = struct{}{}

	return createCluster(clusterName, rancherOCNEClusterConfig)
}

// Creates a OCNE Cluster with node pools through CAPI
func createNodePoolCluster(clusterName, nodePoolName string) error {
	nodePublicKeyContents, err := getFileContents(nodePublicKeyPath)
	if err != nil {
		t.Logs.Infof("error reading node public key file: %v", err)
		return err
	}
	nodePoolSpec := fmt.Sprintf(
		"{\"name\":\"%s\",\"replicas\":1,\"memory\":32,\"ocpus\":2,\"volumeSize\":100,\"shape\":\"VM.Standard.E4.Flex\"}",
		nodePoolName)

	// Fill in the values for the create cluster API request body
	var rancherOCNEEngineConfig RancherOCIOCNEEngine
	rancherOCNEEngineConfig.CalicoImagePath = "olcne"
	rancherOCNEEngineConfig.CloudCredentialID = cloudCredentialID
	rancherOCNEEngineConfig.ClusterCidr = "10.96.0.0/16"
	rancherOCNEEngineConfig.CompartmentID = compartmentID
	rancherOCNEEngineConfig.ControlPlaneMemoryGbs = 16
	rancherOCNEEngineConfig.ControlPlaneOcpus = 2
	rancherOCNEEngineConfig.ControlPlaneShape = "VM.Standard.E4.Flex"
	rancherOCNEEngineConfig.ControlPlaneSubnet = controlPlaneSubnet
	rancherOCNEEngineConfig.ControlPlaneVolumeGbs = 100
	rancherOCNEEngineConfig.CorednsImageTag = "v1.9.3"
	rancherOCNEEngineConfig.DisplayName = clusterName
	rancherOCNEEngineConfig.DriverName = "ociocneengine"
	rancherOCNEEngineConfig.EtcdImageTag = "3.5.6"
	rancherOCNEEngineConfig.ImageDisplayName = "Oracle-Linux-8.7-2023.05.24-0"
	rancherOCNEEngineConfig.ImageID = ""
	rancherOCNEEngineConfig.InstallCalico = true
	rancherOCNEEngineConfig.InstallCcm = true
	rancherOCNEEngineConfig.InstallVerrazzano = false
	rancherOCNEEngineConfig.KubernetesVersion = "v1.25.7"
	rancherOCNEEngineConfig.LoadBalancerSubnet = loadBalancerSubnet
	rancherOCNEEngineConfig.Name = ""
	rancherOCNEEngineConfig.NodePublicKeyContents = nodePublicKeyContents
	rancherOCNEEngineConfig.NumControlPlaneNodes = 1
	rancherOCNEEngineConfig.OcneVersion = "1.6"
	rancherOCNEEngineConfig.PodCidr = "10.244.0.0/16"
	rancherOCNEEngineConfig.PrivateRegistry = ""
	rancherOCNEEngineConfig.ProxyEndpoint = ""
	rancherOCNEEngineConfig.Region = region
	rancherOCNEEngineConfig.SkipOcneInstall = false
	rancherOCNEEngineConfig.TigeraImageTag = "v1.29.0"
	rancherOCNEEngineConfig.UseNodePvEncryption = true
	rancherOCNEEngineConfig.VcnID = vcnID
	rancherOCNEEngineConfig.VerrazzanoResource = "apiVersion: install.verrazzano.io/v1beta1\nkind: Verrazzano\nmetadata:\n  name: managed\n  namespace: default\nspec:\n  profile: managed-cluster"
	rancherOCNEEngineConfig.VerrazzanoTag = "v1.6.0-20230609132620-44e8f4d1"
	rancherOCNEEngineConfig.VerrazzanoVersion = "1.6.0-4574+44e8f4d1"
	rancherOCNEEngineConfig.WorkerNodeSubnet = workerNodeSubnet
	rancherOCNEEngineConfig.Type = "ociocneEngineConfig"
	rancherOCNEEngineConfig.ClusterName = ""
	rancherOCNEEngineConfig.NodeShape = "VM.Standard.E4.Flex"
	rancherOCNEEngineConfig.NumWorkerNodes = 1
	rancherOCNEEngineConfig.NodePools = []interface{}{nodePoolSpec}
	rancherOCNEEngineConfig.ApplyYamls = []interface{}{}

	var rancherOCNEClusterConfig RancherOCNECluster
	rancherOCNEClusterConfig.DockerRootDir = "/var/lib/docker"
	rancherOCNEClusterConfig.EnableClusterAlerting = false
	rancherOCNEClusterConfig.EnableClusterMonitoring = false
	rancherOCNEClusterConfig.EnableNetworkPolicy = false
	rancherOCNEClusterConfig.WindowsPreferedCluster = false
	rancherOCNEClusterConfig.Type = "cluster"
	rancherOCNEClusterConfig.Name = clusterName
	rancherOCNEClusterConfig.OciocneEngineConfig = rancherOCNEEngineConfig
	rancherOCNEClusterConfig.CloudCredentialID = cloudCredentialID
	rancherOCNEClusterConfig.Labels = struct{}{}

	return createCluster(clusterName, rancherOCNEClusterConfig)
}

// Creates an OCNE cluster through ClusterAPI by making a request to the Rancher API
func createCluster(clusterName string, requestPayload RancherOCNECluster) error {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cluster?_replace=true")
	clusterBData, err := json.Marshal(requestPayload)
	if err != nil {
		t.Logs.Errorf("json marshalling error: %v", zap.Error(err))
		return err
	}
	res, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusCreated, clusterBData, t.Logs)
	if res != nil {
		t.Logs.Infof("create cluster response body: %s", res.String())
	}
	if err != nil {
		t.Logs.Errorf("error while retrieving http data: %v", zap.Error(err))
		return err
	}
	return nil
}

// Deletes the OCNE cluster by sending a DELETE request to the Rancher API
func deleteCluster(clusterName string) error {
	clusterID, err := getClusterIDFromName(clusterName)
	if err != nil {
		t.Logs.Infof("could not fetch cluster ID from cluster name %s: %s", clusterName, err)
		return err
	}
	t.Logs.Infof("clusterID for deletion: %s", clusterID)

	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("%s/%s", "v1/provisioning.cattle.io.clusters/fleet-default", clusterID))

	_, err = helpers.HTTPHelper(httpClient, "DELETE", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
	if err != nil {
		t.Logs.Errorf("error while deleting cluster: %v", err)
		return err
	}
	return nil
}

// Returns true if the cluster currently exists and is Active
func isClusterActive(clusterName string) (bool, error) {
	jsonBody, err := getCluster(clusterName)
	if err != nil {
		return false, err
	}

	state := fmt.Sprint(jsonBody.Path("data.0.state").Data())
	t.Logs.Infof("State: %s", state)
	return state == "active", nil
}

// Returns true if the OCNE cluster is deleted/does not exist
func isClusterDeleted(clusterName string) (bool, error) {
	jsonBody, err := getCluster(clusterName)
	if err != nil {
		return false, err
	}
	state := fmt.Sprint(jsonBody.Path("data.0.state").Data())
	t.Logs.Infof("deleting cluster %s state: %s", clusterName, state)

	data := fmt.Sprint(jsonBody.Path("data").Data())
	return data == "[]", nil
}

// Asserts whether the cluster was created as expected
func verifyCluster(clusterName string, numberNodes int) error {
	// Check if the cluster looks good from the Rancher API
	var err error
	if err = verifyGetRequest(clusterName, numberNodes); err != nil {
		return err
	}

	// Get the kubeconfig of the cluster to look inside
	clusterID, err := getClusterIDFromName(clusterName)
	if err != nil {
		t.Logs.Errorf("could not get cluster ID for cluster %s", clusterName)
		return err
	}
	workloadKubeconfigPath, err := writeWorkloadKubeconfig(clusterID)
	if err != nil {
		t.Logs.Errorf("could not get kubeconfig for cluster %s", clusterName)
		return err
	}

	// Check if the cluster has the expected nodes and pods running
	if err = verifyClusterNodes(clusterName, workloadKubeconfigPath, numberNodes); err != nil {
		return err
	}
	return verifyClusterPods(clusterName, workloadKubeconfigPath)
}

// Verifies that a GET request to the Rancher API for this cluster returns expected values
func verifyGetRequest(clusterName string, numberNodes int) error {
	jsonBody, err := getCluster(clusterName)
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
		Expect(a.actual).To(Equal(a.expected), "cluster %s has a %s value of %v but should be %v",
			clusterName, a.name, a.actual, a.expected)
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
		Expect(n.value).ToNot(BeNil(), "cluster %s has a non-nil value for %s", clusterName, n.name)
	}

	return nil
}

// Verifies that the workload cluster has the expected nodes
func verifyClusterNodes(clusterName, kubeconfigPath string, expectedNumberNodes int) error {
	numNodes, err := pkg.GetNodeCountInCluster(kubeconfigPath)
	if err != nil {
		t.Logs.Errorf("could not verify number of nodes in cluster %s: %s", clusterName, err)
		return err
	}
	if numNodes != expectedNumberNodes {
		err = fmt.Errorf("expected %v nodes in cluster %s but got %v", expectedNumberNodes, clusterName, numNodes)
		t.Logs.Error(err)
		return err
	}
	return nil
}

// Verifies that all expected pods in the workload cluster are active,
// given the cluster's kubeconfig path and the cluster name
func verifyClusterPods(clusterName, kubeconfigPath string) error {
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
			t.Logs.Errorf("error while verifying running pods: %s", err)
			return err
		}
		if !podsRunning {
			err = fmt.Errorf("there are missing pods in the %s namespace", namespace)
			t.Logs.Error(err)
			return err
		}
	}

	t.Logs.Infof("all expected pods in cluster %s are running", clusterName)
	return nil
}

// Gets a specified cluster by using the Rancher REST API
func getCluster(clusterName string) (*gabs.Container, error) {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cluster?name="+clusterName)
	return helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
}

// Returns the cluster ID corresponding the given name
func getClusterIDFromName(clusterName string) (string, error) {
	jsonBody, err := getCluster(clusterName)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(jsonBody.Path("data.0.id").Data()), nil
}

// Generates the kubeconfig of the workload cluster with an ID of `clusterID`
// Writes the kubeconfig to a file inside /tmp. Returns the path of the kubeconfig file.
func writeWorkloadKubeconfig(clusterID string) (string, error) {
	t.Logs.Info("+++ Downloading kubeconfig from Rancher +++")
	outputPath := fmt.Sprintf("/tmp/%s-kubeconfig", clusterID)
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("v3/clusters/%s?action=generateKubeconfig", clusterID))
	jsonBody, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
	if err != nil {
		t.Logs.Errorf("error while retrieving http data: %v", zap.Error(err))
		return "", err
	}

	workloadKubeconfig := fmt.Sprint(jsonBody.Path("config").Data())
	err = os.WriteFile(outputPath, []byte(workloadKubeconfig), 0600)
	if err != nil {
		t.Logs.Errorf("error writing workload cluster kubeconfig to a file: %v", zap.Error(err))
		return "", err
	}
	return outputPath, nil
}

// Given a file path, returns the file's contents as a string
func getFileContents(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		t.Logs.Errorf("failed reading file contents: %v", err)
		return "", err
	}
	return string(data), nil
}

// Given the base Rancher URL and the additional URL path,
// returns the full URL and a refreshed Bearer token
func setupRequest(rancherBaseURL, urlPath string) (string, string) {
	adminToken := helpers.GetRancherLoginToken(t.Logs)
	t.Logs.Infof("adminToken: %s", adminToken)
	requestURL := fmt.Sprintf("%s/%s", rancherBaseURL, urlPath)
	t.Logs.Infof("requestURL: %s", requestURL)
	return requestURL, adminToken
}
