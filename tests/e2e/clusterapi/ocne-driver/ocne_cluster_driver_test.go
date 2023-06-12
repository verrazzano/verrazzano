// Copyright (c) 2023, Oracle and/or its affiliates.
// Licensed under the Universal Permissive License v 1.0 as shown at https://oss.oracle.com/licenses/upl.

package ocnedriver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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
	shortWaitTimeout     = 5 * time.Minute
	shortPollingInterval = 10 * time.Second
	waitTimeout          = 120 * time.Minute
	pollingInterval      = 30 * time.Second
	clusterName          = "strudel2"
)

var (
	t                 = framework.NewTestFramework("capi-ocne-driver")
	httpClient        *retryablehttp.Client
	rancherURL        string
	cloudCredentialID string
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

var beforeSuite = t.BeforeSuiteFunc(func() {
	//TODO oci get to check it's working

	kubeconfigPath, err := k8sutil.GetKubeConfigLocation()
	Expect(err).ShouldNot(HaveOccurred())
	if !pkg.IsRancherEnabled(kubeconfigPath) || !pkg.IsClusterAPIEnabled(kubeconfigPath) {
		Skip("Skipping ocne cluster driver test suite since either of rancher and capi components are not enabled")
	}

	httpClient, err = pkg.GetVerrazzanoHTTPClient(kubeconfigPath)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed getting http client: %v", err))
	}

	rancherURL, err = helpers.GetRancherURL(t.Logs)
	if err != nil {
		AbortSuite(fmt.Sprintf("Failed getting rancherURL: %v", err))
	}
	t.Logs.Infof("rancherURL: %s", rancherURL)

	Eventually(func() error {
		cloudCredentialID, err = createCloudCredential("testing-creds")
		return err
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())

	Eventually(func() error {
		return validateCloudCredential(cloudCredentialID)
	}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
})
var _ = BeforeSuite(beforeSuite)

var afterSuite = t.AfterSuiteFunc(func() {
	// Delete the credential
	deleteCredential(cloudCredentialID)

	// Verify the credential is deleted
	Eventually(func() (bool, error) { return isCredentialDeleted(cloudCredentialID) }, waitTimeout, pollingInterval).Should(
		BeTrue(), fmt.Sprintf("Cloud credential %s is not deleted", cloudCredentialID))
})
var _ = AfterSuite(afterSuite)

var _ = t.Describe("OCNE Cluster Driver", Label("f:rancher-capi:ocne-cluster-driver"), Serial, func() {
	t.Context("OCNE cluster creation", func() {
		t.It("create OCNE cluster", func() {
			// Create the cluster
			Eventually(func() error {
				return createCluster(clusterName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("Check OCNE cluster is active", func() {
			// Verify the cluster is active
			Eventually(func() (bool, error) { return isClusterActive(clusterName) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("Cluster %s is not active", clusterName))
		})
	})

	t.Context("OCNE cluster delete", func() {
		t.It("delete OCNE cluster", func() {
			// Delete the cluster
			Eventually(func() error {
				return deleteCluster(clusterName)
			}, shortWaitTimeout, shortPollingInterval).Should(BeNil())
		})

		t.It("Check OCNE cluster is deleted", func() {
			// Verify the cluster is deleted
			Eventually(func() (bool, error) { return isClusterDeleted(clusterName) }, waitTimeout, pollingInterval).Should(
				BeTrue(), fmt.Sprintf("Cluster %s is not deleted", clusterName))
		})
	})
})

func deleteCredential(credID string) {
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("%s%s", "v3/cloudCredentials/", credID))
	helpers.HTTPHelper(httpClient, "DELETE", requestURL, adminToken, "Bearer", http.StatusNoContent, nil, t.Logs)
}

func isCredentialDeleted(credID string) (bool, error) {
	jsonBody, err := getCredential(credID)
	if err != nil {
		return false, err
	}
	jsonData := fmt.Sprint(jsonBody.Path("data").Data())
	fmt.Println("Delete credential jsonData: " + jsonData)
	return jsonData == "[]", nil
}

func getCredential(credID string) (*gabs.Container, error) {
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("%s%s", "v3/cloudcredentials?id=", credID))
	return helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
}

func deleteCluster(clusterName string) error {
	clusterID, err := getClusterIDFromName(clusterName)
	if err != nil {
		t.Logs.Infof("Could not fetch cluster ID from cluster name %s: %s", clusterName, err)
		return err
	}
	t.Logs.Infof("clusterID for deletion: %s", clusterID)

	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("%s/%s", "v1/provisioning.cattle.io.clusters/fleet-default", clusterID))

	_, err = helpers.HTTPHelper(httpClient, "DELETE", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
	if err != nil {
		t.Logs.Errorf("Error while deleting cluster: %v", err)
		return err
	}
	return nil
}

func isClusterDeleted(clusterName string) (bool, error) {
	jsonBody, err := getCluster(clusterName)
	if err != nil {
		return false, err
	}
	jsonData := fmt.Sprint(jsonBody.Path("data").Data())
	return jsonData == "[]", nil
}

func createCloudCredential(credentialName string) (string, error) {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cloudcredentials")
	privateKeyContents, err := getFileContents(privateKeyPath)
	if err != nil {
		t.Logs.Infof("error reading private key file: %v", err)
		return "", err
	}

	var cloudCreds RancherCloudCred
	cloudCreds.Name = "strudel-test"
	cloudCreds.InternalName = cloudCreds.Name
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
		t.Logs.Errorf("Error while retrieving http data: %v", zap.Error(err))
		return "", err
	}
	credID := fmt.Sprint(jsonBody.Path("id").Data())
	return credID, nil
}

// Creates an OCNE cluster through ClusterAPI
func createCluster(clusterName string) error {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cluster")
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

	clusterBData, err := json.Marshal(rancherOCNEClusterConfig)
	if err != nil {
		t.Logs.Errorf("json marshalling error: %v", zap.Error(err))
		return err
	}
	res, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusCreated, clusterBData, t.Logs)
	if res != nil {
		t.Logs.Infof("create cluster response body: %s", res.String())
	}
	if err != nil {
		t.Logs.Errorf("Error while retrieving http data: %v", zap.Error(err))
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

	var cmd helpers.BashCommand
	var cmdArgs []string
	cmdArgs = append(cmdArgs, "kubectl", "get", "clusters.cluster.x-k8s.io", "-A")
	cmd.CommandArgs = cmdArgs
	response := helpers.Runner(&cmd, t.Logs)
	t.Logs.Infof("+++ All CAPI clusters =  %s +++", (&response.StandardOut).String())

	cmdArgs = []string{}
	cmdArgs = append(cmdArgs, "kubectl", "get", "clusters.management.cattle.io")
	cmd.CommandArgs = cmdArgs
	response = helpers.Runner(&cmd, t.Logs)
	t.Logs.Infof("+++ All management clusters =  %s +++", (&response.StandardOut).String())

	cmdArgs = []string{}
	cmdArgs = append(cmdArgs, "kubectl", "get", "clusters.provisioning.cattle.io", "-A")
	cmd.CommandArgs = cmdArgs
	response = helpers.Runner(&cmd, t.Logs)
	t.Logs.Infof("+++ All provisioning clusters =  %s +++", (&response.StandardOut).String())

	cmdArgs = []string{}
	cmdArgs = append(cmdArgs, "kubectl", "get", "ma", "-A")
	cmd.CommandArgs = cmdArgs
	response = helpers.Runner(&cmd, t.Logs)
	t.Logs.Infof("+++ All CAPI machines =  %s +++", (&response.StandardOut).String())

	clusterID, err := getClusterIDFromName(clusterName)
	if err != nil {
		t.Logs.Errorf("Could not fetch cluster ID from cluster name %s: %s", clusterName, err)
	}

	kubeconfigPath, err := writeWorkloadKubeconfig(clusterID)
	if err != nil {
		t.Logs.Error("could not download kubeconfig from rancher")
	}

	cmdArgs = []string{}
	cmdArgs = append(cmdArgs, "kubectl", "--kubeconfig", kubeconfigPath, "get", "nodes", "-o", "wide")
	cmd.CommandArgs = cmdArgs
	response = helpers.Runner(&cmd, t.Logs)
	t.Logs.Infof("+++ All nodes in workload cluster =  %s +++", (&response.StandardOut).String())

	cmdArgs = []string{}
	cmdArgs = append(cmdArgs, "kubectl", "--kubeconfig", kubeconfigPath, "get", "pod", "-A", "-o", "wide")
	cmd.CommandArgs = cmdArgs
	response = helpers.Runner(&cmd, t.Logs)
	t.Logs.Infof("+++ All pods in workload cluster =  %s +++", (&response.StandardOut).String())

	t.Logs.Infof("Check cluster is active jsonBody: %s", jsonBody.String())
	state := fmt.Sprint(jsonBody.Path("data.0.state").Data())
	t.Logs.Infof("State: %s", state)
	return state == "active", nil
}

// Generates the kubeconfig of the workload cluster with an ID of `clusterID`
// Writes the kubeconfig to a file inside /tmp. Returns the path of the kubeconfig file.
func writeWorkloadKubeconfig(clusterID string) (string, error) {
	t.Logs.Info("+++ Downloading kubeconfig from Rancher +++")
	outputPath := fmt.Sprintf("/tmp/%s-kubeconfig", clusterID)
	requestURL, adminToken := setupRequest(rancherURL, fmt.Sprintf("v3/clusters/%s?action=generateKubeconfig", clusterID))
	jsonBody, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
	if err != nil {
		t.Logs.Errorf("Error while retrieving http data: %v", zap.Error(err))
		return "", err
	}

	workloadKubeconfig := fmt.Sprint(jsonBody.Path("config").Data())
	err = os.WriteFile(outputPath, []byte(workloadKubeconfig), 0600)
	if err != nil {
		t.Logs.Errorf("Error writing workload cluster kubeconfig to a file: %v", zap.Error(err))
		return "", err
	}
	return outputPath, nil
}

func getClusterIDFromName(clusterName string) (string, error) {
	jsonBody, err := getCluster(clusterName)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(jsonBody.Path("data.0.id").Data()), nil
}

// Gets a specified cluster by using the Rancher REST API
func getCluster(clusterName string) (*gabs.Container, error) {
	requestURL, adminToken := setupRequest(rancherURL, "v3/cluster?name="+clusterName)
	return helpers.HTTPHelper(httpClient, "GET", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
}

// Sends a test request to check that the cloud credential is confiugred properly
func validateCloudCredential(credID string) error {
	urlPath := fmt.Sprintf("meta/oci/nodeImages?cloudCredentialId=%s&compartment=%s&region=%s", credID, compartmentID, region)
	requestURL, adminToken := setupRequest(rancherURL, urlPath)
	t.Logs.Infof("validateCloudCredential URL = %s", requestURL)
	res, err := helpers.HTTPHelper(httpClient, "POST", requestURL, adminToken, "Bearer", http.StatusOK, nil, t.Logs)
	if err != nil {
		t.Logs.Errorf("Error while retrieving http data: %v", zap.Error(err))
		return err
	}
	t.Logs.Infof("Validate cloud credential response: %s", fmt.Sprint(res))
	return nil
}

func getFileContents(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		t.Logs.Errorf("failed reading file contents: %v", err)
		return "", err
	}
	return string(data), nil
}

func setupRequest(rancherBaseURL, urlPath string) (string, string) {
	adminToken := helpers.GetRancherLoginToken(t.Logs)
	t.Logs.Infof("adminToken: %s", adminToken)
	requestURL := fmt.Sprintf("%s/%s", rancherBaseURL, urlPath)
	t.Logs.Infof("requestURL: %s", requestURL)
	return requestURL, adminToken
}
